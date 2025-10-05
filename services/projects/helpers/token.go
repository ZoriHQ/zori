package helpers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

func GenerateToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	randomPart := hex.EncodeToString(bytes)

	token := fmt.Sprintf("zori_pt_%s", randomPart)
	return token, nil
}
