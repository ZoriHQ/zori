package fixtures

import (
	"context"
	"fmt"
	"testing"
	"time"

	"zori/di"
	appctx "zori/internal/ctx"

	"github.com/google/uuid"
)

// NewTestCtx creates a test context with organization ID
func NewTestCtx(orgID string) *appctx.Ctx {
	return &appctx.Ctx{
		Context: context.Background(),
	}
}

// NewTestCtxWithOrg creates a test context and sets the organization ID
func NewTestCtxWithOrg(orgID string) *appctx.Ctx {
	ctx := NewTestCtx(orgID)
	ctx.SetOrgID(orgID)
	return ctx
}

type EventInsertData struct {
	EventName          *string
	VisitorID          string
	SessionID          string
	ClientTimestampUTC time.Time
	UserAgent          string
	IP                 string
	ReferrerURL        *string
	ReferrerDomain     *string
	PageURL            string
	PagePath           string
	Host               string
	BrowserName        *string
	OSName             *string
	DeviceType         *string
	LocationCountryISO *string
	LocationCity       *string
	UTMSource          *string
	UTMMedium          *string
	UTMCampaign        *string
	ProjectID          string
	OrganizationID     string
	CustomProperties   map[string]interface{}
}

func InsertEventsDirect(t *testing.T, tc *di.TestContainer, events []EventInsertData) error {
	t.Helper()

	ctx := context.Background()
	query := `INSERT INTO events (
		event_name, client_generated_event_id, visitor_id, session_id,
		client_timestamp_utc, user_agent, ip,
		referrer_url, referrer_domain,
		page_url, page_path, host,
		browser_name, os_name, device_type,
		location_country_iso, location_city,
		utm_parameters,
		project_id, organization_id,
		custom_properties,
		created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	for i, e := range events {
		utmParams := make(map[string]string)
		if e.UTMSource != nil {
			utmParams["utm_source"] = *e.UTMSource
		}
		if e.UTMMedium != nil {
			utmParams["utm_medium"] = *e.UTMMedium
		}
		if e.UTMCampaign != nil {
			utmParams["utm_campaign"] = *e.UTMCampaign
		}

		customProps := e.CustomProperties
		if customProps == nil {
			customProps = make(map[string]interface{})
		}

		err := tc.ClickHouse.Db().Exec(ctx, query,
			e.EventName,
			uuid.New().String(),
			e.VisitorID,
			e.SessionID,
			e.ClientTimestampUTC,
			e.UserAgent,
			e.IP,
			e.ReferrerURL,
			e.ReferrerDomain,
			e.PageURL,
			e.PagePath,
			e.Host,
			e.BrowserName,
			e.OSName,
			e.DeviceType,
			e.LocationCountryISO,
			e.LocationCity,
			utmParams,
			e.ProjectID,
			e.OrganizationID,
			customProps,
			e.ClientTimestampUTC, // created_at should match client_timestamp_utc for test data
		)
		if err != nil {
			return fmt.Errorf("failed to insert event %d: %w", i, err)
		}
	}

	return nil
}

func InsertPaymentEventDirect(t *testing.T, tc *di.TestContainer, projectID, organizationID string, visitorID *string, amount int64, timestamp time.Time) error {
	t.Helper()

	ctx := context.Background()
	query := `INSERT INTO payment_events (
		payment_id, visitor_id, provider_type, payment_status, product_name,
		amount, currency, payment_timestamp_utc, project_id, organization_id, metadata, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	err := tc.ClickHouse.Db().Exec(ctx, query,
		uuid.New().String(),
		visitorID,
		"stripe",
		"succeeded",
		"Test Product",
		amount,
		"USD",
		timestamp,
		projectID,
		organizationID,
		make(map[string]string),
		timestamp, // created_at should match the payment timestamp for the timeline query
	)
	if err != nil {
		return fmt.Errorf("failed to insert payment event: %w", err)
	}

	return nil
}

func NewEventInsertData(projectID, organizationID string) EventInsertData {
	return EventInsertData{
		VisitorID:          uuid.New().String(),
		SessionID:          uuid.New().String(),
		ClientTimestampUTC: time.Now().UTC(),
		UserAgent:          "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
		IP:                 "127.0.0.1",
		PageURL:            "/",
		PagePath:           "/",
		Host:               "example.com",
		ProjectID:          projectID,
		OrganizationID:     organizationID,
		CustomProperties:   make(map[string]interface{}),
	}
}

