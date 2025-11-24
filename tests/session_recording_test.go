package tests

import (
	"context"
	"testing"
	"time"

	"zori/di"
	"zori/testutil"
	"zori/testutil/fixtures"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionRecording_EndToEnd(t *testing.T) {
	tc := di.NewTestContainer(t)
	defer tc.Cleanup()

	_, project := fixtures.SetupAccountAndProject(t, tc)

	time.Sleep(200 * time.Millisecond)

	t.Run("should ingest and store session recording events", func(t *testing.T) {
		visitorID := "test-visitor-recording-123"
		sessionID := "test-session-recording-456"

		rrwebEvents := fixtures.CreateSampleRRWebEvents(10)

		recordingEvent := fixtures.NewRecordingEventBuilder().
			WithVisitorID(visitorID).
			WithSessionID(sessionID).
			WithPageURL("/dashboard").
			WithHost(project.Domain).
			WithRecordingEvents(rrwebEvents).
			Build()

		err := fixtures.SendRecordingEventToTestServer(t, tc, project, recordingEvent)
		require.NoError(t, err, "Failed to send recording event")

		ctx := context.Background()
		err = testutil.WaitForCondition(ctx, 10*time.Second, 100*time.Millisecond, func() (bool, error) {
			query := `
				SELECT count()
				FROM session_recordings
				WHERE project_id = ? AND session_id = ?
			`
			var count uint64
			err := tc.ClickHouse.Db().QueryRow(ctx, query, project.ID, sessionID).Scan(&count)
			if err != nil {
				return false, err
			}
			return count > 0, nil
		})
		require.NoError(t, err, "Recording should appear in ClickHouse")

		var (
			storedSessionID string
			storedVisitorID string
			eventCount      uint32
			pageURL         string
			host            string
		)

		query := `
			SELECT session_id, visitor_id, event_count, page_url, host
			FROM session_recordings
			WHERE project_id = ? AND session_id = ?
			LIMIT 1
		`
		row := tc.ClickHouse.Db().QueryRow(ctx, query, project.ID, sessionID)
		err = row.Scan(&storedSessionID, &storedVisitorID, &eventCount, &pageURL, &host)
		require.NoError(t, err, "Should be able to query recording")

		assert.Equal(t, sessionID, storedSessionID)
		assert.Equal(t, visitorID, storedVisitorID)
		assert.Equal(t, uint32(10), eventCount)
		assert.Equal(t, "/dashboard", pageURL)
		assert.Equal(t, project.Domain, host)

		t.Logf("Successfully stored recording with %d events", eventCount)
	})

	t.Run("should handle multiple recording chunks for same session", func(t *testing.T) {
		visitorID := "test-visitor-chunks-123"
		sessionID := "test-session-chunks-456"

		for i := 0; i < 3; i++ {
			rrwebEvents := fixtures.CreateSampleRRWebEvents(5)

			recordingEvent := fixtures.NewRecordingEventBuilder().
				WithVisitorID(visitorID).
				WithSessionID(sessionID).
				WithPageURL("/page-" + string(rune('a'+i))).
				WithHost(project.Domain).
				WithRecordingEvents(rrwebEvents).
				Build()

			err := fixtures.SendRecordingEventToTestServer(t, tc, project, recordingEvent)
			require.NoError(t, err, "Failed to send recording chunk %d", i)

			time.Sleep(50 * time.Millisecond)
		}

		ctx := context.Background()
		err := testutil.WaitForCondition(ctx, 10*time.Second, 100*time.Millisecond, func() (bool, error) {
			query := `
				SELECT count()
				FROM session_recordings
				WHERE project_id = ? AND session_id = ?
			`
			var count uint64
			err := tc.ClickHouse.Db().QueryRow(ctx, query, project.ID, sessionID).Scan(&count)
			if err != nil {
				return false, err
			}
			return count >= 3, nil
		})
		require.NoError(t, err, "All recording chunks should appear in ClickHouse")

		var totalEvents uint64
		sumQuery := `
			SELECT sum(event_count)
			FROM session_recordings
			WHERE project_id = ? AND session_id = ?
		`
		row := tc.ClickHouse.Db().QueryRow(ctx, sumQuery, project.ID, sessionID)
		err = row.Scan(&totalEvents)
		require.NoError(t, err)

		assert.Equal(t, uint64(15), totalEvents, "Should have 15 total events across 3 chunks")

		t.Logf("Successfully stored 3 recording chunks with %d total events", totalEvents)
	})
}

func TestSessionRecording_Validation(t *testing.T) {
	tc := di.NewTestContainer(t)
	defer tc.Cleanup()

	_, project := fixtures.SetupAccountAndProject(t, tc)

	time.Sleep(200 * time.Millisecond)

	t.Run("should reject recording without session_id", func(t *testing.T) {
		recordingEvent := fixtures.NewRecordingEventBuilder().
			WithSessionID("").
			WithHost(project.Domain).
			Build()

		err := fixtures.SendRecordingEventToTestServer(t, tc, project, recordingEvent)
		assert.Error(t, err, "Should reject recording without session_id")
	})

	t.Run("should reject recording with mismatched visitor_id", func(t *testing.T) {
		recordingEvent := fixtures.NewRecordingEventBuilder().
			WithVisitorID("visitor-in-payload").
			WithHost(project.Domain).
			Build()

		recordingEvent.VisitorID = "different-visitor-id"

		err := fixtures.SendRecordingEventToTestServer(t, tc, project, recordingEvent)
		assert.Error(t, err, "Should reject recording with mismatched visitor_id")
	})
}

func TestSessionRecording_BuilderHelpers(t *testing.T) {
	t.Run("NewRecordingEventBuilder creates valid events", func(t *testing.T) {
		event := fixtures.NewRecordingEventBuilder().
			WithVisitorID("test-visitor").
			WithSessionID("test-session").
			WithPageURL("/test-page").
			WithHost("test.example.com").
			Build()

		assert.Equal(t, "test-visitor", event.VisitorID)
		assert.Equal(t, "test-session", event.SessionID)
		assert.Equal(t, "/test-page", event.PageURL)
		assert.Equal(t, "test.example.com", event.Host)
		assert.NotEmpty(t, event.ClientGeneratedEventID)
	})

	t.Run("CreateSampleRRWebEvents creates valid rrweb events", func(t *testing.T) {
		events := fixtures.CreateSampleRRWebEvents(5)

		assert.Len(t, events, 5)

		firstEvent, ok := events[0].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, 4, firstEvent["type"], "First event should be meta type")

		secondEvent, ok := events[1].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, 3, secondEvent["type"], "Subsequent events should be incremental type")
	})

	t.Run("WithRecordingEvents sets event count correctly", func(t *testing.T) {
		events := fixtures.CreateSampleRRWebEvents(10)
		recording := fixtures.NewRecordingEventBuilder().
			WithRecordingEvents(events).
			Build()

		assert.Equal(t, uint32(10), recording.EventCount)
		assert.Len(t, recording.RecordingEvents, 10)
	})
}
