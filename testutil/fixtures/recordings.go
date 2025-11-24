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

type RecordingEventBuilder struct {
	event types.RecordingEventV1
}

func NewRecordingEventBuilder() *RecordingEventBuilder {
	visitorID := uuid.New().String()
	sessionID := uuid.New().String()

	return &RecordingEventBuilder{
		event: types.RecordingEventV1{
			EventName:              "session_recording_events",
			ClientGeneratedEventID: uuid.New().String(),
			VisitorID:              visitorID,
			SessionID:              sessionID,
			ClientTimestampUTC:     time.Now().UTC(),
			PageURL:                "/",
			Host:                   "example.com",
			RecordingEvents:        make([]any, 0),
			EventCount:             0,
		},
	}
}

func (b *RecordingEventBuilder) WithVisitorID(visitorID string) *RecordingEventBuilder {
	b.event.VisitorID = visitorID
	return b
}

func (b *RecordingEventBuilder) WithSessionID(sessionID string) *RecordingEventBuilder {
	b.event.SessionID = sessionID
	return b
}

func (b *RecordingEventBuilder) WithPageURL(url string) *RecordingEventBuilder {
	b.event.PageURL = url
	return b
}

func (b *RecordingEventBuilder) WithHost(host string) *RecordingEventBuilder {
	b.event.Host = host
	return b
}

func (b *RecordingEventBuilder) WithRecordingEvents(events []any) *RecordingEventBuilder {
	b.event.RecordingEvents = events
	b.event.EventCount = uint32(len(events))
	return b
}

func (b *RecordingEventBuilder) Build() types.RecordingEventV1 {
	return b.event
}

func CreateSampleRRWebEvents(count int) []any {
	events := make([]any, count)
	baseTime := time.Now().UnixMilli()

	for i := 0; i < count; i++ {
		eventType := 3
		if i == 0 {
			eventType = 4
		}

		events[i] = map[string]any{
			"type":      eventType,
			"timestamp": baseTime + int64(i*100),
			"data": map[string]any{
				"source":    1,
				"positions": []map[string]any{{"x": 100 + i, "y": 200 + i, "id": 1, "timeOffset": 0}},
			},
		}
	}

	return events
}

func SendRecordingEvent(t *testing.T, ingestionURL string, projectToken string, event types.RecordingEventV1) error {
	t.Helper()

	eventJSON, err := json.Marshal(event)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, ingestionURL+"/recording", bytes.NewBuffer(eventJSON))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Zori-PT", projectToken)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Test)")

	req.AddCookie(&http.Cookie{
		Name:  "visitor_id",
		Value: event.VisitorID,
	})

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send recording event: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

func SendRecordingEventToTestServer(t *testing.T, tc *di.TestContainer, project *ProjectFixture, event types.RecordingEventV1) error {
	t.Helper()
	return SendRecordingEvent(t, tc.IngestionServerURL, project.ProjectToken, event)
}

func SendRecordingEvents(t *testing.T, ingestionURL string, projectToken string, events []types.RecordingEventV1) error {
	t.Helper()

	for i, event := range events {
		if err := SendRecordingEvent(t, ingestionURL, projectToken, event); err != nil {
			return fmt.Errorf("failed to send recording event %d: %w", i, err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	return nil
}

func SendRecordingEventsToTestServer(t *testing.T, tc *di.TestContainer, project *ProjectFixture, events []types.RecordingEventV1) error {
	t.Helper()
	return SendRecordingEvents(t, tc.IngestionServerURL, project.ProjectToken, events)
}
