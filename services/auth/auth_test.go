package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"zori/di"
	"zori/internal/storage/postgres/models"
	"zori/services/auth/services"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthService_Register(t *testing.T) {
	tc := di.NewTestContainer(t)
	defer tc.Cleanup()

	randomEmail := fmt.Sprintf("test-%d@example.com", time.Now().UnixNano())

	tests := []struct {
		name           string
		request        services.RegisterRequest
		expectedStatus int
		checkResponse  func(t *testing.T, response *services.AuthResponse)
		expectError    bool
	}{
		{
			name: "successful registration",
			request: services.RegisterRequest{
				Email:            randomEmail,
				Password:         "ValidPass123!",
				FirstName:        "John",
				LastName:         "Doe",
				OrganizationName: "Test Org",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response *services.AuthResponse) {
				assert.NotEmpty(t, response.AccessToken)
				assert.NotEmpty(t, response.RefreshToken)
				assert.Equal(t, int64(15*60), response.ExpiresIn)
				assert.NotEqual(t, "", response.Account.Email)
				assert.Equal(t, "John", response.Account.FirstName)
				assert.Equal(t, "Doe", response.Account.LastName)
				assert.Equal(t, "Test Org", response.Organization.Name)
				assert.Contains(t, response.Organization.Slug, "test-org")
			},
			expectError: false,
		},
		{
			name: "duplicate email registration",
			request: services.RegisterRequest{
				Email:            randomEmail, // Use the same email to test duplicate
				Password:         "ValidPass123!",
				FirstName:        "John",
				LastName:         "Doe",
				OrganizationName: "Test Org",
			},
			expectedStatus: http.StatusInternalServerError,
			expectError:    true,
		},
		{
			name: "invalid email format",
			request: services.RegisterRequest{
				Email:            "invalid-email",
				Password:         "ValidPass123!",
				FirstName:        "Invalid",
				LastName:         "User",
				OrganizationName: "Invalid Org",
			},
			expectedStatus: http.StatusInternalServerError,
			expectError:    true,
		},
		{
			name: "password too short",
			request: services.RegisterRequest{
				Email:            "short@example.com",
				Password:         "short",
				FirstName:        "Short",
				LastName:         "Pass",
				OrganizationName: "Short Org",
			},
			expectedStatus: http.StatusInternalServerError,
			expectError:    true,
		},
		{
			name: "missing organization name",
			request: services.RegisterRequest{
				Email:     "noorg@example.com",
				Password:  "ValidPass123!",
				FirstName: "No",
				LastName:  "Org",
			},
			expectedStatus: http.StatusInternalServerError,
			expectError:    true,
		},
	}

	// First, register a user for duplicate email test
	// e := echo.New()
	setupRequest := services.RegisterRequest{
		Email:            "duplicate@example.com",
		Password:         "ValidPass123!",
		FirstName:        "Original",
		LastName:         "User",
		OrganizationName: "Original Org",
	}
	reqBody, _ := json.Marshal(setupRequest)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	// c := e.NewContext(req, rec)
	tc.Server.Echo.ServeHTTP(rec, req)

	// Run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare request
			reqBody, err := json.Marshal(tt.request)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(reqBody))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()

			// Execute request
			tc.Server.Echo.ServeHTTP(rec, req)

			// Check status code
			if rec.Code != tt.expectedStatus {
				t.Logf("Unexpected status code. Response body: %s", rec.Body.String())
			}
			assert.Equal(t, tt.expectedStatus, rec.Code)

			if !tt.expectError {
				// Parse response
				var response services.AuthResponse
				err := json.Unmarshal(rec.Body.Bytes(), &response)
				require.NoError(t, err)

				// Check response
				if tt.checkResponse != nil {
					tt.checkResponse(t, &response)
				}

				// Verify data was persisted in database
				ctx := context.Background()

				// Check account exists
				var account models.Account
				err = tc.DB.DB.NewSelect().
					Model(&account).
					Where("email = ?", tt.request.Email).
					Scan(ctx)
				require.NoError(t, err)
				assert.Equal(t, tt.request.Email, account.Email)
				assert.Equal(t, tt.request.FirstName, account.FirstName)
				assert.Equal(t, tt.request.LastName, account.LastName)

				// Check organization exists
				var org models.Organization
				err = tc.DB.DB.NewSelect().
					Model(&org).
					Where("name = ?", tt.request.OrganizationName).
					Scan(ctx)
				require.NoError(t, err)
				assert.Equal(t, tt.request.OrganizationName, org.Name)

				// Check organization member exists with owner role
				var member models.OrganizationMember
				err = tc.DB.DB.NewSelect().
					Model(&member).
					Where("account_id = ? AND organization_id = ?", response.Account.ID, response.Organization.ID).
					Scan(ctx)
				require.NoError(t, err)
				assert.Equal(t, models.RoleOwner, member.Role)

				// Check session was created
				var session models.Session
				err = tc.DB.DB.NewSelect().
					Model(&session).
					Where("account_id = ?", response.Account.ID).
					Scan(ctx)
				require.NoError(t, err)
				assert.True(t, session.ExpiresAt.After(time.Now()))
			}
		})
	}
}

