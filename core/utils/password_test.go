package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPasswordHashUsesVersionedArgon2id(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(hash, "argon2id$v=19$m=65536,t=3,p=2$"))

	valid, needsMigration, err := VerifyPassword("correct horse battery staple", hash)
	require.NoError(t, err)
	require.True(t, valid)
	require.False(t, needsMigration)

	valid, needsMigration, err = VerifyPassword("wrong password", hash)
	require.NoError(t, err)
	require.False(t, valid)
	require.False(t, needsMigration)
}

func TestPasswordHashRecognizesLegacyMD5OnlyForMigration(t *testing.T) {
	legacyHash := EncryptMd5("legacy-password")

	valid, needsMigration, err := VerifyPassword("legacy-password", legacyHash)
	require.NoError(t, err)
	require.True(t, valid)
	require.True(t, needsMigration)

	valid, needsMigration, err = VerifyPassword("wrong-password", legacyHash)
	require.NoError(t, err)
	require.False(t, valid)
	require.False(t, needsMigration)

	valid, needsMigration, err = VerifyPassword("legacy-password", "not-a-password-hash")
	require.NoError(t, err)
	require.False(t, valid)
	require.False(t, needsMigration)
}

func TestPasswordHashRejectsUnsafeArgon2idParameters(t *testing.T) {
	hash, err := HashPassword("valid-password")
	require.NoError(t, err)

	unsafeHash := strings.Replace(hash, "m=65536", "m=999999999", 1)
	valid, _, err := VerifyPassword("valid-password", unsafeHash)
	require.Error(t, err)
	require.False(t, valid)
}
