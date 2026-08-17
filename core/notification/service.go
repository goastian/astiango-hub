package notification

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/goastian/astiango-hub/core/interfaces"
	"github.com/goastian/astiango-hub/core/utils"

	"github.com/goastian/astiango-hub/core/constants"
	"github.com/goastian/astiango-hub/core/entity"
	"github.com/goastian/astiango-hub/core/models/models"
	"github.com/goastian/astiango-hub/core/models/service"
	"github.com/gomarkdown/markdown"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Service struct {
	interfaces.Logger
}

func (svc *Service) Send(s *models.NotificationSetting, args ...any) {
	title := s.Title

	// Use bounded goroutine pool to prevent unlimited goroutine creation
	const maxWorkers = 5
	wg := sync.WaitGroup{}
	semaphore := make(chan struct{}, maxWorkers)

	for _, chId := range s.ChannelIds {
		wg.Add(1)

		// Acquire semaphore
		semaphore <- struct{}{}

		go func(chId primitive.ObjectID) {
			defer func() {
				<-semaphore // Release semaphore
				wg.Done()
				if r := recover(); r != nil {
					svc.Errorf("[NotificationService] channel handler panic for %s: %v", chId.Hex(), r)
				}
			}()

			ch, err := service.NewModelService[models.NotificationChannel]().GetById(chId)
			if err != nil {
				svc.Errorf("[NotificationService] get channel error: %v", err)
				return
			}
			content := svc.getContent(s, ch, args...)
			switch ch.Type {
			case TypeMail:
				svc.SendMail(s, ch, title, content)
			case TypeIM:
				svc.SendIM(s, ch, title, content)
			}
		}(chId)
	}
	wg.Wait()
}

func (svc *Service) SendMail(s *models.NotificationSetting, ch *models.NotificationChannel, title, content string) {
	mailTo := s.MailTo
	mailCc := s.MailCc
	mailBcc := s.MailBcc

	// request
	r, _ := svc.createRequestMail(s, ch, title, content)

	// send mail
	err := SendMail(s, ch, mailTo, mailCc, mailBcc, title, content)
	if err != nil {
		svc.Errorf("[NotificationService] send mail error: %v", err)
	}

	// save request synchronously to avoid unbounded goroutines
	svc.saveRequest(r, err)
}

func (svc *Service) SendIM(s *models.NotificationSetting, ch *models.NotificationChannel, title, content string) {
	// request
	r, _ := svc.createRequestIM(s, ch, title, content, false)

	// send mobile notification
	err := SendIMNotification(ch, title, content)
	if err != nil {
		svc.Errorf("[NotificationService] send mobile notification error: %v", err)
	}

	// save request synchronously to avoid unbounded goroutines
	svc.saveRequest(r, err)
}

func (svc *Service) SendTestMessage(locale string, ch *models.NotificationChannel, toMail []string) (err error) {
	// Test message content
	var title, content string
	switch locale {
	case "zh":
		title = "测试通知"
		content = "这是来自 AstianGO Hub 的测试通知。如果您收到此消息，说明您的通知渠道配置正确。"
	default:
		title = "Test Notification"
		content = "This is a test notification from AstianGO Hub. If you receive this message, your notification channel is configured correctly."
	}

	// Notification request
	var r *models.NotificationRequest

	// Send test message based on channel type
	switch ch.Type {
	case TypeMail:
		// If toMail is nil, use the default email address
		if toMail == nil {
			toMail = []string{ch.SMTPUsername}
		}

		// Create request
		r, _ = svc.createRequestMailTest(nil, ch, title, content, toMail)

		// For email
		err = SendMail(nil, ch, toMail, nil, nil, title, content)
		if err != nil {
			svc.Errorf("failed to send test email: %v", err)
		}

	case TypeIM:
		// Create request
		r, _ = svc.createRequestIM(nil, ch, title, content, true)

		// For instant messaging
		err = SendIMNotification(ch, title, content)
		if err != nil {
			svc.Errorf("failed to send test IM notification: %v", err)
		}

	default:
		return fmt.Errorf("unsupported notification channel type: %s", ch.Type)
	}

	// Save request synchronously to avoid unbounded goroutines
	svc.saveRequest(r, err)

	return err
}