func TestAuthService_Login(t *testing.T) {
	tc := di.NewTestContainer(t)
	defer tc.Cleanup()

	// Generate random email for this test run
	randomEmail := fmt.Sprintf("login-%d@example.com", time.Now().UnixNano())

	// First, register a user to test login
	setupUser := services.RegisterRequest{
		Email:            randomEmail,
		Password:         "ValidPass123!",
		FirstName:        "Login",
		LastName:         "Test",
		OrganizationName: "Login Org",
	}

	reqBody, _ := json.Marshal(setupUser)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	tc.Server.Echo.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, "Failed to register test user")

	tests := []struct {
		name           string
		request        services.LoginRequest
		expectedStatus int
		checkResponse  func(t *testing.T, response *services.AuthResponse)
		expectError    bool
	}{
		{
			name: "successful login",
			request: services.LoginRequest{
				Email:    randomEmail,
				Password: "ValidPass123!",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response *services.AuthResponse) {
				assert.NotEmpty(t, response.AccessToken)
				assert.NotEmpty(t, response.RefreshToken)
				assert.Equal(t, int64(15*60), response.ExpiresIn)
				assert.Equal(t, randomEmail, response.Account.Email)
				assert.Equal(t, "Login", response.Account.FirstName)
				assert.Equal(t, "Test", response.Account.LastName)
				assert.Equal(t, "Login Org", response.Organization.Name)
			},
			expectError: false,
		},
		{
			name: "wrong password",
			request: services.LoginRequest{
				Email:    randomEmail,
				Password: "WrongPassword123!",
			},
			expectedStatus: http.StatusInternalServerError,
			expectError:    true,
		},
		{
			name: "non-existent email",
			request: services.LoginRequest{
				Email:    "nonexistent@example.com",
				Password: "SomePassword123!",
			},
			expectedStatus: http.StatusInternalServerError,
			expectError:    true,
		},
		{
			name: "invalid email format",
			request: services.LoginRequest{
				Email:    "invalid-email",
				Password: "ValidPass123!",
			},
			expectedStatus: http.StatusInternalServerError,
			expectError:    true,
		},
		{
			name: "empty password",
			request: services.LoginRequest{
				Email:    "login@example.com",
				Password: "",
			},
			expectedStatus: http.StatusInternalServerError,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare request
			reqBody, err := json.Marshal(tt.request)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(reqBody))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()

			// Execute request
			tc.Server.Echo.ServeHTTP(rec, req)

			// Check status code
			if rec.Code != tt.expectedStatus {
				t.Logf("Unexpected status code. Response body: %s", rec.Body.String())
			}
			assert.Equal(t, tt.expectedStatus, rec.Code)

			if !tt.expectError {
				// Parse response
				var response services.AuthResponse
				err := json.Unmarshal(rec.Body.Bytes(), &response)
				require.NoError(t, err)

				// Check response
				if tt.checkResponse != nil {
					tt.checkResponse(t, &response)
				}

				ctx := context.Background()
				var session models.Session
				err = tc.DB.DB.NewSelect().
					Model(&session).
					Where("account_id = ?", response.Account.ID).
					Order("created_at DESC").
					Limit(1).
					Scan(ctx)
				require.NoError(t, err)
				assert.True(t, session.ExpiresAt.After(time.Now()))
			}
		})
	}
}

