package utils

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestValidateSecurityConfigurationRequiresInjectedAuthAndEncryptionKeys(t *testing.T) {
	previousAuthKey := viper.Get("auth.key")
	previousKeyset := viper.Get("encryption.keyset")
	t.Cleanup(func() {
		viper.Set("auth.key", previousAuthKey)
		viper.Set("encryption.keyset", previousKeyset)
	})

	viper.Set("auth.key", "")
	viper.Set("encryption.keyset", encryptionTestKeyset)
	require.Error(t, ValidateSecurityConfiguration())

	viper.Set("auth.key", "shared-authentication-secret-value")
	require.NoError(t, ValidateSecurityConfiguration())
}
