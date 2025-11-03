package fixtures

import (
	"context"
	"fmt"
	"testing"
	"time"

	"zori/di"
	"zori/internal/storage/postgres/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type ProjectFixture struct {
	ID             string
	Name           string
	Domain         string
	ProjectToken   string
	OrganizationID string
	AllowLocalHost bool
}

func CreateProject(t *testing.T, tc *di.TestContainer, account *AccountFixture, name string, domain string, allowLocalhost bool) *ProjectFixture {
	t.Helper()

	ctx := context.Background()
	projectID := uuid.New().String()
	projectToken := fmt.Sprintf("test_token_%d", time.Now().UnixNano())

	project := &models.Project{
		ID:             projectID,
		Name:           name,
		Domain:         domain,
		ProjectToken:   projectToken,
		OrganizationID: account.OrgID,
		AllowLocalHost: allowLocalhost,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	_, err := tc.DB.DB.NewInsert().Model(project).Exec(ctx)
	require.NoError(t, err, "Failed to create project")

	return &ProjectFixture{
		ID:             project.ID,
		Name:           project.Name,
		Domain:         project.Domain,
		ProjectToken:   project.ProjectToken,
		OrganizationID: project.OrganizationID,
		AllowLocalHost: project.AllowLocalHost,
	}
}

func SetupAccountAndProject(t *testing.T, tc *di.TestContainer) (*AccountFixture, *ProjectFixture) {
	t.Helper()

	account := CreateAccount(t, tc)
	project := CreateProject(
		t,
		tc,
		account,
		fmt.Sprintf("Test Project %d", time.Now().UnixNano()),
		"https://example.com",
		true,
	)

	return account, project
}