func (svc *Service) getContent(s *models.NotificationSetting, ch *models.NotificationChannel, args ...any) (content string) {
	vd := svc.getVariableData(args...)
	switch s.TemplateMode {
	case constants.NotificationTemplateModeMarkdown:
		variables := svc.parseTemplateVariables(s.TemplateMarkdown)
		content = svc.geContentWithVariables(s.TemplateMarkdown, variables, vd)
		if ch.Type == TypeMail {
			content = svc.convertMarkdownToHtml(content)
		}
		return content
	case constants.NotificationTemplateModeRichText:
		template := s.TemplateRichText
		if ch.Type == TypeIM {
			template = s.TemplateMarkdown
		}
		variables := svc.parseTemplateVariables(template)
		return svc.geContentWithVariables(template, variables, vd)
	}

	return content
}

func (svc *Service) geContentWithVariables(template string, variables []entity.NotificationVariable, vd VariableData) (content string) {
	content = template
	for _, v := range variables {
		switch v.Category {
		case "task":
			if vd.Task == nil {
				content = strings.ReplaceAll(content, v.GetKey(), "N/A")
				continue
			}
			switch v.Name {
			case "id":
				content = strings.ReplaceAll(content, v.GetKey(), vd.Task.Id.Hex())
			case "status":
				content = strings.ReplaceAll(content, v.GetKey(), vd.Task.Status)
			case "cmd":
				content = strings.ReplaceAll(content, v.GetKey(), vd.Task.Cmd)
			case "param":
				content = strings.ReplaceAll(content, v.GetKey(), vd.Task.Param)
			case "error":
				content = strings.ReplaceAll(content, v.GetKey(), vd.Task.Error)
			case "pid":
				content = strings.ReplaceAll(content, v.GetKey(), fmt.Sprintf("%d", vd.Task.Pid))
			case "mode":
				content = strings.ReplaceAll(content, v.GetKey(), vd.Task.Mode)
			case "priority":
				content = strings.ReplaceAll(content, v.GetKey(), fmt.Sprintf("%d", vd.Task.Priority))
			case "created_at":
				content = strings.ReplaceAll(content, v.GetKey(), svc.getFormattedTime(vd.Task.CreatedAt))
			case "created_by":
				content = strings.ReplaceAll(content, v.GetKey(), svc.getUsernameById(vd.Task.CreatedBy))
			case "updated_at":
				content = strings.ReplaceAll(content, v.GetKey(), svc.getFormattedTime(vd.Task.UpdatedAt))
			case "updated_by":
				content = strings.ReplaceAll(content, v.GetKey(), svc.getUsernameById(vd.Task.UpdatedBy))
			}

		case "task_stat":
			if vd.TaskStat == nil {
				content = strings.ReplaceAll(content, v.GetKey(), "N/A")
				continue
			}
			switch v.Name {
			case "started_at":
				content = strings.ReplaceAll(content, v.GetKey(), svc.getFormattedTime(vd.TaskStat.StartedAt))
			case "ended_at":
				content = strings.ReplaceAll(content, v.GetKey(), svc.getFormattedTime(vd.TaskStat.EndedAt))
			case "wait_duration":
				content = strings.ReplaceAll(content, v.GetKey(), fmt.Sprintf("%ds", vd.TaskStat.WaitDuration/1000))
			case "runtime_duration":
				content = strings.ReplaceAll(content, v.GetKey(), fmt.Sprintf("%ds", vd.TaskStat.RuntimeDuration/1000))
			case "total_duration":
				content = strings.ReplaceAll(content, v.GetKey(), fmt.Sprintf("%ds", vd.TaskStat.TotalDuration/1000))
			case "result_count":
				content = strings.ReplaceAll(content, v.GetKey(), fmt.Sprintf("%d", vd.TaskStat.ResultCount))
			}

		case "spider":
			if vd.Spider == nil {
				content = strings.ReplaceAll(content, v.GetKey(), "N/A")
				continue
			}
			switch v.Name {
			case "id":
				content = strings.ReplaceAll(content, v.GetKey(), vd.Spider.Id.Hex())
			case "name":
				content = strings.ReplaceAll(content, v.GetKey(), vd.Spider.Name)
			case "description":
				content = strings.ReplaceAll(content, v.GetKey(), vd.Spider.Description)
			case "mode":
				content = strings.ReplaceAll(content, v.GetKey(), vd.Spider.Mode)
			case "cmd":
				content = strings.ReplaceAll(content, v.GetKey(), vd.Spider.Cmd)
			case "param":
				content = strings.ReplaceAll(content, v.GetKey(), vd.Spider.Param)
			case "priority":
				content = strings.ReplaceAll(content, v.GetKey(), fmt.Sprintf("%d", vd.Spider.Priority))
			case "created_at":
				content = strings.ReplaceAll(content, v.GetKey(), svc.getFormattedTime(vd.Spider.CreatedAt))
			case "created_by":
				content = strings.ReplaceAll(content, v.GetKey(), svc.getUsernameById(vd.Spider.CreatedBy))
			case "updated_at":
				content = strings.ReplaceAll(content, v.GetKey(), svc.getFormattedTime(vd.Spider.UpdatedAt))
			case "updated_by":
				content = strings.ReplaceAll(content, v.GetKey(), svc.getUsernameById(vd.Spider.UpdatedBy))
			}

		case "node":
			if vd.Node == nil {
				content = strings.ReplaceAll(content, v.GetKey(), "N/A")
				continue
			}
			switch v.Name {
			case "id":
				content = strings.ReplaceAll(content, v.GetKey(), vd.Node.Id.Hex())
			case "key":
				content = strings.ReplaceAll(content, v.GetKey(), vd.Node.Key)
			case "name":
				content = strings.ReplaceAll(content, v.GetKey(), vd.Node.Name)
			case "is_master":
				content = strings.ReplaceAll(content, v.GetKey(), fmt.Sprintf("%t", vd.Node.IsMaster))
			case "ip":
				content = strings.ReplaceAll(content, v.GetKey(), vd.Node.Ip)
			case "mac":
				content = strings.ReplaceAll(content, v.GetKey(), vd.Node.Mac)
			case "hostname":
				content = strings.ReplaceAll(content, v.GetKey(), vd.Node.Hostname)
			case "description":
				content = strings.ReplaceAll(content, v.GetKey(), vd.Node.Description)
			case "status":
				content = strings.ReplaceAll(content, v.GetKey(), vd.Node.Status)
			case "enabled":
				content = strings.ReplaceAll(content, v.GetKey(), fmt.Sprintf("%t", vd.Node.Enabled))
			case "active":
				content = strings.ReplaceAll(content, v.GetKey(), fmt.Sprintf("%t", vd.Node.Active))
			case "active_at":
				content = strings.ReplaceAll(content, v.GetKey(), svc.getFormattedTime(vd.Node.ActiveAt))
			case "current_runners":
				content = strings.ReplaceAll(content, v.GetKey(), fmt.Sprintf("%d", vd.Node.CurrentRunners))
			case "max_runners":
				content = strings.ReplaceAll(content, v.GetKey(), fmt.Sprintf("%d", vd.Node.MaxRunners))
			case "created_at":
				content = strings.ReplaceAll(content, v.GetKey(), svc.getFormattedTime(vd.Node.CreatedAt))
			case "created_by":
				content = strings.ReplaceAll(content, v.GetKey(), svc.getUsernameById(vd.Node.CreatedBy))
			case "updated_at":
				content = strings.ReplaceAll(content, v.GetKey(), svc.getFormattedTime(vd.Node.UpdatedAt))
			case "updated_by":
				content = strings.ReplaceAll(content, v.GetKey(), svc.getUsernameById(vd.Node.UpdatedBy))
			}

		case "schedule":
			if vd.Schedule == nil {
				content = strings.ReplaceAll(content, v.GetKey(), "N/A")
				continue
			}
			switch v.Name {
			case "id":
				content = strings.ReplaceAll(content, v.GetKey(), vd.Schedule.Id.Hex())
			case "name":
				content = strings.ReplaceAll(content, v.GetKey(), vd.Schedule.Name)
			case "description":
				content = strings.ReplaceAll(content, v.GetKey(), vd.Schedule.Description)
			case "cron":
				content = strings.ReplaceAll(content, v.GetKey(), vd.Schedule.Cron)
			case "cmd":
				content = strings.ReplaceAll(content, v.GetKey(), vd.Schedule.Cmd)
			case "param":
				content = strings.ReplaceAll(content, v.GetKey(), vd.Schedule.Param)
			case "mode":
				content = strings.ReplaceAll(content, v.GetKey(), vd.Schedule.Mode)
			case "priority":
				content = strings.ReplaceAll(content, v.GetKey(), fmt.Sprintf("%d", vd.Schedule.Priority))
			case "enabled":
				content = strings.ReplaceAll(content, v.GetKey(), fmt.Sprintf("%t", vd.Schedule.Enabled))
			case "created_at":
				content = strings.ReplaceAll(content, v.GetKey(), svc.getFormattedTime(vd.Schedule.CreatedAt))
			case "created_by":
				content = strings.ReplaceAll(content, v.GetKey(), svc.getUsernameById(vd.Schedule.CreatedBy))
			case "updated_at":
				content = strings.ReplaceAll(content, v.GetKey(), svc.getFormattedTime(vd.Schedule.UpdatedAt))
			case "updated_by":
				content = strings.ReplaceAll(content, v.GetKey(), svc.getUsernameById(vd.Schedule.UpdatedBy))
			}

		case "alert":
			switch v.Name {
			case "id":
				content = strings.ReplaceAll(content, v.GetKey(), vd.Alert.Id.Hex())
			case "name":
				content = strings.ReplaceAll(content, v.GetKey(), vd.Alert.Name)
			case "description":
				content = strings.ReplaceAll(content, v.GetKey(), vd.Alert.Description)
			case "enabled":
				content = strings.ReplaceAll(content, v.GetKey(), fmt.Sprintf("%t", vd.Alert.Enabled))
			case "metric_name":
				content = strings.ReplaceAll(content, v.GetKey(), vd.Alert.MetricName)
			case "operator":
				content = strings.ReplaceAll(content, v.GetKey(), vd.Alert.Operator)
			case "lasting_seconds":
				content = strings.ReplaceAll(content, v.GetKey(), fmt.Sprintf("%d", vd.Alert.LastingSeconds))
			case "target_value":
				content = strings.ReplaceAll(content, v.GetKey(), svc.getFormattedTargetValue(vd.Alert))
			case "level":
				content = strings.ReplaceAll(content, v.GetKey(), vd.Alert.Level)
			}

		case "metric":
			if vd.Metric == nil {
				content = strings.ReplaceAll(content, v.GetKey(), "N/A")
				continue
			}
			switch v.Name {
			case "type":
				content = strings.ReplaceAll(content, v.GetKey(), vd.Metric.Type)
			case "node_id":
				content = strings.ReplaceAll(content, v.GetKey(), vd.Metric.NodeId.Hex())
			default:
				content = strings.ReplaceAll(content, v.GetKey(), svc.getFormattedMetricValue(v.Name, vd.Metric))
			}

		}
	}
	return content
}

