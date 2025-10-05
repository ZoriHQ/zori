package services

import (
	"fmt"
	"time"
	"zori/internal/config"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type JWTClaims struct {
	AccountID      string `json:"account_id"`
	OrganizationID string `json:"organization_id"`
	Email          string `json:"email"`
	Role           string `json:"role"`
	JTI            string `json:"jti"` // JWT ID for uniqueness
	jwt.RegisteredClaims
}

type RefreshTokenClaims struct {
	SessionID string `json:"session_id"`
	AccountID string `json:"account_id"`
	JTI       string `json:"jti"` // JWT ID for uniqueness
	jwt.RegisteredClaims
}

type JWTService struct {
	secretKey       []byte
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

func NewJWTService(cfg *config.Config) *JWTService {
	return &JWTService{
		secretKey:       []byte(cfg.JWTSecretKey),
		accessTokenTTL:  cfg.JWTAccessTokenTTL,
		refreshTokenTTL: cfg.JWTRefreshTokenTTL,
	}
}

func (j *JWTService) GenerateTokenPair(sessionID, accountID, orgID, email, role string) (accessToken, refreshToken string, err error) {
	// Generate access token
	accessToken, err = j.GenerateAccessToken(accountID, orgID, email, role)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate refresh token with session ID
	refreshToken, err = j.GenerateRefreshToken(sessionID, accountID)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

func (j *JWTService) GenerateAccessToken(accountID, orgID, email, role string) (string, error) {
	claims := JWTClaims{
		AccountID:      accountID,
		OrganizationID: orgID,
		Email:          email,
		Role:           role,
		JTI:            uuid.New().String(), // Add unique JWT ID
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.accessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "zori-auth",
			Subject:   accountID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secretKey)
}

func (j *JWTService) GenerateRefreshToken(sessionID, accountID string) (string, error) {
	claims := RefreshTokenClaims{
		SessionID: sessionID,
		AccountID: accountID,
		JTI:       uuid.New().String(), // Add unique JWT ID
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.refreshTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "zori-auth",
			Subject:   accountID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secretKey)
}

func (j *JWTService) ValidateAccessToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return j.secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

func (j *JWTService) ValidateRefreshToken(tokenString string) (*RefreshTokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &RefreshTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return j.secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse refresh token: %w", err)
	}

	if claims, ok := token.Claims.(*RefreshTokenClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid refresh token")
}

func (j *JWTService) GetTokenExpiry(tokenString string) (*time.Time, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		return j.secretKey, nil
	}, jwt.WithoutClaimsValidation())

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(*jwt.RegisteredClaims); ok {
		if claims.ExpiresAt != nil {
			expiry := claims.ExpiresAt.Time
			return &expiry, nil
		}
	}

	return nil, fmt.Errorf("no expiry found in token")
}

func (j *JWTService) IsTokenExpired(tokenString string) bool {
	expiry, err := j.GetTokenExpiry(tokenString)
	if err != nil {
		return true
	}

	return time.Now().After(*expiry)
}
