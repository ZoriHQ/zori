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
	"zori/internal/storage/postgres/models"
	"zori/services/projects/types"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

// ProjectFixture holds the created project details
type ProjectFixture struct {
	ID             string
	Name           string
	Domain         string
	ProjectToken   string
	OrganizationID string
	AllowLocalHost bool
}

// CreateProject creates a new project for the given account
func CreateProject(t *testing.T, tc *di.TestContainer, account *AccountFixture, name string, domain string, allowLocalhost bool) *ProjectFixture {
	t.Helper()

	createReq := types.CreateProjectRequest{
		Name:           name,
		WebsiteURL:     domain,
		AllowLocalHost: allowLocalhost,
	}

	reqBody, err := json.Marshal(createReq)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", bytes.NewBuffer(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Authorization", "Bearer "+account.AccessToken)
	rec := httptest.NewRecorder()

	tc.Server.Echo.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code, "Failed to create project: %s", rec.Body.String())

	var project models.Project
	err = json.Unmarshal(rec.Body.Bytes(), &project)
	require.NoError(t, err)

	return &ProjectFixture{
		ID:             project.ID,
		Name:           project.Name,
		Domain:         project.Domain,
		ProjectToken:   project.ProjectToken,
		OrganizationID: project.OrganizationID,
		AllowLocalHost: project.AllowLocalHost,
	}
}

// SetupAccountAndProject is a convenience function that creates both account and project
func SetupAccountAndProject(t *testing.T, tc *di.TestContainer) (*AccountFixture, *ProjectFixture) {
	t.Helper()

	account := CreateAccount(t, tc)
	project := CreateProject(
		t,
		tc,
		account,
		fmt.Sprintf("Test Project %d", time.Now().UnixNano()),
		"https://example.com",
		true, // Allow localhost for testing
	)

	return account, project
}
