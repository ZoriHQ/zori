package fixtures

import (
	"fmt"
	"testing"
	"time"

	"zori/di"

	"github.com/google/uuid"
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

// CreateAccount creates a mock account fixture for testing.
// Note: Accounts are managed by Stack Auth (external auth provider).
// This function only returns mock data - no database records are created.
func CreateAccount(t *testing.T, tc *di.TestContainer) *AccountFixture {
	t.Helper()

	randomEmail := fmt.Sprintf("test-%d@example.com", time.Now().UnixNano())
	orgName := fmt.Sprintf("Test Org %d", time.Now().UnixNano())
	accountID := uuid.New().String()
	orgID := uuid.New().String()
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
