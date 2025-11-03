package services

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateAccessToken_ClerkToken(t *testing.T) {
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

	jwtService := NewJWTService()
	claims, err := jwtService.ValidateAccessToken(tokenString)

	require.NoError(t, err)
	assert.Equal(t, "org_34pWlRrDmAJIsz5oAeIJWVFcYTm", claims.OrganizationID)
	assert.Equal(t, "admin", claims.Role)
}

func TestValidateAccessToken_OSSToken(t *testing.T) {
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

	jwtService := NewJWTService()
	claims, err := jwtService.ValidateAccessToken(tokenString)

	require.NoError(t, err)
	assert.Equal(t, "oss-org-123", claims.OrganizationID)
}

func TestValidateAccessToken_InvalidToken(t *testing.T) {
	jwtService := NewJWTService()

	_, err := jwtService.ValidateAccessToken("invalid-token")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid token format")
}

func TestValidateAccessToken_MissingOrgID(t *testing.T) {
	claims := &OSSJWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		OrgID: "",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte("test-secret"))
	require.NoError(t, err)

	jwtService := NewJWTService()
	_, err = jwtService.ValidateAccessToken(tokenString)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid token format")
}
