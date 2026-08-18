package middlewares

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/goastian/astiango-hub/core/constants"
	"github.com/goastian/astiango-hub/core/models/models"
	"github.com/goastian/astiango-hub/core/models/service"
	"github.com/goastian/astiango-hub/core/user"
	"github.com/goastian/astiango-hub/core/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"strconv"
	"sync"
	"time"
)

const (
	syncHeaderNodeKey    = "X-AstianGO-Node-Key"
	syncHeaderNodeSecret = "X-AstianGO-Node-Secret"
	syncHeaderTimestamp  = "X-AstianGO-Timestamp"
	syncHeaderNonce      = "X-AstianGO-Nonce"
	syncNonceTTL         = 2 * time.Minute
	syncNodeRateWindow   = time.Minute
	syncNodeRateLimit    = 120
)

var syncNonces = struct {
	sync.Mutex
	values map[string]time.Time
}{values: make(map[string]time.Time)}

type syncRateState struct {
	window time.Time
	count  int
}

var syncNodeRates = struct {
	sync.Mutex
	values map[string]syncRateState
}{values: make(map[string]syncRateState)}

func AuthorizationMiddleware() gin.HandlerFunc {
	userSvc, _ := user.GetUserService()
	return func(c *gin.Context) {
		// disable auth for test
		if utils.IsAuthDisabled() {
			u, err := service.NewModelService[models.User]().GetOne(bson.M{"root_admin": true}, nil)
			if err != nil {
				utils.HandleErrorInternalServerError(c, err)
				return
			}
			c.Set(constants.UserContextKey, u)
			c.Next()
			return
		}

		// token string
		tokenStr := utils.GetAPITokenFromContext(c)

		// validate token
		u, err := userSvc.CheckToken(tokenStr)
		if err != nil {
			// validation failed, return error response
			utils.HandleErrorUnauthorized(c, errors.New("invalid token"))
			return
		}

		// set user in context
		c.Set(constants.UserContextKey, u)

		// The bootstrap account is deliberately constrained until its password is
		// replaced. Enforcing this server-side prevents API clients from bypassing
		// the UI prompt.
		if u.MustChangePassword && !(c.Request.Method == "POST" && c.Request.URL.Path == "/users/me/change-password") {
			utils.HandleError(403, c, errors.New("password change required"))
			return
		}

		// validation success
		c.Next()
	}
}

func SyncAuthorizationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if utils.IsAuthDisabled() {
			c.Next()
			return
		}

		nodeKey, secret := c.GetHeader(syncHeaderNodeKey), c.GetHeader(syncHeaderNodeSecret)
		timestamp, parseErr := strconv.ParseInt(c.GetHeader(syncHeaderTimestamp), 10, 64)
		nonce := c.GetHeader(syncHeaderNonce)
		if nodeKey == "" || secret == "" || nonce == "" || parseErr != nil || time.Since(time.Unix(timestamp, 0)).Abs() > syncNonceTTL {
			utils.HandleErrorUnauthorized(c, errors.New("invalid auth key"))
			return
		}
		if !consumeSyncNonce(nodeKey, nonce) {
			utils.HandleErrorUnauthorized(c, errors.New("replayed sync request"))
			return
		}
		if !allowSyncNodeRequest(nodeKey) {
			utils.HandleError(429, c, errors.New("sync node rate limit exceeded"))
			return
		}
		node, err := service.NewModelService[models.Node]().GetOne(bson.M{"key": nodeKey, "enabled": true, "active": true}, nil)
		if err != nil {
			utils.HandleErrorUnauthorized(c, errors.New("unknown sync node"))
			return
		}
		valid, _, err := utils.VerifyPassword(secret, node.SyncKeyHash)
		if err != nil || !valid {
			utils.HandleErrorUnauthorized(c, errors.New("invalid sync node credential"))
			return
		}
		id, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			utils.HandleErrorUnauthorized(c, errors.New("invalid sync resource"))
			return
		}
		spider, err := service.NewModelService[models.Spider]().GetOne(bson.M{"$or": []bson.M{{"_id": id}, {"git_id": id}}}, nil)
		if err != nil {
			utils.HandleErrorUnauthorized(c, errors.New("unknown sync resource"))
			return
		}
		_, err = service.NewModelService[models.Task]().GetOne(bson.M{"node_id": node.Id, "spider_id": spider.Id, "status": bson.M{"$in": []string{"assigned", "running"}}}, nil)
		if err != nil {
			utils.HandleErrorUnauthorized(c, errors.New("sync node is not authorized for this resource"))
			return
		}
		c.Set("sync_node", node)

		c.Next()
	}
}

func consumeSyncNonce(nodeKey, nonce string) bool {
	now := time.Now()
	key := nodeKey + ":" + nonce
	syncNonces.Lock()
	defer syncNonces.Unlock()
	for existing, expiry := range syncNonces.values {
		if now.After(expiry) {
			delete(syncNonces.values, existing)
		}
	}
	if _, exists := syncNonces.values[key]; exists {
		return false
	}
	syncNonces.values[key] = now.Add(syncNonceTTL)
	return true
}

func ConsumeSyncNonce(nodeKey, nonce string) bool {
	if nonce == "" {
		return false
	}
	return consumeSyncNonce(nodeKey, nonce)
}

func allowSyncNodeRequest(nodeKey string) bool {
	now := time.Now()
	syncNodeRates.Lock()
	defer syncNodeRates.Unlock()
	state := syncNodeRates.values[nodeKey]
	if state.window.IsZero() || now.Sub(state.window) >= syncNodeRateWindow {
		state = syncRateState{window: now}
	}
	if state.count >= syncNodeRateLimit {
		return false
	}
	state.count++
	syncNodeRates.values[nodeKey] = state
	return true
}

func NewSyncNonce() string {
	value := make([]byte, 18)
	if _, err := rand.Read(value); err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(value)
}
