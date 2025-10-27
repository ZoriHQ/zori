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

// CreateAccount creates a new account with organization directly in the database
// NOTE: This bypasses Stack Auth for testing purposes
func CreateAccount(t *testing.T, tc *di.TestContainer) *AccountFixture {
	t.Helper()

	randomEmail := fmt.Sprintf("test-%d@example.com", time.Now().UnixNano())
	orgName := fmt.Sprintf("Test Org %d", time.Now().UnixNano())
	accountID := uuid.New().String()
	orgID := uuid.New().String()

	ctx := context.Background()

	// Create account directly in database
	account := &models.Account{
		ID:        accountID,
		Email:     randomEmail,
		FirstName: "Test",
		LastName:  "User",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_, err := tc.DB.DB.NewInsert().Model(account).Exec(ctx)
	require.NoError(t, err, "Failed to create account")

	// Create organization directly in database
	org := &models.Organization{
		ID:        orgID,
		Name:      orgName,
		Slug:      fmt.Sprintf("test-org-%d", time.Now().UnixNano()),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_, err = tc.DB.DB.NewInsert().Model(org).Exec(ctx)
	require.NoError(t, err, "Failed to create organization")

	// Create organization membership
	member := &models.OrganizationMember{
		ID:             uuid.New().String(),
		OrganizationID: orgID,
		AccountID:      accountID,
		Role:           models.RoleOwner,
		JoinedAt:       time.Now(),
	}
	_, err = tc.DB.DB.NewInsert().Model(member).Exec(ctx)
	require.NoError(t, err, "Failed to create organization member")

	// For Stack Auth integration, we use the accountID as a mock token
	// In real tests, you would use actual Stack Auth tokens
	mockToken := accountID

	return &AccountFixture{
		Email:        randomEmail,
		Password:     "test-password",
		AccessToken:  mockToken,
		RefreshToken: "",
		AccountID:    accountID,
		OrgID:        orgID,
		OrgName:      orgName,
	}
}
