package ir

import (
	"crypto/sha256"
	"encoding/hex"
)

func Hash(m *Model) (string, error) {
	b, err := CanonicalJSON(m)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}