func (svc *Service) getVariableData(args ...any) (vd VariableData) {
	for _, arg := range args {
		switch arg := arg.(type) {
		case *models.Task:
			vd.Task = arg
		case *models.TaskStat:
			vd.TaskStat = arg
		case *models.Spider:
			vd.Spider = arg
		case *models.Node:
			vd.Node = arg
		case *models.Schedule:
			vd.Schedule = arg
		case *models.NotificationAlert:
			vd.Alert = arg
		case *models.Metric:
			vd.Metric = arg
		}
	}
	return vd
}

func (svc *Service) parseTemplateVariables(template string) (variables []entity.NotificationVariable) {
	// regex pattern
	regex := regexp.MustCompile(`\$\{(\w+):(\w+)\}`)

	// find all matches
	matches := regex.FindAllStringSubmatch(template, -1)

	// variables map
	variablesMap := make(map[string]entity.NotificationVariable)

	// iterate over matches
	for _, match := range matches {
		variable := entity.NotificationVariable{
			Category: match[1],
			Name:     match[2],
		}
		key := fmt.Sprintf("%s:%s", variable.Category, variable.Name)
		if _, ok := variablesMap[key]; !ok {
			variablesMap[key] = variable
		}
	}

	// convert map to slice
	for _, variable := range variablesMap {
		variables = append(variables, variable)
	}

	return variables
}

