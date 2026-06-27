package core

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

func NewPrefixedID(prefix string) (string, error) {
	var bytes [12]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return prefix + "_" + hex.EncodeToString(bytes[:]), nil
}

func StablePrefixedID(prefix string, key string) string {
	digest := sha256.Sum256([]byte(key))
	return prefix + "_" + hex.EncodeToString(digest[:12])
}
