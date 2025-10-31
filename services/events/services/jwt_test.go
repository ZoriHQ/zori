package services

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateAccessToken_ClerkToken(t *testing.T) {
	// Create a Clerk-style token with nested organization structure
	clerkClaims := &ClerkJWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   "user_34pWkoDFpv13k3hCZEK8H3zK6gQ",
		},
	}
	clerkClaims.O.ID = "org_34pWlRrDmAJIsz5oAeIJWVFcYTm"
	clerkClaims.O.Role = "admin"
	clerkClaims.O.Slug = "zorihq"

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, clerkClaims)
	tokenString, err := token.SignedString([]byte("test-secret"))
	require.NoError(t, err)

	// Test validation
	jwtService := NewJWTService()
	claims, err := jwtService.ValidateAccessToken(tokenString)

	require.NoError(t, err)
	assert.Equal(t, "org_34pWlRrDmAJIsz5oAeIJWVFcYTm", claims.OrganizationID)
	assert.Equal(t, "admin", claims.Role)
}

func TestValidateAccessToken_OSSToken(t *testing.T) {
	// Create an OSS-style token with flat org_id structure
	ossClaims := &OSSJWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "zori-oss",
		},
		OrgID: "oss-org-123",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, ossClaims)
	tokenString, err := token.SignedString([]byte("test-secret"))
	require.NoError(t, err)

	// Test validation
	jwtService := NewJWTService()
	claims, err := jwtService.ValidateAccessToken(tokenString)

	require.NoError(t, err)
	assert.Equal(t, "oss-org-123", claims.OrganizationID)
}

func TestValidateAccessToken_InvalidToken(t *testing.T) {
	jwtService := NewJWTService()

	// Test with invalid token string
	_, err := jwtService.ValidateAccessToken("invalid-token")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid token format")
}

func TestValidateAccessToken_MissingOrgID(t *testing.T) {
	// Create a token without organization ID
	claims := &OSSJWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		OrgID: "", // Empty org ID
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte("test-secret"))
	require.NoError(t, err)

	// Test validation
	jwtService := NewJWTService()
	_, err = jwtService.ValidateAccessToken(tokenString)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid token format")
}