func (svc *Service) getUsernameById(id primitive.ObjectID) (username string) {
	if id.IsZero() {
		return ""
	}
	u, err := service.NewModelService[models.User]().GetById(id)
	if err != nil {
		svc.Errorf("[NotificationService] get user error: %v", err)
		return ""
	}
	return u.Username
}

func (svc *Service) getFormattedTime(t time.Time) (res string) {
	if t.IsZero() {
		return "N/A"
	}
	return t.Local().Format(time.DateTime)
}

func (svc *Service) getFormattedTargetValue(a *models.NotificationAlert) (res string) {
	if strings.HasSuffix(a.MetricName, "_percent") {
		return fmt.Sprintf("%.2f%%", a.TargetValue)
	} else if strings.HasSuffix(a.MetricName, "_memory") {
		return fmt.Sprintf("%dMB", int(a.TargetValue/(1024*1024)))
	} else if strings.HasSuffix(a.MetricName, "_disk") {
		return fmt.Sprintf("%dGB", int(a.TargetValue/(1024*1024*1024)))
	} else if strings.HasSuffix(a.MetricName, "_rate") {
		return fmt.Sprintf("%.2fMB/s", a.TargetValue/(1024*1024))
	} else {
		return fmt.Sprintf("%f", a.TargetValue)
	}
}

