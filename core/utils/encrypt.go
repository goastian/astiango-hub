package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/viper"
)

const (
	encryptionCipherVersion = "v1"
	encryptionCipherPrefix  = "agcm"
	minimumEncryptionKeyLen = 32
)

type encryptionKeysetConfig struct {
	ActiveKID string            `json:"active_kid"`
	Keys      map[string]string `json:"keys"`
}

type encryptionKeyset struct {
	activeKID string
	keys      map[string][]byte
}

// ValidateEncryptionConfiguration verifies that encryption can only use a
// caller-provided AES-256 keyset. No key is compiled into the binary.
func ValidateEncryptionConfiguration() error {
	_, err := loadEncryptionKeyset()
	return err
}

func loadEncryptionKeyset() (*encryptionKeyset, error) {
	raw := viper.GetString("encryption.keyset")
	if raw == "" {
		return nil, errors.New("encryption keyset is required; inject ASTIANGO_ENCRYPTION_KEYSET from a secret manager")
	}

	var config encryptionKeysetConfig
	if err := json.Unmarshal([]byte(raw), &config); err != nil {
		return nil, fmt.Errorf("invalid encryption keyset JSON: %w", err)
	}
	if config.ActiveKID == "" || len(config.Keys) == 0 {
		return nil, errors.New("encryption keyset requires active_kid and keys")
	}

	keys := make(map[string][]byte, len(config.Keys))
	for kid, encoded := range config.Keys {
		if kid == "" || strings.Contains(kid, ":") {
			return nil, errors.New("encryption key id is invalid")
		}
		key, err := decodeBase64Key(encoded)
		if err != nil || len(key) != minimumEncryptionKeyLen {
			return nil, fmt.Errorf("encryption key %q must be Base64-encoded and exactly 32 bytes", kid)
		}
		keys[kid] = key
	}
	if _, ok := keys[config.ActiveKID]; !ok {
		return nil, errors.New("active encryption key id does not exist in the keyset")
	}

	return &encryptionKeyset{activeKID: config.ActiveKID, keys: keys}, nil
}

func decodeBase64Key(encoded string) ([]byte, error) {
	key, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil {
		key, err = base64.StdEncoding.DecodeString(encoded)
	}
	return key, err
}

func ComputeHmacSha256(message string, secret string) string {
	key := []byte(secret)
	h := hmac.New(sha256.New, key)
	h.Write([]byte(message))
	sha := hex.EncodeToString(h.Sum(nil))
	return base64.StdEncoding.EncodeToString([]byte(sha))
}

func EncryptMd5(str string) string {
	w := md5.New()
	_, _ = io.WriteString(w, str)
	md5str := fmt.Sprintf("%x", w.Sum(nil))
	return md5str
}

