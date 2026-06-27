package secretbox

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
)

const (
	envKey     = "ENCRYPTION_KEY"
	ciphertext = "v1:aesgcm:"
	keySize    = 32
)

type Box struct {
	aead cipher.AEAD
}

func FromEnv() (*Box, error) {
	value := strings.TrimSpace(os.Getenv(envKey))
	if value == "" {
		return nil, nil
	}
	box, err := New(value)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", envKey, err)
	}
	return box, nil
}

func New(encodedKey string) (*Box, error) {
	key, err := decodeKey(encodedKey)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create aes cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create gcm: %w", err)
	}
	return &Box{aead: aead}, nil
}

func (box *Box) Seal(value string) (string, error) {
	if box == nil {
		return "", errors.New("secret box is not configured")
	}
	nonce := make([]byte, box.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}
	sealed := box.aead.Seal(nonce, nonce, []byte(value), nil)
	return ciphertext + base64.RawStdEncoding.EncodeToString(sealed), nil
}

func (box *Box) Open(value string) (string, error) {
	if box == nil {
		return "", errors.New("secret box is not configured")
	}
	if !IsCiphertext(value) {
		return "", errors.New("secret is not encrypted with the current scheme")
	}
	encoded := strings.TrimPrefix(value, ciphertext)
	sealed, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("decode ciphertext: %w", err)
	}
	nonceSize := box.aead.NonceSize()
	if len(sealed) <= nonceSize {
		return "", errors.New("ciphertext is too short")
	}
	nonce := sealed[:nonceSize]
	ciphertextBytes := sealed[nonceSize:]
	plain, err := box.aead.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt secret: %w", err)
	}
	return string(plain), nil
}

func IsCiphertext(value string) bool {
	return strings.HasPrefix(value, ciphertext)
}

func decodeKey(value string) ([]byte, error) {
	candidates := []func(string) ([]byte, error){
		base64.StdEncoding.DecodeString,
		base64.RawStdEncoding.DecodeString,
		base64.RawURLEncoding.DecodeString,
		hex.DecodeString,
	}
	for _, decode := range candidates {
		key, err := decode(value)
		if err == nil && len(key) == keySize {
			return key, nil
		}
	}
	return nil, errors.New("must be base64 or hex encoded 32-byte key")
}
