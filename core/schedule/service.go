package schedule

import (
	"errors"
	"sync"
	"time"

	"github.com/goastian/astiango-hub/core/interfaces"
	"github.com/goastian/astiango-hub/core/models/models"
	"github.com/goastian/astiango-hub/core/models/service"
	"github.com/goastian/astiango-hub/core/spider/admin"
	"github.com/goastian/astiango-hub/core/utils"
	"github.com/robfig/cron/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Service struct {
	// dependencies
	modelSvc *service.ModelService[models.Schedule]
	adminSvc *admin.Service

	// settings variables
	loc            *time.Location
	updateInterval time.Duration

	// internals
	cron      *cron.Cron
	logger    cron.Logger
	schedules []models.Schedule
	stopped   bool
	mu        sync.Mutex
	interfaces.Logger
}

func (svc *Service) GetUpdateInterval() (interval time.Duration) {
	return svc.updateInterval
}

func (svc *Service) SetUpdateInterval(interval time.Duration) {
	svc.updateInterval = interval
}

func (svc *Service) Init() (err error) {
	// Validate dependencies
	if svc.modelSvc == nil {
		return errors.New("model service is not initialized")
	}
	if svc.adminSvc == nil {
		return errors.New("admin service is not initialized")
	}
	if svc.cron == nil {
		return errors.New("cron service is not initialized")
	}

	// Fetch and validate existing schedules
	err = svc.fetch()
	if err != nil {
		svc.Fatalf("failed to initialize schedule service: %v", err)
		return err
	}

	// Validate and enable existing schedules
	for _, s := range svc.schedules {
		if s.Enabled {
			// Validate cron expression
			if _, err := cron.ParseStandard(s.Cron); err != nil {
				svc.Errorf("invalid cron expression for schedule %s: %v", s.Id.Hex(), err)
				// Disable invalid schedules
				if disableErr := svc.Disable(s, s.GetUpdatedBy()); disableErr != nil {
					svc.Errorf("failed to disable invalid schedule %s: %v", s.Id.Hex(), disableErr)
				}
				continue
			}

			// Add to cron
			if err := svc.Enable(s, s.GetUpdatedBy()); err != nil {
				svc.Errorf("failed to enable schedule %s during initialization: %v", s.Id.Hex(), err)
			}
		}
	}

	svc.Infof("initialized schedule service with %d enabled schedules", len(svc.schedules))
	return nil
}

func (svc *Service) Start() {
	svc.Infof("starting schedule service")
	svc.cron.Start()
	go svc.Update()
	svc.Infof("schedule service started successfully")
}

func (svc *Service) Wait() {
	utils.DefaultWait()
	svc.Stop()
}

func (svc *Service) Stop() {
	svc.stopped = true
	svc.cron.Stop()
}

func (svc *Service) Enable(s models.Schedule, by primitive.ObjectID) (err error) {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	// Validate cron expression
	if _, err := cron.ParseStandard(s.Cron); err != nil {
		return errors.New("invalid cron expression: " + err.Error())
	}

	id, err := svc.cron.AddFunc(s.Cron, svc.schedule(s.Id))
	if err != nil {
		svc.Errorf("failed to add cron job: %v", err)
		return err
	}
	s.Enabled = true
	s.EntryId = id
	s.SetUpdated(by)
	return svc.modelSvc.ReplaceById(s.Id, s)
}

func (svc *Service) Disable(s models.Schedule, by primitive.ObjectID) (err error) {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	// Store the current entry ID before modifying
	entryId := s.EntryId

	// First update the database
	s.Enabled = false
	s.EntryId = -1
	s.SetUpdated(by)
	if err := svc.modelSvc.ReplaceById(s.Id, s); err != nil {
		return err
	}

	// Only remove from cron after successful database update using the stored entry ID
	if entryId != -1 {
		svc.cron.Remove(entryId)
	}
	return nil
}

func (svc *Service) Update() {
	for {
		if svc.stopped {
			return
		}

		svc.update()

		time.Sleep(svc.updateInterval)
	}
}

func (svc *Service) GetCron() (c *cron.Cron) {
	return svc.cron
}

// GetScheduleCount returns the number of enabled schedules
func (svc *Service) GetScheduleCount() int {
	return len(svc.schedules)
}

// GetCronEntryCount returns the number of active cron entries
func (svc *Service) GetCronEntryCount() int {
	return len(svc.cron.Entries())
}

// IsHealthy performs a health check on the schedule service
func (svc *Service) IsHealthy() bool {
	// Check if cron is running
	if svc.cron == nil {
		return false
	}

	// Check if service is not stopped
	if svc.stopped {
		return false
	}

	// Check if we can fetch schedules from database
	if err := svc.fetch(); err != nil {
		svc.Errorf("health check failed: cannot fetch schedules: %v", err)
		return false
	}

	return true
}

// GetHealthStatus returns detailed health information
func (svc *Service) GetHealthStatus() map[string]interface{} {
	status := map[string]interface{}{
		"healthy":          svc.IsHealthy(),
		"stopped":          svc.stopped,
		"schedule_count":   svc.GetScheduleCount(),
		"cron_entry_count": svc.GetCronEntryCount(),
		"update_interval":  svc.updateInterval.String(),
		"location":         svc.loc.String(),
	}

	// Add cron entries info
	entries := make([]map[string]interface{}, 0)
	for _, entry := range svc.cron.Entries() {
		entries = append(entries, map[string]interface{}{
			"id":   entry.ID,
			"next": entry.Next.Format(time.RFC3339),
			"prev": entry.Prev.Format(time.RFC3339),
		})
	}
	status["cron_entries"] = entries

	return status
}

