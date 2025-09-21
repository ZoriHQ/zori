package services

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

type TokenService struct{}

func NewTokenService() *TokenService {
	return &TokenService{}
}

func (ts *TokenService) GenerateSecureToken(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("token length must be positive")
	}

	if length > 256 {
		return "", fmt.Errorf("token length too large, maximum is 256 bytes")
	}

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}

	return hex.EncodeToString(bytes), nil
}

func (ts *TokenService) GenerateRefreshToken() (string, error) {
	return ts.GenerateSecureToken(64)
}

func (ts *TokenService) GenerateVerificationToken() (string, error) {
	return ts.GenerateSecureToken(32)
}

func (ts *TokenService) GenerateResetToken() (string, error) {
	return ts.GenerateSecureToken(32)
}

func (ts *TokenService) GenerateAPIKey() (string, error) {
	return ts.GenerateSecureToken(48)
}

func (ts *TokenService) GenerateSessionID() (string, error) {
	return ts.GenerateSecureToken(24)
}

func (ts *TokenService) ValidateTokenFormat(token string, expectedByteLength int) bool {
	// Hex string should be 2x the byte length
	expectedStringLength := expectedByteLength * 2

	if len(token) != expectedStringLength {
		return false
	}

	_, err := hex.DecodeString(token)
	return err == nil
}

func (ts *TokenService) IsValidRefreshToken(token string) bool {
	return ts.ValidateTokenFormat(token, 64)
}

func (ts *TokenService) IsValidVerificationToken(token string) bool {
	return ts.ValidateTokenFormat(token, 32)
}

func (ts *TokenService) IsValidResetToken(token string) bool {
	return ts.ValidateTokenFormat(token, 32)
}

func (ts *TokenService) IsValidAPIKey(token string) bool {
	return ts.ValidateTokenFormat(token, 48)
}
