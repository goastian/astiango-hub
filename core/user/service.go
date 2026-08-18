package user

import (
	errors2 "errors"
	"fmt"
	"sync"
	"time"

	"github.com/goastian/astiango-hub/core/constants"
	"github.com/goastian/astiango-hub/core/errors"
	"github.com/goastian/astiango-hub/core/interfaces"
	"github.com/goastian/astiango-hub/core/models/models"
	"github.com/goastian/astiango-hub/core/models/service"
	"github.com/goastian/astiango-hub/core/utils"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

const minimumBootstrapAdminPasswordLength = 12

type Service struct {
	jwt *jwtKeyset
	interfaces.Logger
}

func (svc *Service) Init() (err error) {
	return svc.ensureBootstrapAdmin()
}

func (svc *Service) ensureBootstrapAdmin() (err error) {
	u, err := service.NewModelService[models.User]().GetOne(bson.M{"root_admin": true}, nil)
	if err != nil {
		if !errors2.Is(err, mongo.ErrNoDocuments) {
			return err
		}
	} else {
		// A root administrator already exists, so bootstrap credentials are not
		// read or retained by the service.
		return
	}

	username, password, err := loadBootstrapAdminCredentials()
	if err != nil {
		return err
	}
	passwordHash, err := utils.HashPassword(password)
	if err != nil {
		return err
	}
	u = &models.User{
		Username:           username,
		Password:           passwordHash,
		MustChangePassword: true,
		Role:               constants.RoleAdmin,
		RootAdmin:          true,
	}
	u.SetCreatedAt(time.Now())
	u.SetUpdatedAt(time.Now())
	_, err = service.NewModelService[models.User]().InsertOne(*u)
	return err
}

func loadBootstrapAdminCredentials() (username, password string, err error) {
	username = viper.GetString("bootstrap.admin.username")
	password = viper.GetString("bootstrap.admin.password")
	if username == "" || password == "" {
		return "", "", fmt.Errorf("bootstrap administrator is required for an empty installation; inject ASTIANGO_BOOTSTRAP_ADMIN_USERNAME and ASTIANGO_BOOTSTRAP_ADMIN_PASSWORD from a secret manager")
	}
	if len(password) < minimumBootstrapAdminPasswordLength {
		return "", "", fmt.Errorf("bootstrap administrator password must be at least %d characters", minimumBootstrapAdminPasswordLength)
	}
	return username, password, nil
}

func (svc *Service) Create(username, password, role, email string, by primitive.ObjectID) (err error) {
	// validate options
	if username == "" || password == "" {
		return errors.ErrorUserMissingRequiredFields
	}
	if len(password) < 5 {
		return errors.ErrorUserInvalidPassword
	}

	// normalize options
	if role == "" {
		role = constants.RoleNormal
	}

	// check if user exists
	if u, err := service.NewModelService[models.User]().GetOne(bson.M{"username": username}, nil); err == nil && u != nil && !u.Id.IsZero() {
		return errors.ErrorUserAlreadyExists
	}

	// add user
	passwordHash, err := utils.HashPassword(password)
	if err != nil {
		return err
	}
	u := models.User{
		Username: username,
		Role:     role,
		Password: passwordHash,
		Email:    email,
	}
	u.SetCreated(by)
	u.SetUpdated(by)
	_, err = service.NewModelService[models.User]().InsertOne(u)

	return err
}

func (svc *Service) CreateUser(u *models.User, by primitive.ObjectID) (err error) {
	// validate options
	if u.Username == "" || u.Password == "" {
		return errors.ErrorUserMissingRequiredFields
	}
	if len(u.Password) < 5 {
		return errors.ErrorUserInvalidPassword
	}

	// check if user exists
	if u, err := service.NewModelService[models.User]().GetOne(bson.M{"username": u.Username}, nil); err == nil && u != nil && !u.Id.IsZero() {
		return errors.ErrorUserAlreadyExists
	}

	passwordHash, err := utils.HashPassword(u.Password)
	if err != nil {
		return err
	}
	u.Password = passwordHash

	// add user
	u.SetCreated(by)
	u.SetUpdated(by)
	_, err = service.NewModelService[models.User]().InsertOne(*u)

	return err
}

func (svc *Service) Login(username, password string) (token string, u *models.User, err error) {
	pair, u, err := svc.LoginWithTokens(username, password)
	if err != nil {
		return "", nil, err
	}
	return pair.AccessToken, u, nil
}

func (svc *Service) LoginWithTokens(username, password string) (pair *TokenPair, u *models.User, err error) {
	u, err = service.NewModelService[models.User]().GetOne(bson.M{"username": username}, nil)
	if err != nil {
		return nil, nil, err
	}
	valid, needsMigration, err := utils.VerifyPassword(password, u.Password)
	if err != nil || !valid {
		return nil, nil, errors.ErrorUserMismatch
	}
	if needsMigration {
		u.Password, err = utils.HashPassword(password)
		if err != nil {
			return nil, nil, err
		}
		u.SetUpdatedAt(time.Now())
		if err := service.NewModelService[models.User]().ReplaceById(u.Id, *u); err != nil {
			return nil, nil, err
		}
	}
	pair, err = svc.issueTokenPair(u)
	if err != nil {
		return nil, nil, err
	}
	return pair, u, nil
}

func (svc *Service) CheckToken(tokenStr string) (u *models.User, err error) {
	return svc.checkToken(tokenStr)
}

func (svc *Service) ChangePassword(id primitive.ObjectID, password string, by primitive.ObjectID) (err error) {
	u, err := service.NewModelService[models.User]().GetById(id)
	if err != nil {
		return err
	}
	u.Password, err = utils.HashPassword(password)
	if err != nil {
		return err
	}
	u.MustChangePassword = false
	u.SetUpdated(by)
	return service.NewModelService[models.User]().ReplaceById(id, *u)
}

func (svc *Service) MakeToken(user *models.User) (tokenStr string, err error) {
	return svc.makeToken(user)
}

func (svc *Service) makeToken(user *models.User) (tokenStr string, err error) {
	tokenStr, _, err = svc.issueToken(user, accessTokenType, svc.jwt.accessTTL)
	return tokenStr, err
}

func (svc *Service) checkToken(tokenStr string) (user *models.User, err error) {
	claims, err := svc.parseToken(tokenStr, accessTokenType)
	if err != nil {
		return nil, errors2.New("invalid token")
	}
	revoked, err := svc.isAccessRevoked(claims.ID)
	if err != nil || revoked {
		return nil, errors2.New("invalid token")
	}

	id, err := primitive.ObjectIDFromHex(claims.Subject)
	if err != nil {
		return nil, errors2.New("invalid token")
	}
	u, err := service.NewModelService[models.User]().GetById(id)
	if err != nil {
		return nil, errors2.New("user not exists")
	}

	if u.Username != claims.Username {
		return nil, errors2.New("username mismatch")
	}

	return u, nil
}

func newUserService() (svc *Service, err error) {
	keyset, err := loadJWTKeyset()
	if err != nil {
		return nil, err
	}
	// service
	svc = &Service{
		jwt:    keyset,
		Logger: utils.NewLogger("UserService"),
	}

	// initialize
	if err := svc.Init(); err != nil {
		svc.Errorf("failed to initialize user service: %v", err)
		return nil, err
	}

	return svc, nil
}

var userSvc *Service
var userSvcOnce sync.Once

func GetUserService() (svc *Service, err error) {
	userSvcOnce.Do(func() {
		userSvc, err = newUserService()
		if err != nil {
			return
		}
	})
	return userSvc, nil
}
