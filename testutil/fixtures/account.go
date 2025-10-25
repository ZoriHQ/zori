package fixtures

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"zori/di"
	"zori/services/auth/services"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

// AccountFixture holds the created account, organization, and auth tokens
type AccountFixture struct {
	Email        string
	Password     string
	AccessToken  string
	RefreshToken string
	AccountID    string
	OrgID        string
	OrgName      string
}

// CreateAccount creates a new account with organization via the register endpoint
func CreateAccount(t *testing.T, tc *di.TestContainer) *AccountFixture {
	t.Helper()

	randomEmail := fmt.Sprintf("test-%d@example.com", time.Now().UnixNano())
	password := "ValidPass123!"
	orgName := fmt.Sprintf("Test Org %d", time.Now().UnixNano())

	registerReq := services.RegisterRequest{
		Email:            randomEmail,
		Password:         password,
		FirstName:        "Test",
		LastName:         "User",
		OrganizationName: orgName,
	}

	reqBody, err := json.Marshal(registerReq)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	tc.Server.Echo.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, "Failed to register account: %s", rec.Body.String())

	var authResponse services.AuthResponse
	err = json.Unmarshal(rec.Body.Bytes(), &authResponse)
	require.NoError(t, err)

	return &AccountFixture{
		Email:        randomEmail,
		Password:     password,
		AccessToken:  authResponse.AccessToken,
		RefreshToken: authResponse.RefreshToken,
		AccountID:    authResponse.Account.ID,
		OrgID:        authResponse.Organization.ID,
		OrgName:      authResponse.Organization.Name,
	}
}