func (svc *Service) getFormattedMetricValue(metricName string, m *models.Metric) (res string) {
	switch metricName {
	case "cpu_usage_percent":
		return fmt.Sprintf("%.2f%%", m.CpuUsagePercent)
	case "total_memory":
		return fmt.Sprintf("%dMB", m.TotalMemory/(1024*1024))
	case "available_memory":
		return fmt.Sprintf("%dMB", m.AvailableMemory/(1024*1024))
	case "used_memory":
		return fmt.Sprintf("%dMB", m.UsedMemory/(1024*1024))
	case "used_memory_percent":
		return fmt.Sprintf("%.2f%%", m.UsedMemoryPercent)
	case "total_disk":
		return fmt.Sprintf("%dGB", m.TotalDisk/(1024*1024*1024))
	case "available_disk":
		return fmt.Sprintf("%dGB", m.AvailableDisk/(1024*1024*1024))
	case "used_disk":
		return fmt.Sprintf("%dGB", m.UsedDisk/(1024*1024*1024))
	case "used_disk_percent":
		return fmt.Sprintf("%.2f%%", m.UsedDiskPercent)
	case "disk_read_bytes_rate":
		return fmt.Sprintf("%.2fMB/s", m.DiskReadBytesRate/(1024*1024))
	case "disk_write_bytes_rate":
		return fmt.Sprintf("%.2fMB/s", m.DiskWriteBytesRate/(1024*1024))
	case "network_bytes_sent_rate":
		return fmt.Sprintf("%.2fMB/s", m.NetworkBytesSentRate/(1024*1024))
	case "network_bytes_recv_rate":
		return fmt.Sprintf("%.2fMB/s", m.NetworkBytesRecvRate/(1024*1024))
	case "goroutine_count":
		return fmt.Sprintf("%d", m.GoroutineCount)
	default:
		return "N/A"
	}
}

func (svc *Service) convertMarkdownToHtml(content string) (html string) {
	return string(markdown.ToHTML([]byte(content), nil, nil))
}

func (svc *Service) SendNodeNotification(node *models.Node) {
	// arguments
	var args []any
	args = append(args, node)

	// settings
	settings, err := service.NewModelService[models.NotificationSetting]().GetMany(bson.M{
		"enabled": true,
		"trigger": bson.M{
			"$regex": constants.NotificationTriggerPatternNode,
		},
	}, nil)
	if err != nil {
		svc.Errorf("get notification settings error: %v", err)
		return
	}

	// Use bounded goroutine pool for node notifications
	const maxWorkers = 3
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, maxWorkers)

	for _, s := range settings {
		wg.Add(1)

		// Acquire semaphore
		semaphore <- struct{}{}

		go func(setting models.NotificationSetting) {
			defer func() {
				<-semaphore // Release semaphore
				wg.Done()
				if r := recover(); r != nil {
					svc.Errorf("[NotificationService] node notification panic for setting %s: %v", setting.Id.Hex(), r)
				}
			}()

			// send notification
			switch setting.Trigger {
			case constants.NotificationTriggerNodeStatusChange:
				svc.Send(&setting, args...)
			case constants.NotificationTriggerNodeOnline:
				if node.Status == constants.NodeStatusOnline {
					svc.Send(&setting, args...)
				}
			case constants.NotificationTriggerNodeOffline:
				if node.Status == constants.NodeStatusOffline {
					svc.Send(&setting, args...)
				}
			}
		}(s)
	}
	wg.Wait()
}

