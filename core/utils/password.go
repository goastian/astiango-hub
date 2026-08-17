package utils

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	passwordHashAlgorithm = "argon2id"
	passwordHashVersion   = 19
	passwordHashMemory    = 64 * 1024 // KiB (64 MiB)
	passwordHashTime      = 3
	passwordHashThreads   = 2
	passwordHashSaltLen   = 16
	passwordHashKeyLen    = 32

	// Stored hash parameters must be bounded before invoking Argon2. This
	// prevents a malformed database value from turning a login attempt into a
	// memory or CPU exhaustion attack.
	passwordHashMaxMemory  = 256 * 1024 // KiB (256 MiB)
	passwordHashMaxTime    = 10
	passwordHashMaxThreads = 8
)

// HashPassword creates a self-describing Argon2id hash. Parameters are stored
// with the hash so they can be safely increased in a future release.
func HashPassword(password string) (string, error) {
	salt := make([]byte, passwordHashSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, passwordHashTime, passwordHashMemory, passwordHashThreads, passwordHashKeyLen)
	return fmt.Sprintf("%s$v=%d$m=%d,t=%d,p=%d$%s$%s",
		passwordHashAlgorithm,
		passwordHashVersion,
		passwordHashMemory,
		passwordHashTime,
		passwordHashThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

// VerifyPassword verifies either the current Argon2id format or a legacy MD5
// hash. The second result indicates that a valid legacy hash must be upgraded.
func VerifyPassword(password, encoded string) (valid, needsMigration bool, err error) {
	if strings.HasPrefix(encoded, passwordHashAlgorithm+"$") {
		valid, err = verifyArgon2idPassword(password, encoded)
		return valid, false, err
	}

	// MD5 was the historical format. Only a well-formed digest is considered a
	// legacy hash; malformed stored values are never accepted.
	if len(encoded) != 32 {
		return false, false, nil
	}
	legacyHash := EncryptMd5(password)
	if subtle.ConstantTimeCompare([]byte(strings.ToLower(encoded)), []byte(legacyHash)) != 1 {
		return false, false, nil
	}
	return true, true, nil
}

func verifyArgon2idPassword(password, encoded string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 5 || parts[0] != passwordHashAlgorithm {
		return false, fmt.Errorf("invalid Argon2id password hash format")
	}

	var version int
	if _, err := fmt.Sscanf(parts[1], "v=%d", &version); err != nil || version != passwordHashVersion {
		return false, fmt.Errorf("unsupported Argon2id password hash version")
	}

	var memory uint32
	var timeCost uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[2], "m=%d,t=%d,p=%d", &memory, &timeCost, &threads); err != nil {
		return false, fmt.Errorf("invalid Argon2id password hash parameters")
	}
	if memory == 0 || memory > passwordHashMaxMemory ||
		timeCost == 0 || timeCost > passwordHashMaxTime ||
		threads == 0 || threads > passwordHashMaxThreads {
		return false, fmt.Errorf("invalid Argon2id password hash parameters")
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil || len(salt) < passwordHashSaltLen {
		return false, fmt.Errorf("invalid Argon2id password salt")
	}
	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(expectedHash) == 0 {
		return false, fmt.Errorf("invalid Argon2id password hash")
	}

	actualHash := argon2.IDKey([]byte(password), salt, timeCost, memory, threads, uint32(len(expectedHash)))
	return subtle.ConstantTimeCompare(actualHash, expectedHash) == 1, nil
}
