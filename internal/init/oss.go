package init

import (
	"bufio"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"zori/internal/config"
	"zori/internal/storage/postgres"
	"zori/internal/storage/postgres/models"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	passwordLength = 16
	usernamePrefix = "admin_"
)

type OSSInitializer struct {
	cfg *config.Config
	db  *postgres.PostgresDB
}

func NewOSSInitializer(cfg *config.Config, db *postgres.PostgresDB) *OSSInitializer {
	return &OSSInitializer{
		cfg: cfg,
		db:  db,
	}
}

func (o *OSSInitializer) InitializeOrReset(ctx context.Context, resetAuth bool) error {
	if resetAuth {
		return o.ResetAuth(ctx)
	}
	return o.Initialize(ctx)
}

func (o *OSSInitializer) Initialize(ctx context.Context) error {
	if !o.cfg.ZoriOSS {
		return nil
	}

	fmt.Println("===========================================")
	fmt.Println("OSS MODE DETECTED - Checking initialization")
	fmt.Println("===========================================")

	system, err := o.getSystemConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get system config: %w", err)
	}

	if system.AdminUsername != nil && *system.AdminUsername != "" {
		fmt.Println("System already initialized with admin credentials")
		fmt.Println("===========================================")
		return nil
	}

	username, password, err := o.generateCredentials()
	if err != nil {
		return fmt.Errorf("failed to generate credentials: %w", err)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Generate Clerk-compatible org ID (format: org_xxxxxxxxxxxxxxxxxxxxxxxxxx)
	defaultOrgID := o.generateClerkCompatibleOrgID()

	hashedPasswordStr := string(hashedPassword)
	defaultOrgIDStr := defaultOrgID
	system.AdminUsername = &username
	system.AdminPasswordHash = &hashedPasswordStr
	system.DefaultOrgID = &defaultOrgIDStr

	_, err = o.db.DB.NewUpdate().
		Model(system).
		Column("admin_username", "admin_password_hash", "default_org_id").
		Where("id = ?", system.ID).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to update system config: %w", err)
	}

	o.displayCredentials(username, password, defaultOrgID)

	return nil
}

func (o *OSSInitializer) ResetAuth(ctx context.Context) error {
	if !o.cfg.ZoriOSS {
		return nil
	}

	fmt.Println("===========================================")
	fmt.Println("OSS AUTH RESET MODE")
	fmt.Println("===========================================")

	system, err := o.getSystemConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get system config: %w", err)
	}

	if system.AdminUsername == nil || *system.AdminUsername == "" {
		fmt.Println("No existing credentials found. Nothing to reset.")
		fmt.Println("===========================================")
		return nil
	}

	fmt.Print("\nAre you sure you want to reset admin credentials? This will invalidate the current username and password. (y/n): ")
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read user input: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response != "y" && response != "yes" {
		fmt.Println("Reset cancelled. Starting server with existing credentials...")
		fmt.Println("===========================================")
		return nil
	}

	username, password, err := o.generateCredentials()
	if err != nil {
		return fmt.Errorf("failed to generate credentials: %w", err)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	hashedPasswordStr := string(hashedPassword)
	system.AdminUsername = &username
	system.AdminPasswordHash = &hashedPasswordStr

	_, err = o.db.DB.NewUpdate().
		Model(system).
		Column("admin_username", "admin_password_hash").
		Where("id = ?", system.ID).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to update system config: %w", err)
	}

	fmt.Println("\n✓ Credentials reset successfully!")

	orgID := ""
	if system.DefaultOrgID != nil {
		orgID = *system.DefaultOrgID
	}
	o.displayCredentials(username, password, orgID)

	return nil
}

func (o *OSSInitializer) getSystemConfig(ctx context.Context) (*models.System, error) {
	system := &models.System{}
	err := o.db.DB.NewSelect().
		Model(system).
		Limit(1).
		Scan(ctx)

	if err != nil {
		if err == sql.ErrNoRows {
			system.ID = uuid.New().String()
			_, err = o.db.DB.NewInsert().
				Model(system).
				Exec(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to create system record: %w", err)
			}
			return system, nil
		}
		return nil, err
	}

	return system, nil
}

func (o *OSSInitializer) generateCredentials() (username, password string, err error) {
	randomBytes := make([]byte, 6)
	_, err = rand.Read(randomBytes)
	if err != nil {
		return "", "", err
	}
	username = usernamePrefix + base64.URLEncoding.EncodeToString(randomBytes)[:8]

	passwordBytes := make([]byte, passwordLength)
	_, err = rand.Read(passwordBytes)
	if err != nil {
		return "", "", err
	}
	password = base64.URLEncoding.EncodeToString(passwordBytes)[:passwordLength]

	return username, password, nil
}

// generateClerkCompatibleOrgID generates an organization ID in Clerk's format
// Format: org_xxxxxxxxxxxxxxxxxxxxxxxxxx (org_ prefix + 26 random alphanumeric characters)
func (o *OSSInitializer) generateClerkCompatibleOrgID() string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	const idLength = 26

	randomBytes := make([]byte, idLength)
	_, err := rand.Read(randomBytes)
	if err != nil {
		// Fallback to UUID-based generation if random fails
		return "org_" + strings.ReplaceAll(uuid.New().String(), "-", "")[:idLength]
	}

	// Convert random bytes to alphanumeric characters
	id := make([]byte, idLength)
	for i := 0; i < idLength; i++ {
		id[i] = charset[randomBytes[i]%byte(len(charset))]
	}

	return "org_" + string(id)
}

func (o *OSSInitializer) displayCredentials(username, password, orgID string) {
	banner := `
╔═══════════════════════════════════════════════════════════════╗
║                                                               ║
║              ZORI OSS - INITIAL SETUP COMPLETE                ║
║                                                               ║
╠═══════════════════════════════════════════════════════════════╣
║                                                               ║
║  ⚠️  IMPORTANT: Save these credentials securely!              ║
║     They will NOT be shown again!                             ║
║                                                               ║
║  Admin Username: %-44s ║
║  Admin Password: %-44s ║
║                                                               ║
║                                                               ║
║                                                               ║
╚═══════════════════════════════════════════════════════════════╝
`

	apiHost := o.cfg.ZoriAPIHost
	if len(apiHost) > 30 {
		apiHost = apiHost[:27] + "..."
	}

	formattedBanner := fmt.Sprintf(banner, username, password)

	fmt.Println(formattedBanner)
}
