package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

const (
	keySizeBytes   = 32
	nonceSizeBytes = 12
	minPayloadSize = nonceSizeBytes + 1
)

var (
	errEmptyPassword   = errors.New("password cannot be empty")
	errInvalidCipher   = errors.New("invalid ciphertext payload")
	errInvalidEncoding = errors.New("invalid ciphertext encoding")
)

func EncryptString(plaintext string, password string) (string, error) {
	if password == "" {
		return "", errEmptyPassword
	}

	key := deriveKey(password)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create aes cipher: %w", err)
	}

	gcm, err := cipher.NewGCMWithNonceSize(block, nonceSizeBytes)
	if err != nil {
		return "", fmt.Errorf("create gcm cipher: %w", err)
	}

	nonce := make([]byte, nonceSizeBytes)
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	cipherBytes := gcm.Seal(nil, nonce, []byte(plaintext), nil)
	payload := append(nonce, cipherBytes...)
	return base64.StdEncoding.EncodeToString(payload), nil
}

func DecryptString(ciphertext string, password string) (string, error) {
	if password == "" {
		return "", errEmptyPassword
	}

	payload, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("%w: %v", errInvalidEncoding, err)
	}
	if len(payload) < minPayloadSize {
		return "", errInvalidCipher
	}

	nonce := payload[:nonceSizeBytes]
	cipherBytes := payload[nonceSizeBytes:]
	key := deriveKey(password)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create aes cipher: %w", err)
	}

	gcm, err := cipher.NewGCMWithNonceSize(block, nonceSizeBytes)
	if err != nil {
		return "", fmt.Errorf("create gcm cipher: %w", err)
	}

	plainBytes, err := gcm.Open(nil, nonce, cipherBytes, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt payload: %w", err)
	}
	return string(plainBytes), nil
}

func deriveKey(password string) []byte {
	sum := sha256.Sum256([]byte(password))
	key := make([]byte, keySizeBytes)
	copy(key, sum[:])
	return key
}
