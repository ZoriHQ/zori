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

// OSSJWTClaims represents OSS JWT token structure with flat org_id field
type OSSJWTClaims struct {
	OrgID string `json:"org_id"`
	jwt.RegisteredClaims
}

// ClerkJWTClaims represents Clerk JWT token structure with nested organization object
type ClerkJWTClaims struct {
	O struct {
		ID   string `json:"id"`
		Role string `json:"rol"`
		Slug string `json:"slg"`
	} `json:"o"`
	jwt.RegisteredClaims
}

// JWTService provides JWT validation for websocket connections
// NOTE: This is a legacy service maintained only for websocket event streaming.
// All HTTP endpoints should use ClerkAuthMiddleware (or OSSAuthMiddleware in OSS mode) instead.
type JWTService struct{}

func NewJWTService() *JWTService {
	return &JWTService{}
}

// ValidateAccessToken validates JWT tokens from both Clerk and OSS auth systems
// This works with both token formats and normalizes them into JWTClaims
func (j *JWTService) ValidateAccessToken(tokenString string) (*JWTClaims, error) {
	clerkToken, _, err := jwt.NewParser().ParseUnverified(tokenString, &ClerkJWTClaims{})
	if err == nil {
		if clerkClaims, ok := clerkToken.Claims.(*ClerkJWTClaims); ok && clerkClaims.O.ID != "" {
			return &JWTClaims{
				OrganizationID: clerkClaims.O.ID,
				Role:           clerkClaims.O.Role,
			}, nil
		}
	}

	ossToken, _, err := jwt.NewParser().ParseUnverified(tokenString, &OSSJWTClaims{})
	if err == nil {
		if ossClaims, ok := ossToken.Claims.(*OSSJWTClaims); ok && ossClaims.OrgID != "" {
			return &JWTClaims{
				OrganizationID: ossClaims.OrgID,
			}, nil
		}
	}

	return nil, fmt.Errorf("invalid token format: unable to extract organization ID from token")
}
