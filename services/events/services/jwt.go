package services

import (
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

type JWTClaims struct {
	AccountID      string `json:"account_id"`
	OrganizationID string `json:"organization_id"`
	Email          string `json:"email"`
	Role           string `json:"role"`
	JTI            string `json:"jti"`
	jwt.RegisteredClaims
}

type OSSJWTClaims struct {
	OrgID string `json:"org_id"`
	jwt.RegisteredClaims
}

type ClerkJWTClaims struct {
	O struct {
		ID   string `json:"id"`
		Role string `json:"rol"`
		Slug string `json:"slg"`
	} `json:"o"`
	jwt.RegisteredClaims
}

type JWTService struct{}

func NewJWTService() *JWTService {
	return &JWTService{}
}

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
