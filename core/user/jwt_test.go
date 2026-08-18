package user

import (
	"encoding/base64"
	"testing"
	"time"

	"github.com/goastian/astiango-hub/core/models/models"
	"github.com/goastian/astiango-hub/core/models/service"
	"github.com/goastian/astiango-hub/core/utils"
	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func currentJWTTestKey(t *testing.T) []byte {
	t.Helper()
	key, err := base64.StdEncoding.DecodeString("MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY=")
	require.NoError(t, err)
	return key
}

func TestJWTConfigurationRejectsMissingAndWeakSecrets(t *testing.T) {
	configureJWTForTest()
	t.Cleanup(configureJWTForTest)

	viper.Set("jwt.keyset", "")
	require.Error(t, ValidateJWTConfiguration())

	viper.Set("jwt.keyset", `{"active_kid":"weak","keys":{"weak":"c2hvcnQ="}}`)
	require.Error(t, ValidateJWTConfiguration())

	configureJWTForTest()
	require.NoError(t, ValidateJWTConfiguration())
}

func TestJWTTokenPairStrictValidationRotationAndRevocation(t *testing.T) {
	setupUserServiceTest(t)

	passwordHash, err := utils.HashPassword("jwt-password")
	require.NoError(t, err)
	modelSvc := service.NewModelService[models.User]()
	userID, err := modelSvc.InsertOne(models.User{Username: "jwt-user", Password: passwordHash})
	require.NoError(t, err)

	svc, err := GetUserService()
	require.NoError(t, err)
	pair, loggedInUser, err := svc.LoginWithTokens("jwt-user", "jwt-password")
	require.NoError(t, err)
	require.Equal(t, userID, loggedInUser.Id)
	require.NotEmpty(t, pair.AccessToken)
	require.NotEmpty(t, pair.RefreshToken)
	require.Greater(t, pair.AccessExpiresIn, int64(0))
	require.Greater(t, pair.RefreshExpiresIn, pair.AccessExpiresIn)

	accessClaims := &tokenClaims{}
	access, _, err := new(jwt.Parser).ParseUnverified(pair.AccessToken, accessClaims)
	require.NoError(t, err)
	require.Equal(t, "test-2026", access.Header["kid"])
	require.Equal(t, accessTokenType, accessClaims.TokenType)
	require.Equal(t, "astiango-hub-test", accessClaims.Issuer)
	require.Contains(t, accessClaims.Audience, "astiango-hub-test-api")
	require.Equal(t, userID.Hex(), accessClaims.Subject)
	require.NotNil(t, accessClaims.IssuedAt)
	require.NotNil(t, accessClaims.NotBefore)
	require.NotNil(t, accessClaims.ExpiresAt)
	require.NotEmpty(t, accessClaims.ID)

	_, err = svc.CheckToken(pair.AccessToken)
	require.NoError(t, err)

	// Rotation keeps the previous key available for validation while every new
	// token is signed with the active kid.
	previousKeyToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	previousKeyToken.Header["kid"] = "test-2025"
	previousKeyRaw, err := previousKeyToken.SignedString([]byte("fedcba9876543210fedcba9876543210"))
	require.NoError(t, err)
	_, err = svc.CheckToken(previousKeyRaw)
	require.NoError(t, err)

	// A valid-looking token signed with a different accepted JWT family is
	// rejected before its key can be used (algorithm-confusion defense).
	wrongAlgorithm := jwt.NewWithClaims(jwt.SigningMethodHS384, accessClaims)
	wrongAlgorithm.Header["kid"] = "test-2026"
	wrongRaw, err := wrongAlgorithm.SignedString(currentJWTTestKey(t))
	require.NoError(t, err)
	_, err = svc.CheckToken(wrongRaw)
	require.Error(t, err)

	rotated, err := svc.Refresh(pair.RefreshToken)
	require.NoError(t, err)
	require.NotEqual(t, pair.AccessToken, rotated.AccessToken)
	require.NotEqual(t, pair.RefreshToken, rotated.RefreshToken)
	_, err = svc.Refresh(pair.RefreshToken)
	require.Error(t, err, "refresh tokens must be single use")

	require.NoError(t, svc.Logout(rotated.AccessToken, rotated.RefreshToken))
	_, err = svc.CheckToken(rotated.AccessToken)
	require.Error(t, err, "logout must revoke the active access token")
	_, err = svc.Refresh(rotated.RefreshToken)
	require.Error(t, err, "logout must revoke the active refresh token")
}

func TestJWTRejectsRequiredClaimsAndExpiredTokens(t *testing.T) {
	setupUserServiceTest(t)
	svc, err := GetUserService()
	require.NoError(t, err)

	missingClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": "000000000000000000000000"})
	missingClaims.Header["kid"] = "test-2026"
	rawMissing, err := missingClaims.SignedString(currentJWTTestKey(t))
	require.NoError(t, err)
	_, err = svc.CheckToken(rawMissing)
	require.Error(t, err)

	now := time.Now().UTC()
	expired := jwt.NewWithClaims(jwt.SigningMethodHS256, tokenClaims{
		Username: "expired", TokenType: accessTokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: "astiango-hub-test", Subject: "000000000000000000000000", Audience: jwt.ClaimStrings{"astiango-hub-test-api"},
			IssuedAt: jwt.NewNumericDate(now.Add(-2 * time.Hour)), NotBefore: jwt.NewNumericDate(now.Add(-2 * time.Hour)), ExpiresAt: jwt.NewNumericDate(now.Add(-time.Hour)), ID: "expired-jti",
		},
	})
	expired.Header["kid"] = "test-2026"
	rawExpired, err := expired.SignedString(currentJWTTestKey(t))
	require.NoError(t, err)
	_, err = svc.CheckToken(rawExpired)
	require.Error(t, err)

	_, err = svc.Refresh("")
	require.Error(t, err)
}
