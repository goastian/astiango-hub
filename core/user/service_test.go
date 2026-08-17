package user

import (
	"context"
	"strings"
	"testing"

	"github.com/goastian/astiango-hub/core/constants"
	"github.com/goastian/astiango-hub/core/models/models"
	"github.com/goastian/astiango-hub/core/models/service"
	"github.com/goastian/astiango-hub/core/mongo"
	"github.com/goastian/astiango-hub/core/utils"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
)

func setupUserServiceTest(t *testing.T) {
	t.Helper()
	viper.Set("mongo.db", "user_service_testdb")
	mongo.GetMongoDb("user_service_testdb").Drop(context.Background())
	t.Cleanup(func() { mongo.GetMongoDb("user_service_testdb").Drop(context.Background()) })
}

func TestLoginMigratesLegacyMD5Password(t *testing.T) {
	setupUserServiceTest(t)

	modelSvc := service.NewModelService[models.User]()
	id, err := modelSvc.InsertOne(models.User{
		Username: "legacy-user",
		Password: utils.EncryptMd5("legacy-password"),
	})
	require.NoError(t, err)

	svc, err := GetUserService()
	require.NoError(t, err)
	_, loggedInUser, err := svc.Login("legacy-user", "legacy-password")
	require.NoError(t, err)
	require.Equal(t, id, loggedInUser.Id)

	persisted, err := modelSvc.GetOne(bson.M{"_id": id}, nil)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(persisted.Password, "argon2id$"))
	valid, needsMigration, err := utils.VerifyPassword("legacy-password", persisted.Password)
	require.NoError(t, err)
	require.True(t, valid)
	require.False(t, needsMigration)
}

func TestLoginRequiresChangingDefaultAdminPasswordEvenWithArgon2id(t *testing.T) {
	setupUserServiceTest(t)

	passwordHash, err := utils.HashPassword(constants.DefaultAdminPassword)
	require.NoError(t, err)
	modelSvc := service.NewModelService[models.User]()
	id, err := modelSvc.InsertOne(models.User{
		Username:  constants.DefaultAdminUsername,
		Password:  passwordHash,
		RootAdmin: true,
	})
	require.NoError(t, err)

	svc, err := GetUserService()
	require.NoError(t, err)
	_, loggedInUser, err := svc.Login(constants.DefaultAdminUsername, constants.DefaultAdminPassword)
	require.NoError(t, err)
	require.True(t, loggedInUser.MustChangePassword)

	persisted, err := modelSvc.GetOne(bson.M{"_id": id}, nil)
	require.NoError(t, err)
	require.True(t, persisted.MustChangePassword)
}
