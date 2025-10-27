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

func CreateAccount(t *testing.T, tc *di.TestContainer) *AccountFixture {
	t.Helper()

	randomEmail := fmt.Sprintf("test-%d@example.com", time.Now().UnixNano())
	orgName := fmt.Sprintf("Test Org %d", time.Now().UnixNano())
	accountID := uuid.New().String()
	orgID := uuid.New().String()

	ctx := context.Background()

	// Create organization first (required for foreign key constraint)
	org := &models.Organization{
		ID:        orgID,
		Name:      orgName,
		Slug:      fmt.Sprintf("test-org-%d", time.Now().UnixNano()),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_, err := tc.DB.DB.NewInsert().Model(org).Exec(ctx)
	require.NoError(t, err, "Failed to create organization")

	// Now create account
	account := &models.Account{
		ID:        accountID,
		Email:     randomEmail,
		FirstName: "Test",
		LastName:  "User",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_, err = tc.DB.DB.NewInsert().Model(account).Exec(ctx)
	require.NoError(t, err, "Failed to create account")

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
