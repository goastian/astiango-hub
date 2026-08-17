package user

import (
	errors2 "errors"
	"github.com/goastian/astiango-hub/core/constants"
	"github.com/goastian/astiango-hub/core/errors"
	"github.com/goastian/astiango-hub/core/interfaces"
	"github.com/goastian/astiango-hub/core/models/models"
	"github.com/goastian/astiango-hub/core/models/service"
	"github.com/goastian/astiango-hub/core/utils"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"sync"
	"time"
)

type Service struct {
	jwtSecret        string
	jwtSigningMethod jwt.SigningMethod
	interfaces.Logger
}

func (svc *Service) Init() (err error) {
	if utils.IsPro() {
		return svc.initPro()
	}
	return svc.init()
}

func (svc *Service) init() (err error) {
	u, err := service.NewModelService[models.User]().GetOne(bson.M{"username": constants.DefaultAdminUsername}, nil)
	if err != nil {
		if !errors2.Is(err, mongo.ErrNoDocuments) {
			return err
		}
	} else {
		// exists
		return
	}

	// add user
	u = &models.User{
		Username:  constants.DefaultAdminUsername,
		Password:  utils.EncryptMd5(constants.DefaultAdminPassword),
		Role:      constants.RoleAdmin,
		RootAdmin: true,
	}
	u.SetCreatedAt(time.Now())
	u.SetUpdatedAt(time.Now())
	_, err = service.NewModelService[models.User]().InsertOne(*u)
	return err
}

func (svc *Service) initPro() (err error) {
	u, err := service.NewModelService[models.User]().GetOne(bson.M{
		"$or": []bson.M{
			{"username": constants.DefaultAdminUsername},
			{"root_admin": true},
		},
	}, nil)
	if err != nil {
		if !errors2.Is(err, mongo.ErrNoDocuments) {
			return err
		}
	} else {
		// exists, compatible with old versions
		u.RootAdmin = true
		u.SetUpdatedAt(time.Now())
		err = service.NewModelService[models.User]().ReplaceById(u.Id, *u)
		if err != nil {
			svc.Errorf("failed to update user: %v", err)
		}
		return
	}

	// add user
	u = &models.User{
		Username:  constants.DefaultAdminUsername,
		Password:  utils.EncryptMd5(constants.DefaultAdminPassword),
		RootAdmin: true,
	}
	u.SetCreatedAt(time.Now())
	u.SetUpdatedAt(time.Now())
	_, err = service.NewModelService[models.User]().InsertOne(*u)
	return err
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
	u := models.User{
		Username: username,
		Role:     role,
		Password: utils.EncryptMd5(password),
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

	// add user
	u.SetCreated(by)
	u.SetUpdated(by)
	_, err = service.NewModelService[models.User]().InsertOne(*u)

	return err
}

func (svc *Service) Login(username, password string) (token string, u *models.User, err error) {
	u, err = service.NewModelService[models.User]().GetOne(bson.M{"username": username}, nil)
	if err != nil {
		return "", nil, err
	}
	if u.Password != utils.EncryptMd5(password) {
		return "", nil, errors.ErrorUserMismatch
	}
	token, err = svc.makeToken(u)
	if err != nil {
		return "", nil, err
	}
	return token, u, nil
}

func (svc *Service) CheckToken(tokenStr string) (u *models.User, err error) {
	return svc.checkToken(tokenStr)
}

func (svc *Service) ChangePassword(id primitive.ObjectID, password string, by primitive.ObjectID) (err error) {
	u, err := service.NewModelService[models.User]().GetById(id)
	if err != nil {
		return err
	}
	u.Password = utils.EncryptMd5(password)
	u.SetCreatedBy(by)
	return service.NewModelService[models.User]().ReplaceById(id, *u)
}

func (svc *Service) MakeToken(user *models.User) (tokenStr string, err error) {
	return svc.makeToken(user)
}

func (svc *Service) makeToken(user *models.User) (tokenStr string, err error) {
	token := jwt.NewWithClaims(svc.jwtSigningMethod, jwt.MapClaims{
		"id":       user.Id,
		"username": user.Username,
		"nbf":      time.Now().Unix(),
	})
	return token.SignedString([]byte(svc.jwtSecret))
}

func (svc *Service) checkToken(tokenStr string) (user *models.User, err error) {
	token, err := jwt.Parse(tokenStr, svc.getSecretFunc())
	if err != nil {
		return nil, errors2.New("invalid token")
	}

	claim, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors2.New("invalid type")
	}

	if !token.Valid {
		return nil, errors2.New("invalid token")
	}

	id, err := primitive.ObjectIDFromHex(claim["id"].(string))
	if err != nil {
		return nil, errors2.New("invalid token")
	}
	username := claim["username"].(string)
	u, err := service.NewModelService[models.User]().GetById(id)
	if err != nil {
		return nil, errors2.New("user not exists")
	}

	if username != u.Username {
		return nil, errors2.New("username mismatch")
	}

	return u, nil
}

func (svc *Service) getSecretFunc() jwt.Keyfunc {
	return func(token *jwt.Token) (interface{}, error) {
		return []byte(svc.jwtSecret), nil
	}
}

func newUserService() (svc *Service, err error) {
	// service
	svc = &Service{
		jwtSecret:        "astiango-hub",
		jwtSigningMethod: jwt.SigningMethodHS256,
		Logger:           utils.NewLogger("UserService"),
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
