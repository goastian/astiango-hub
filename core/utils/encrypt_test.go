package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

const encryptionTestKeyset = `{"active_kid":"current","keys":{"previous":"MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY=","current":"ZmVkY2JhOTg3NjU0MzIxMGZlZGNiYTk4NzY1NDMyMTA="}}`

func configureEncryptionForTest(t *testing.T) {
	t.Helper()
	previousKeyset := viper.Get("encryption.keyset")
	previousLegacyKey := viper.Get("encryption.legacy_key")
	viper.Set("encryption.keyset", encryptionTestKeyset)
	viper.Set("encryption.legacy_key", "")
	t.Cleanup(func() {
		viper.Set("encryption.keyset", previousKeyset)
		viper.Set("encryption.legacy_key", previousLegacyKey)
	})
}

func TestEncryptAESUsesVersionedAuthenticatedCiphertext(t *testing.T) {
	configureEncryptionForTest(t)

	ciphertext, err := EncryptAES("database-password")
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(ciphertext, "agcm:v1:current:"))

	plaintext, err := DecryptAES(ciphertext)
	require.NoError(t, err)
	require.Equal(t, "database-password", plaintext)

	tampered := ciphertext[:len(ciphertext)-1] + "A"
	if strings.HasSuffix(ciphertext, "A") {
		tampered = ciphertext[:len(ciphertext)-1] + "B"
	}
	_, err = DecryptAES(tampered)
	require.Error(t, err, "GCM must reject modified ciphertext")
}

func TestEncryptAESSupportsKeyRotation(t *testing.T) {
	configureEncryptionForTest(t)
	viper.Set("encryption.keyset", `{"active_kid":"previous","keys":{"previous":"MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY=","current":"ZmVkY2JhOTg3NjU0MzIxMGZlZGNiYTk4NzY1NDMyMTA="}}`)
	oldCiphertext, err := EncryptAES("rotated credential")
	require.NoError(t, err)

	viper.Set("encryption.keyset", encryptionTestKeyset)
	plaintext, err := DecryptAES(oldCiphertext)
	require.NoError(t, err)
	require.Equal(t, "rotated credential", plaintext)

	newCiphertext, err := EncryptAES("rotated credential")
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(newCiphertext, "agcm:v1:current:"))
}

func TestReencryptAESMigratesLegacyCBCOnlyWithConfiguredLegacyKey(t *testing.T) {
	configureEncryptionForTest(t)
	legacyKey := []byte("legacy-key-16-b!")
	legacyCiphertext := legacyCBCForTest(t, "legacy credential", legacyKey)

	_, _, err := ReencryptAES(legacyCiphertext)
	require.Error(t, err)

	viper.Set("encryption.legacy_key", base64.StdEncoding.EncodeToString(legacyKey))
	migratedCiphertext, migrated, err := ReencryptAES(legacyCiphertext)
	require.NoError(t, err)
	require.True(t, migrated)
	require.True(t, strings.HasPrefix(migratedCiphertext, "agcm:v1:current:"))

	plaintext, err := DecryptAES(migratedCiphertext)
	require.NoError(t, err)
	require.Equal(t, "legacy credential", plaintext)
}

func TestEncryptAESRejectsMissingOrWeakKeysets(t *testing.T) {
	configureEncryptionForTest(t)
	viper.Set("encryption.keyset", "")
	require.Error(t, ValidateEncryptionConfiguration())

	viper.Set("encryption.keyset", `{"active_kid":"weak","keys":{"weak":"c2hvcnQ="}}`)
	require.Error(t, ValidateEncryptionConfiguration())
}

func legacyCBCForTest(t *testing.T, plaintext string, key []byte) string {
	t.Helper()
	block, err := aes.NewCipher(key)
	require.NoError(t, err)
	padded := append([]byte(plaintext), bytes.Repeat([]byte{byte(block.BlockSize() - len(plaintext)%block.BlockSize())}, block.BlockSize()-len(plaintext)%block.BlockSize())...)
	cipher.NewCBCEncrypter(block, key).CryptBlocks(padded, padded)
	return fmt.Sprintf("%x", padded)
}
