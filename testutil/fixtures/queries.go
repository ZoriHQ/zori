package fixtures

import (
	"context"
	"fmt"
	"testing"
	"time"

	"zori/di"
	"zori/internal/storage/clickhouse/models"
	"zori/testutil"

	"github.com/stretchr/testify/require"
)

// QueryEventsOptions holds options for querying events
type QueryEventsOptions struct {
	ProjectID      *string
	OrganizationID *string
	VisitorID      *string
	SessionID      *string
	EventName      *string
	Limit          int
}

// QueryEvents retrieves events from ClickHouse based on the provided options
func QueryEvents(t *testing.T, tc *di.TestContainer, opts QueryEventsOptions) ([]models.Event, error) {
	t.Helper()

	// Select explicit columns to avoid issues with materialized columns
	query := `SELECT
		event_name, client_generated_event_id, visitor_id, session_id,
		user_id, external_id, email_hash,
		client_timestamp_utc,
		user_agent, ip, referrer_url, page_url, page_path, host,
		referrer_domain, referrer_path,
		browser_name, os_name, device_type,
		click_element_tag, click_element_selector, click_element_text,
		click_position_x, click_position_y, click_screen_width, click_screen_height,
		utm_parameters, custom_properties,
		project_id, organization_id,
		location_country_iso, location_city, location_latitude, location_longitude
	FROM events WHERE 1=1`
	args := []interface{}{}

	if opts.ProjectID != nil {
		query += " AND project_id = ?"
		args = append(args, *opts.ProjectID)
	}

	if opts.OrganizationID != nil {
		query += " AND organization_id = ?"
		args = append(args, *opts.OrganizationID)
	}

	if opts.VisitorID != nil {
		query += " AND visitor_id = ?"
		args = append(args, *opts.VisitorID)
	}

	if opts.SessionID != nil {
		query += " AND session_id = ?"
		args = append(args, *opts.SessionID)
	}

	if opts.EventName != nil {
		query += " AND event_name = ?"
		args = append(args, *opts.EventName)
	}

	query += " ORDER BY client_timestamp_utc DESC"

	if opts.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", opts.Limit)
	}

	var events []models.Event
	err := tc.ClickHouse.Db().Select(context.Background(), &events, query, args...)
	if err != nil {
		return nil, err
	}

	return events, nil
}

// WaitForEvents waits for events to appear in ClickHouse with the given options
// Returns the events once they appear or times out after the specified duration
func WaitForEvents(t *testing.T, tc *di.TestContainer, opts QueryEventsOptions, expectedCount int, timeout time.Duration) ([]models.Event, error) {
	t.Helper()

	var events []models.Event
	ctx := context.Background()

	err := testutil.WaitForCondition(ctx, timeout, 100*time.Millisecond, func() (bool, error) {
		var err error
		events, err = QueryEvents(t, tc, opts)
		if err != nil {
			return false, err
		}

		return len(events) >= expectedCount, nil
	})

	if err != nil {
		return events, fmt.Errorf("timeout waiting for events: %w (got %d events, expected %d)", err, len(events), expectedCount)
	}

	return events, nil
}

// CountEvents counts the number of events matching the given options
func CountEvents(t *testing.T, tc *di.TestContainer, opts QueryEventsOptions) (int, error) {
	t.Helper()

	query := "SELECT count(*) FROM events WHERE 1=1"
	args := []interface{}{}

	if opts.ProjectID != nil {
		query += " AND project_id = ?"
		args = append(args, *opts.ProjectID)
	}

	if opts.OrganizationID != nil {
		query += " AND organization_id = ?"
		args = append(args, *opts.OrganizationID)
	}

	if opts.VisitorID != nil {
		query += " AND visitor_id = ?"
		args = append(args, *opts.VisitorID)
	}

	if opts.SessionID != nil {
		query += " AND session_id = ?"
		args = append(args, *opts.SessionID)
	}

	if opts.EventName != nil {
		query += " AND event_name = ?"
		args = append(args, *opts.EventName)
	}

	var count uint64
	err := tc.ClickHouse.Db().QueryRow(context.Background(), query, args...).Scan(&count)
	if err != nil {
		return 0, err
	}

	return int(count), nil
}

// AssertEventExists asserts that at least one event exists matching the options
func AssertEventExists(t *testing.T, tc *di.TestContainer, opts QueryEventsOptions) {
	t.Helper()

	events, err := QueryEvents(t, tc, opts)
	require.NoError(t, err)
	require.NotEmpty(t, events, "Expected at least one event to exist")
}

// AssertEventCount asserts that the exact number of events exist matching the options
func AssertEventCount(t *testing.T, tc *di.TestContainer, opts QueryEventsOptions, expectedCount int) {
	t.Helper()

	count, err := CountEvents(t, tc, opts)
	require.NoError(t, err)
	require.Equal(t, expectedCount, count, "Event count mismatch")
}