// EncryptAES encrypts a value using AES-256-GCM. The result is versioned and
// carries the key identifier needed to decrypt values during key rotation:
// agcm:v1:<kid>:<base64url(nonce|ciphertext|tag)>.
func EncryptAES(src string) (string, error) {
	keyset, err := loadEncryptionKeyset()
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(keyset.keys[keyset.activeKID])
	if err != nil {
		return "", fmt.Errorf("create AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create GCM cipher: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate GCM nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, []byte(src), encryptionAdditionalData(keyset.activeKID))
	payload := append(nonce, ciphertext...)
	return strings.Join([]string{
		encryptionCipherPrefix,
		encryptionCipherVersion,
		keyset.activeKID,
		base64.RawURLEncoding.EncodeToString(payload),
	}, ":"), nil
}

// DecryptAES decrypts the current AEAD ciphertext format. Legacy CBC values
// can only be read temporarily when ASTIANGO_ENCRYPTION_LEGACY_KEY is
// explicitly supplied for migration; no legacy key is embedded in the binary.
func DecryptAES(src string) (string, error) {
	plaintext, _, err := decryptAESWithMigrationStatus(src)
	return plaintext, err
}

// ReencryptAES converts a legacy ciphertext to the current AES-256-GCM
// format. Current ciphertexts are returned unchanged so the operation is
// safe to run repeatedly during a migration.
func ReencryptAES(src string) (ciphertext string, migrated bool, err error) {
	plaintext, needsMigration, err := decryptAESWithMigrationStatus(src)
	if err != nil {
		return "", false, err
	}
	if !needsMigration {
		return src, false, nil
	}
	ciphertext, err = EncryptAES(plaintext)
	if err != nil {
		return "", false, err
	}
	return ciphertext, true, nil
}

func decryptAESWithMigrationStatus(src string) (plaintext string, needsMigration bool, err error) {
	if strings.HasPrefix(src, encryptionCipherPrefix+":") {
		plaintext, err = decryptGCM(src)
		return plaintext, false, err
	}
	plaintext, err = decryptLegacyCBC(src)
	return plaintext, true, err
}

func decryptGCM(src string) (string, error) {
	parts := strings.Split(src, ":")
	if len(parts) != 4 || parts[0] != encryptionCipherPrefix || parts[1] != encryptionCipherVersion || parts[2] == "" || parts[3] == "" {
		return "", errors.New("invalid encrypted value format")
	}
	keyset, err := loadEncryptionKeyset()
	if err != nil {
		return "", err
	}
	key, ok := keyset.keys[parts[2]]
	if !ok {
		return "", errors.New("encrypted value uses an unavailable key id")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[3])
	if err != nil {
		return "", fmt.Errorf("decode encrypted value: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create GCM cipher: %w", err)
	}
	if len(payload) < gcm.NonceSize()+gcm.Overhead() {
		return "", errors.New("encrypted value is too short")
	}
	nonce, ciphertext := payload[:gcm.NonceSize()], payload[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, encryptionAdditionalData(parts[2]))
	if err != nil {
		return "", errors.New("encrypted value authentication failed")
	}
	return string(plaintext), nil
}

func encryptionAdditionalData(kid string) []byte {
	return []byte("astiango-hub:" + encryptionCipherPrefix + ":" + encryptionCipherVersion + ":" + kid)
}

func decryptLegacyCBC(src string) (string, error) {
	encodedKey := viper.GetString("encryption.legacy_key")
	if encodedKey == "" {
		return "", errors.New("legacy CBC ciphertext requires ASTIANGO_ENCRYPTION_LEGACY_KEY for migration")
	}
	key, err := decodeBase64Key(encodedKey)
	if err != nil || (len(key) != 16 && len(key) != 24 && len(key) != 32) {
		return "", errors.New("legacy encryption key must be Base64-encoded AES key material")
	}
	ciphertext, err := hex.DecodeString(src)
	if err != nil || len(ciphertext) == 0 || len(ciphertext)%aes.BlockSize != 0 {
		return "", errors.New("invalid legacy CBC ciphertext")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create legacy AES cipher: %w", err)
	}
	// Legacy data used the key as its IV. This compatibility path is available
	// only during a documented migration and is never used for encryption.
	cipher.NewCBCDecrypter(block, key[:aes.BlockSize]).CryptBlocks(ciphertext, ciphertext)
	plaintext, err := unpadPKCS7(ciphertext, aes.BlockSize)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func unpadPKCS7(src []byte, blockSize int) ([]byte, error) {
	if len(src) == 0 || len(src)%blockSize != 0 {
		return nil, errors.New("invalid PKCS#7 padding")
	}
	paddingLength := int(src[len(src)-1])
	if paddingLength == 0 || paddingLength > blockSize || paddingLength > len(src) {
		return nil, errors.New("invalid PKCS#7 padding")
	}
	for _, b := range src[len(src)-paddingLength:] {
		if int(b) != paddingLength {
			return nil, errors.New("invalid PKCS#7 padding")
		}
	}
	return src[:len(src)-paddingLength], nil
}
