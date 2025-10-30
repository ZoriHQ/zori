package init

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
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

	defaultOrgID := uuid.New().String()

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

	// Display credentials prominently in logs
	o.displayCredentials(username, password, defaultOrgID)

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
			// Create a new system record if none exists
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

	// Pad the API host to fit the format
	apiHost := o.cfg.ZoriAPIHost
	if len(apiHost) > 30 {
		apiHost = apiHost[:27] + "..."
	}

	formattedBanner := fmt.Sprintf(banner, username, password, orgID, padRight(apiHost, 30))

	fmt.Println(formattedBanner)
}

// padRight pads a string with spaces on the right to reach the desired length
func padRight(s string, length int) string {
	if len(s) >= length {
		return s
	}
	return s + strings.Repeat(" ", length-len(s))
}