func TestAuthService_RefreshToken(t *testing.T) {
	tc := di.NewTestContainer(t)
	defer tc.Cleanup()
	randomEmail := fmt.Sprintf("refresh-%d@example.com", time.Now().UnixNano())

	setupUser := services.RegisterRequest{
		Email:            randomEmail,
		Password:         "ValidPass123!",
		FirstName:        "Refresh",
		LastName:         "Test",
		OrganizationName: "Refresh Org",
	}

	reqBody, _ := json.Marshal(setupUser)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	tc.Server.Echo.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var initialResponse services.AuthResponse
	err := json.Unmarshal(rec.Body.Bytes(), &initialResponse)
	require.NoError(t, err)

	t.Run("successful token refresh", func(t *testing.T) {
		refreshRequest := map[string]string{
			"refresh_token": initialResponse.RefreshToken,
		}

		reqBody, err := json.Marshal(refreshRequest)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBuffer(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()

		tc.Server.Echo.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Logf("Response body: %s", rec.Body.String())
		}
		assert.Equal(t, http.StatusOK, rec.Code)

		var response services.AuthResponse
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.NotEmpty(t, response.AccessToken)
		assert.NotEqual(t, initialResponse.AccessToken, response.AccessToken)
		assert.NotEmpty(t, response.RefreshToken)
		assert.Equal(t, int64(15*60), response.ExpiresIn)
	})

	t.Run("invalid refresh token", func(t *testing.T) {
		refreshRequest := map[string]string{
			"refresh_token": "invalid-refresh-token",
		}

		reqBody, err := json.Marshal(refreshRequest)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBuffer(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()

		tc.Server.Echo.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

func TestAuthService_JWTValidation(t *testing.T) {
	tc := di.NewTestContainer(t)
	defer tc.Cleanup()

	randomEmail := fmt.Sprintf("jwt-%d@example.com", time.Now().UnixNano())

	setupUser := services.RegisterRequest{
		Email:            randomEmail,
		Password:         "ValidPass123!",
		FirstName:        "JWT",
		LastName:         "Test",
		OrganizationName: "JWT Org",
	}

	reqBody, _ := json.Marshal(setupUser)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	tc.Server.Echo.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var authResponse services.AuthResponse
	err := json.Unmarshal(rec.Body.Bytes(), &authResponse)
	require.NoError(t, err)

	jwtService := services.NewJWTService(tc.Config)

	t.Run("validate access token", func(t *testing.T) {
		claims, err := jwtService.ValidateAccessToken(authResponse.AccessToken)
		require.NoError(t, err)
		assert.Equal(t, authResponse.Account.ID, claims.AccountID)
		assert.Equal(t, authResponse.Organization.ID, claims.OrganizationID)
		assert.Equal(t, authResponse.Account.Email, claims.Email)
		assert.Equal(t, models.RoleOwner, claims.Role)
	})

	t.Run("validate refresh token", func(t *testing.T) {
		refreshTokenClaims, err := jwtService.ValidateRefreshToken(authResponse.RefreshToken)
		require.NoError(t, err)
		assert.Equal(t, authResponse.Account.ID, refreshTokenClaims.AccountID)
	})

	t.Run("invalid token format", func(t *testing.T) {
		_, err := jwtService.ValidateAccessToken("invalid.token.format")
		assert.Error(t, err)
	})

	t.Run("token expiry check", func(t *testing.T) {
		expiry, err := jwtService.GetTokenExpiry(authResponse.AccessToken)
		require.NoError(t, err)
		assert.True(t, expiry.After(time.Now()))
		assert.True(t, expiry.Before(time.Now().Add(16*time.Minute))) // Should expire in ~15 minutes

		isExpired := jwtService.IsTokenExpired(authResponse.AccessToken)
		assert.False(t, isExpired)
	})
}
