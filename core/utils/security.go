package utils

import (
	"errors"
	"strings"

	"github.com/spf13/viper"
)

const minimumAuthKeyLength = 32

// ValidateSecurityConfiguration rejects missing shared authentication material
// before the server accepts requests from workers or internal clients.
func ValidateSecurityConfiguration() error {
	if len(strings.TrimSpace(viper.GetString("auth.key"))) < minimumAuthKeyLength {
		return errors.New("auth key is required and must be at least 32 bytes; inject ASTIANGO_AUTH_KEY from a secret manager")
	}
	return ValidateEncryptionConfiguration()
}
