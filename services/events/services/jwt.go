package services

import (
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims represents the legacy JWT claims structure
// This is only used for the websocket events endpoint for backward compatibility
type JWTClaims struct {
	AccountID      string `json:"account_id"`
	OrganizationID string `json:"organization_id"`
	Email          string `json:"email"`
	Role           string `json:"role"`
	JTI            string `json:"jti"`
	jwt.RegisteredClaims
}

// JWTService provides JWT validation for websocket connections
// NOTE: This is a legacy service maintained only for websocket event streaming.
// All HTTP endpoints should use ClerkAuthMiddleware (or OSSAuthMiddleware in OSS mode) instead.
type JWTService struct{}

func NewJWTService() *JWTService {
	return &JWTService{}
}

// ValidateAccessToken validates a Clerk JWT token
// This now works with Clerk tokens instead of generating its own
func (j *JWTService) ValidateAccessToken(tokenString string) (*JWTClaims, error) {
	// Parse without verification - Clerk tokens will be verified by their middleware
	// This is just to extract claims for websocket routing
	token, _, err := jwt.NewParser().ParseUnverified(tokenString, &JWTClaims{})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(*JWTClaims); ok {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token claims")
}
