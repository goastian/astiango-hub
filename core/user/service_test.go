package user

import (
	"context"
	"strings"
	"testing"

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
	configureJWTForTest()
	mongo.GetMongoDb("user_service_testdb").Drop(context.Background())
	t.Cleanup(func() { mongo.GetMongoDb("user_service_testdb").Drop(context.Background()) })
}

func configureJWTForTest() {
	viper.Set("jwt.keyset", `{"active_kid":"test-2026","keys":{"test-2025":"ZmVkY2JhOTg3NjU0MzIxMGZlZGNiYTk4NzY1NDMyMTA=","test-2026":"MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY="}}`)
	viper.Set("jwt.issuer", "astiango-hub-test")
	viper.Set("jwt.audience", "astiango-hub-test-api")
	viper.Set("jwt.access_ttl", "15m")
	viper.Set("jwt.refresh_ttl", "168h")
	viper.Set("jwt.leeway", "0s")
	viper.Set("bootstrap.admin.username", "test-bootstrap-admin")
	viper.Set("bootstrap.admin.password", "test-bootstrap-password")
}

func TestBootstrapAdminCredentialsRequireInjectedStrongValues(t *testing.T) {
	previousUsername := viper.Get("bootstrap.admin.username")
	previousPassword := viper.Get("bootstrap.admin.password")
	t.Cleanup(func() {
		viper.Set("bootstrap.admin.username", previousUsername)
		viper.Set("bootstrap.admin.password", previousPassword)
	})

	viper.Set("bootstrap.admin.username", "")
	viper.Set("bootstrap.admin.password", "")
	_, _, err := loadBootstrapAdminCredentials()
	require.Error(t, err)

	viper.Set("bootstrap.admin.username", "bootstrap-admin")
	viper.Set("bootstrap.admin.password", "short")
	_, _, err = loadBootstrapAdminCredentials()
	require.Error(t, err)

	viper.Set("bootstrap.admin.password", "unique-bootstrap-password")
	username, password, err := loadBootstrapAdminCredentials()
	require.NoError(t, err)
	require.Equal(t, "bootstrap-admin", username)
	require.Equal(t, "unique-bootstrap-password", password)
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

func TestLoginPreservesBootstrapPasswordChangeRequirement(t *testing.T) {
	setupUserServiceTest(t)

	passwordHash, err := utils.HashPassword("test-bootstrap-password")
	require.NoError(t, err)
	modelSvc := service.NewModelService[models.User]()
	id, err := modelSvc.InsertOne(models.User{
		Username:           "test-bootstrap-admin",
		Password:           passwordHash,
		RootAdmin:          true,
		MustChangePassword: true,
	})
	require.NoError(t, err)

	svc, err := GetUserService()
	require.NoError(t, err)
	_, loggedInUser, err := svc.Login("test-bootstrap-admin", "test-bootstrap-password")
	require.NoError(t, err)
	require.True(t, loggedInUser.MustChangePassword)

	persisted, err := modelSvc.GetOne(bson.M{"_id": id}, nil)
	require.NoError(t, err)
	require.True(t, persisted.MustChangePassword)
}
