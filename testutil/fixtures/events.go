package fixtures

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"zori/di"
	"zori/services/ingestion/types"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// EventBuilder provides a fluent API to build test events
type EventBuilder struct {
	event types.ClientEventV1
}

// NewEventBuilder creates a new EventBuilder with sensible defaults
func NewEventBuilder() *EventBuilder {
	visitorID := uuid.New().String()
	sessionID := uuid.New().String()

	return &EventBuilder{
		event: types.ClientEventV1{
			ClientGeneratedEventID: uuid.New().String(),
			VisitorID:              visitorID,
			SessionID:              sessionID,
			ClientTimeStampUTC:     time.Now().UTC(),
			UserAgent:              "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36",
			IP:                     "127.0.0.1",
			PageURL:                "/",
			Host:                   "example.com",
			UTMParameters:          make(map[string]string),
			CustomProperties:       make(map[string]any),
		},
	}
}

// WithEventName sets the event name
func (b *EventBuilder) WithEventName(name string) *EventBuilder {
	b.event.EventName = &name
	return b
}

// WithVisitorID sets the visitor ID
func (b *EventBuilder) WithVisitorID(visitorID string) *EventBuilder {
	b.event.VisitorID = visitorID
	return b
}

// WithSessionID sets the session ID
func (b *EventBuilder) WithSessionID(sessionID string) *EventBuilder {
	b.event.SessionID = sessionID
	return b
}

// WithPageURL sets the page URL
func (b *EventBuilder) WithPageURL(url string) *EventBuilder {
	b.event.PageURL = url
	return b
}

// WithHost sets the host
func (b *EventBuilder) WithHost(host string) *EventBuilder {
	b.event.Host = host
	return b
}

// WithReferrer sets the referrer
func (b *EventBuilder) WithReferrer(referrer string) *EventBuilder {
	b.event.Referrer = &referrer
	return b
}

// WithUTMSource sets the UTM source
func (b *EventBuilder) WithUTMSource(source string) *EventBuilder {
	b.event.UTMParameters["utm_source"] = source
	return b
}

// WithUTMMedium sets the UTM medium
func (b *EventBuilder) WithUTMMedium(medium string) *EventBuilder {
	b.event.UTMParameters["utm_medium"] = medium
	return b
}

// WithUTMCampaign sets the UTM campaign
func (b *EventBuilder) WithUTMCampaign(campaign string) *EventBuilder {
	b.event.UTMParameters["utm_campaign"] = campaign
	return b
}

// WithCustomProperty adds a custom property
func (b *EventBuilder) WithCustomProperty(key string, value any) *EventBuilder {
	b.event.CustomProperties[key] = value
	return b
}

// WithClickElement sets click element information
func (b *EventBuilder) WithClickElement(tag, selector, text string) *EventBuilder {
	b.event.ClickElement = &types.ClickElement{
		Tag:      tag,
		Selector: selector,
		Text:     text,
	}
	return b
}

// WithClickPosition sets click position information
func (b *EventBuilder) WithClickPosition(x, y float64, width, height uint16) *EventBuilder {
	b.event.ClickPosition = &types.ClickPosition{
		X:            x,
		Y:            y,
		ScreenWidth:  width,
		ScreenHeight: height,
	}
	return b
}

// WithIdentity sets identity information
func (b *EventBuilder) WithIdentity(userID, externalID, email string) *EventBuilder {
	if userID != "" {
		b.event.UserID = &userID
	}
	if externalID != "" {
		b.event.ExternalID = &externalID
	}
	if email != "" {
		b.event.Email = &email
	}
	return b
}

// Build returns the built event
func (b *EventBuilder) Build() types.ClientEventV1 {
	return b.event
}

// SendEvent sends an event to the ingestion endpoint via HTTP
func SendEvent(t *testing.T, ingestionURL string, projectToken string, event types.ClientEventV1) error {
	t.Helper()

	eventJSON, err := json.Marshal(event)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, ingestionURL+"/ingest", bytes.NewBuffer(eventJSON))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Zori-PT", projectToken)
	req.Header.Set("User-Agent", event.UserAgent)

	// Set visitor_id cookie
	req.AddCookie(&http.Cookie{
		Name:  "visitor_id",
		Value: event.VisitorID,
	})

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send event: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// SendEventToTestServer sends an event using the test container's ingestion server
func SendEventToTestServer(t *testing.T, tc *di.TestContainer, project *ProjectFixture, event types.ClientEventV1) error {
	t.Helper()
	return SendEvent(t, tc.IngestionServerURL, project.ProjectToken, event)
}

// SendEvents sends multiple events to the ingestion endpoint
func SendEvents(t *testing.T, ingestionURL string, projectToken string, events []types.ClientEventV1) error {
	t.Helper()

	for i, event := range events {
		if err := SendEvent(t, ingestionURL, projectToken, event); err != nil {
			return fmt.Errorf("failed to send event %d: %w", i, err)
		}
		// Small delay between events to avoid overwhelming the system
		time.Sleep(10 * time.Millisecond)
	}

	return nil
}

// SendEventsToTestServer sends multiple events using the test container's ingestion server
func SendEventsToTestServer(t *testing.T, tc *di.TestContainer, project *ProjectFixture, events []types.ClientEventV1) error {
	t.Helper()
	return SendEvents(t, tc.IngestionServerURL, project.ProjectToken, events)
}
