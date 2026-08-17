package utils

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestEncryptAesPassword(t *testing.T) {
	plainText := "astiango-hub"
	encryptedText, err := EncryptAES(plainText)
	require.Nil(t, err)
	decryptedText, err := DecryptAES(encryptedText)
	require.Nil(t, err)
	require.Equal(t, decryptedText, plainText)
	require.NotEqual(t, decryptedText, encryptedText)
}
