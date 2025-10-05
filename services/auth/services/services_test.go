package services

import (
	"strings"
	"testing"
	"time"
	"zori/internal/config"
)

func TestPasswordService(t *testing.T) {
	ps := NewPasswordService()

	t.Run("HashPassword", func(t *testing.T) {
		password := "testpassword123"
		hash, err := ps.HashPassword(password)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if hash == "" {
			t.Fatal("Expected hash to be non-empty")
		}

		if hash == password {
			t.Fatal("Expected hash to be different from password")
		}
	})

	t.Run("VerifyPassword", func(t *testing.T) {
		password := "testpassword123"
		hash, _ := ps.HashPassword(password)

		err := ps.VerifyPassword(hash, password)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
	})

	t.Run("VerifyWrongPassword", func(t *testing.T) {
		password := "testpassword123"
		wrongPassword := "wrongpassword"
		hash, _ := ps.HashPassword(password)

		err := ps.VerifyPassword(hash, wrongPassword)
		if err == nil {
			t.Fatal("Expected error for wrong password")
		}
	})

	t.Run("IsPasswordValid", func(t *testing.T) {
		tests := []struct {
			password string
			valid    bool
		}{
			{"short", false},                  // too short
			{"validpass123", true},            // valid
			{strings.Repeat("a", 129), false}, // too long
		}

		for _, test := range tests {
			err := ps.IsPasswordValid(test.password)
			if test.valid && err != nil {
				t.Errorf("Expected password '%s' to be valid, got error: %v", test.password, err)
			}
			if !test.valid && err == nil {
				t.Errorf("Expected password '%s' to be invalid, got no error", test.password)
			}
		}
	})
}

func TestJWTService(t *testing.T) {
	js := NewJWTService(&config.Config{
		JWTSecretKey:       "test-secret-key-32-characters-long",
		JWTAccessTokenTTL:  15 * time.Minute,
		JWTRefreshTokenTTL: 7 * 24 * time.Hour,
	})

	t.Run("GenerateTokenPair", func(t *testing.T) {
		sessionID := "323e4567-e89b-12d3-a456-426614174002"
		accountID := "123e4567-e89b-12d3-a456-426614174000"
		orgID := "223e4567-e89b-12d3-a456-426614174001"
		email := "test@example.com"
		role := "owner"

		accessToken, refreshToken, err := js.GenerateTokenPair(sessionID, accountID, orgID, email, role)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if accessToken == "" {
			t.Fatal("Expected access token to be non-empty")
		}

		if refreshToken == "" {
			t.Fatal("Expected refresh token to be non-empty")
		}

		if accessToken == refreshToken {
			t.Fatal("Expected access and refresh tokens to be different")
		}
	})

	t.Run("ValidateAccessToken", func(t *testing.T) {
		sessionID := "323e4567-e89b-12d3-a456-426614174002"
		accountID := "123e4567-e89b-12d3-a456-426614174000"
		orgID := "223e4567-e89b-12d3-a456-426614174001"
		email := "test@example.com"
		role := "admin"

		accessToken, _, _ := js.GenerateTokenPair(sessionID, accountID, orgID, email, role)

		claims, err := js.ValidateAccessToken(accessToken)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if claims.AccountID != accountID {
			t.Errorf("Expected AccountID %s, got %s", accountID, claims.AccountID)
		}

		if claims.OrganizationID != orgID {
			t.Errorf("Expected OrganizationID %s, got %s", orgID, claims.OrganizationID)
		}

		if claims.Email != email {
			t.Errorf("Expected Email %s, got %s", email, claims.Email)
		}

		if claims.Role != role {
			t.Errorf("Expected Role %s, got %s", role, claims.Role)
		}
	})

	t.Run("ValidateRefreshToken", func(t *testing.T) {
		sessionID := "323e4567-e89b-12d3-a456-426614174002"
		accountID := "123e4567-e89b-12d3-a456-426614174000"
		orgID := "223e4567-e89b-12d3-a456-426614174001"
		email := "test@example.com"
		role := "member"

		_, refreshToken, _ := js.GenerateTokenPair(sessionID, accountID, orgID, email, role)

		refreshClaims, err := js.ValidateRefreshToken(refreshToken)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if refreshClaims.AccountID != accountID {
			t.Errorf("Expected AccountID %s, got %s", accountID, refreshClaims.AccountID)
		}

		if refreshClaims.SessionID != sessionID {
			t.Errorf("Expected SessionID %s, got %s", sessionID, refreshClaims.SessionID)
		}
	})

	t.Run("ValidateInvalidToken", func(t *testing.T) {
		invalidToken := "invalid.token.here"

		_, err := js.ValidateAccessToken(invalidToken)
		if err == nil {
			t.Fatal("Expected error for invalid token")
		}
	})
}

func TestTokenService(t *testing.T) {
	ts := NewTokenService()

	t.Run("GenerateSecureToken", func(t *testing.T) {
		length := 32
		token, err := ts.GenerateSecureToken(length)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(token) != length*2 { // hex encoding doubles the length
			t.Errorf("Expected token length %d, got %d", length*2, len(token))
		}

		// Generate another token and ensure they're different
		token2, _ := ts.GenerateSecureToken(length)
		if token == token2 {
			t.Error("Expected different tokens, got identical ones")
		}
	})

	t.Run("GenerateRefreshToken", func(t *testing.T) {
		token, err := ts.GenerateRefreshToken()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(token) != 128 { // 64 bytes * 2 (hex)
			t.Errorf("Expected token length 128, got %d", len(token))
		}

		if !ts.IsValidRefreshToken(token) {
			t.Error("Generated refresh token should be valid")
		}
	})

	t.Run("GenerateVerificationToken", func(t *testing.T) {
		token, err := ts.GenerateVerificationToken()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(token) != 64 { // 32 bytes * 2 (hex)
			t.Errorf("Expected token length 64, got %d", len(token))
		}

		if !ts.IsValidVerificationToken(token) {
			t.Error("Generated verification token should be valid")
		}
	})

	t.Run("ValidateTokenFormat", func(t *testing.T) {
		tests := []struct {
			token      string
			byteLength int
			valid      bool
		}{
			{"abcdef", 3, true},   // valid hex, correct length
			{"abcdef", 2, false},  // valid hex, wrong length
			{"xyz123", 3, false},  // invalid hex
			{"abcdef01", 4, true}, // valid hex, correct length
		}

		for _, test := range tests {
			valid := ts.ValidateTokenFormat(test.token, test.byteLength)
			if valid != test.valid {
				t.Errorf("Token '%s' with length %d: expected valid=%v, got %v",
					test.token, test.byteLength, test.valid, valid)
			}
		}
	})
}