func (svc *Service) update() {
	// Add recovery mechanism
	defer func() {
		if r := recover(); r != nil {
			svc.Errorf("panic in schedule update: %v", r)
		}
	}()

	// fetch enabled schedules
	if err := svc.fetch(); err != nil {
		svc.Errorf("failed to fetch schedules: %v", err)
		return
	}

	// entry id map
	entryIdsMap := svc.getEntryIdsMap()

	// iterate enabled schedules
	for _, s := range svc.schedules {
		_, ok := entryIdsMap[s.EntryId]
		if !ok {
			// Schedule is enabled but not in cron, add it
			if s.Enabled {
				// Add retry mechanism for enabling schedules
				err := svc.enableWithRetry(s, s.GetCreatedBy(), 3)
				if err != nil {
					svc.Errorf("failed to enable schedule after retries: %v", err)
					continue
				}
			}
		} else {
			// Mark as found
			entryIdsMap[s.EntryId] = true
		}
	}

	// remove non-existent entries
	for id, ok := range entryIdsMap {
		if !ok {
			svc.cron.Remove(id)
		}
	}
}

// enableWithRetry attempts to enable a schedule with retry logic
func (svc *Service) enableWithRetry(s models.Schedule, by primitive.ObjectID, maxRetries int) error {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		if err := svc.Enable(s, by); err != nil {
			lastErr = err
			svc.Warnf("failed to enable schedule (attempt %d/%d): %v", i+1, maxRetries, err)
			time.Sleep(time.Duration(i+1) * time.Second) // exponential backoff
			continue
		}
		return nil
	}
	return lastErr
}

func (svc *Service) getEntryIdsMap() (res map[cron.EntryID]bool) {
	res = map[cron.EntryID]bool{}
	for _, e := range svc.cron.Entries() {
		res[e.ID] = false
	}
	return res
}

func (svc *Service) fetch() (err error) {
	query := bson.M{
		"enabled": true,
	}
	svc.schedules, err = svc.modelSvc.GetMany(query, nil)
	if err != nil {
		return err
	}
	return nil
}

func (svc *Service) schedule(id primitive.ObjectID) (fn func()) {
	return func() {
		// Add recovery mechanism for individual schedule executions
		defer func() {
			if r := recover(); r != nil {
				svc.Errorf("panic in schedule execution for %s: %v", id.Hex(), r)
			}
		}()

		// Add execution logging
		svc.Infof("executing schedule: %s", id.Hex())
		startTime := time.Now()

		// schedule
		s, err := svc.modelSvc.GetById(id)
		if err != nil {
			svc.Errorf("failed to get schedule: %v", err)
			return
		}

		// Verify schedule still exists and is enabled
		if s == nil || !s.Enabled {
			svc.Warnf("schedule %s no longer exists or is disabled, removing from cron", id.Hex())
			// Use a goroutine to avoid potential deadlock when removing from within cron execution
			go func() {
				svc.mu.Lock()
				defer svc.mu.Unlock()
				svc.cron.Remove(s.EntryId)
			}()
			return
		}

		// spider
		spider, err := service.NewModelService[models.Spider]().GetById(s.SpiderId)
		if err != nil {
			svc.Errorf("failed to get spider: %v", err)
			return
		}

		// Verify spider still exists
		if spider == nil {
			svc.Errorf("spider %s no longer exists", s.SpiderId.Hex())
			return
		}

		// options
		opts := &interfaces.SpiderRunOptions{
			Mode:       s.Mode,
			NodeIds:    s.NodeIds,
			Cmd:        s.Cmd,
			Param:      s.Param,
			Priority:   s.Priority,
			ScheduleId: s.Id,
			UserId:     s.GetCreatedBy(),
		}

		// normalize options
		if opts.Mode == "" {
			opts.Mode = spider.Mode
		}
		if len(opts.NodeIds) == 0 {
			opts.NodeIds = spider.NodeIds
		}
		if opts.Cmd == "" {
			opts.Cmd = spider.Cmd
		}
		if opts.Param == "" {
			opts.Param = spider.Param
		}
		if opts.Priority == 0 {
			if spider.Priority > 0 {
				opts.Priority = spider.Priority
			} else {
				opts.Priority = 5
			}
		}

		// schedule or assign a task in the task queue
		taskIds, err := svc.adminSvc.Schedule(s.SpiderId, opts)
		if err != nil {
			svc.Errorf("failed to schedule spider: %v", err)
			return
		}

		// Log successful execution
		duration := time.Since(startTime)
		svc.Infof("successfully executed schedule %s, created %d tasks in %v", id.Hex(), len(taskIds), duration)
	}
}

func newScheduleService() *Service {
	// service
	svc := &Service{
		loc:            time.Local,
		updateInterval: 1 * time.Minute,
		adminSvc:       admin.GetSpiderAdminService(),
		modelSvc:       service.NewModelService[models.Schedule](),
		Logger:         utils.NewLogger("ScheduleService"),
	}

	// logger
	svc.logger = NewCronLogger()

	// cron
	svc.cron = cron.New(
		cron.WithLogger(svc.logger),
		cron.WithLocation(svc.loc),
		cron.WithChain(cron.Recover(svc.logger)),
	)

	// initialize
	if err := svc.Init(); err != nil {
		panic(err)
	}

	return svc
}

var _service *Service
var _serviceOnce sync.Once

func GetScheduleService() *Service {
	_serviceOnce.Do(func() {
		_service = newScheduleService()
	})
	return _service
}
