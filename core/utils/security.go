package utils

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"

	"github.com/spf13/viper"
)

func NewSecret() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

const minimumAuthKeyLength = 32

// ValidateSecurityConfiguration rejects missing shared authentication material
// before the server accepts requests from workers or internal clients.
func ValidateSecurityConfiguration() error {
	if len(strings.TrimSpace(viper.GetString("auth.key"))) < minimumAuthKeyLength {
		return errors.New("auth key is required and must be at least 32 bytes; inject ASTIANGO_AUTH_KEY from a secret manager")
	}
	return ValidateEncryptionConfiguration()
}