func (svc *Service) createRequestMail(s *models.NotificationSetting, ch *models.NotificationChannel, title, content string) (res *models.NotificationRequest, err error) {
	senderEmail := ch.SMTPUsername
	if s.UseCustomSenderEmail {
		senderEmail = s.SenderEmail
	}
	r := models.NotificationRequest{
		Status:      StatusSending,
		SettingId:   s.Id,
		ChannelId:   ch.Id,
		Title:       title,
		Content:     content,
		SenderEmail: senderEmail,
		SenderName:  s.SenderName,
		MailTo:      s.MailTo,
		MailCc:      s.MailCc,
		MailBcc:     s.MailBcc,
	}
	r.SetCreatedAt(time.Now())
	r.SetUpdatedAt(time.Now())
	r.Id, err = service.NewModelService[models.NotificationRequest]().InsertOne(r)
	if err != nil {
		svc.Errorf("[NotificationService] save request error: %v", err)
		return nil, err
	}
	return &r, nil
}

func (svc *Service) createRequestMailTest(s *models.NotificationSetting, ch *models.NotificationChannel, title, content string, mailTo []string) (res *models.NotificationRequest, err error) {
	if mailTo == nil {
		mailTo = []string{ch.SMTPUsername}
	}

	r := models.NotificationRequest{
		Status:      StatusSending,
		ChannelId:   ch.Id,
		Title:       title,
		Content:     content,
		SenderEmail: ch.SMTPUsername,
		SenderName:  ch.SMTPUsername,
		MailTo:      mailTo,
		Test:        true,
	}

	// Set SettingId if setting is provided (for non-test requests)
	if s != nil {
		r.SettingId = s.Id
	}

	r.SetCreatedAt(time.Now())
	r.SetUpdatedAt(time.Now())
	r.Id, err = service.NewModelService[models.NotificationRequest]().InsertOne(r)
	if err != nil {
		svc.Errorf("[NotificationService] save request error: %v", err)
		return nil, err
	}
	return &r, nil
}

func (svc *Service) createRequestIM(s *models.NotificationSetting, ch *models.NotificationChannel, title, content string, test bool) (res *models.NotificationRequest, err error) {
	r := models.NotificationRequest{
		Status:    StatusSending,
		ChannelId: ch.Id,
		Title:     title,
		Content:   content,
		Test:      test,
	}

	// Set SettingId if setting is provided (for non-test requests)
	if s != nil {
		r.SettingId = s.Id
	}

	r.SetCreatedAt(time.Now())
	r.SetUpdatedAt(time.Now())
	r.Id, err = service.NewModelService[models.NotificationRequest]().InsertOne(r)
	if err != nil {
		svc.Errorf("[NotificationService] save request error: %v", err)
		return nil, err
	}
	return &r, nil
}

func (svc *Service) saveRequest(r *models.NotificationRequest, err error) {
	if r == nil {
		return
	}

	if err != nil {
		r.Status = StatusError
		r.Error = err.Error()
	} else {
		r.Status = StatusSuccess
	}
	r.SetUpdatedAt(time.Now())
	err = service.NewModelService[models.NotificationRequest]().ReplaceById(r.Id, *r)
	if err != nil {
		svc.Errorf("[NotificationService] save request error: %v", err)
	}
}

func newNotificationService() *Service {
	return &Service{
		Logger: utils.NewLogger("NotificationService"),
	}
}

var _service *Service
var _serviceOnce sync.Once

func GetNotificationService() *Service {
	_serviceOnce.Do(func() {
		_service = newNotificationService()
	})
	return _service
}
