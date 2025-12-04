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

type LangfuseKeyPair struct {
	PublicKey string // pk-lf-<random> - used as Basic Auth username
	SecretKey string // sk-lf-<random> - used as Basic Auth password
}

func GenerateLangfuseKeys() (*LangfuseKeyPair, error) {
	publicBytes := make([]byte, 16)
	if _, err := rand.Read(publicBytes); err != nil {
		return nil, fmt.Errorf("failed to generate public key: %w", err)
	}
	publicKey := fmt.Sprintf("pk-lf-%s", hex.EncodeToString(publicBytes))

	secretBytes := make([]byte, 24)
	if _, err := rand.Read(secretBytes); err != nil {
		return nil, fmt.Errorf("failed to generate secret key: %w", err)
	}
	secretKey := fmt.Sprintf("sk-lf-%s", hex.EncodeToString(secretBytes))

	return &LangfuseKeyPair{
		PublicKey: publicKey,
		SecretKey: secretKey,
	}, nil
}
