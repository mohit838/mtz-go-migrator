package migration

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
)

func checksumFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:]), nil
}
