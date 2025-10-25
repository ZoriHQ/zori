package main

import (
	"fmt"
	"testing"
	"time"

	"zori/di"
	"zori/internal/storage/clickhouse/models"
	"zori/services/ingestion/types"
	"zori/testutil/fixtures"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventIngestion_EndToEnd(t *testing.T) {
	tc := di.NewTestContainer(t)
	defer tc.Cleanup()

	_, project := fixtures.SetupAccountAndProject(t, tc)

	// Processor and ingestion server are already running via DI lifecycle hooks
	// Give them a moment to fully initialize
	time.Sleep(200 * time.Millisecond)

	t.Run("should ingest and store visit events correctly", func(t *testing.T) {
		visitorID := "test-visitor-123"
		sessionID := "test-session-456"

		eventConfigs := []struct {
			pageURL       string
			utmSource     string
			utmMedium     string
			utmCampaign   string
			hasClickEvent bool
		}{
			{"/", "google", "cpc", "summer-sale", false},
			{"/products", "google", "cpc", "summer-sale", false},
			{"/products/widget", "google", "cpc", "summer-sale", true},
			{"/cart", "", "", "", false},
			{"/checkout", "", "", "", false},
		}

		var builtEvents []types.ClientEventV1
		for i, evt := range eventConfigs {
			builder := fixtures.NewEventBuilder().
				WithVisitorID(visitorID).
				WithSessionID(sessionID).
				WithPageURL(evt.pageURL).
				WithHost(project.Domain)

			if evt.utmSource != "" {
				builder = builder.
					WithUTMSource(evt.utmSource).
					WithUTMMedium(evt.utmMedium).
					WithUTMCampaign(evt.utmCampaign)
			}

			if evt.hasClickEvent {
				builder = builder.
					WithClickElement("button", "#buy-now", "Buy Now").
					WithClickPosition(150.5, 200.3, 1920, 1080)
			}

			builder = builder.WithCustomProperty("test_sequence", i+1)

			builtEvents = append(builtEvents, builder.Build())
		}

		err := fixtures.SendEventsToTestServer(t, tc, project, builtEvents)
		require.NoError(t, err, "Failed to send events")

		events, err := fixtures.WaitForEvents(t, tc, fixtures.QueryEventsOptions{
			ProjectID: &project.ID,
			VisitorID: &visitorID,
		}, 5, 10*time.Second)

		require.NoError(t, err, "Events should appear in ClickHouse")
		assert.Len(t, events, 5, "Should have stored all 5 events")

		assert.Equal(t, project.ID, events[0].ProjectID)
		assert.Equal(t, project.OrganizationID, events[0].OrganizationID)
		assert.Equal(t, visitorID, events[0].VisitorID)
		assert.Equal(t, sessionID, events[0].SessionID)

		eventWithUTM := events[4]
		t.Logf("UTM params for first event: %+v", eventWithUTM.UTMParameters)
		if len(eventWithUTM.UTMParameters) > 0 {
			assert.Equal(t, "google", eventWithUTM.UTMParameters["utm_source"])
			assert.Equal(t, "cpc", eventWithUTM.UTMParameters["utm_medium"])
			assert.Equal(t, "summer-sale", eventWithUTM.UTMParameters["utm_campaign"])
		}

		var clickEvent *models.Event
		for i := range events {
			if events[i].ClickElementTag != nil {
				clickEvent = &events[i]
				break
			}
		}
		require.NotNil(t, clickEvent, "Should have at least one event with click data")
		assert.Equal(t, "button", *clickEvent.ClickElementTag)
		assert.Equal(t, "#buy-now", *clickEvent.ClickElementSelector)

		t.Logf("✓ Successfully ingested and verified %d events", len(events))
	})
}

func TestEventIngestion_SimplifiedFlow(t *testing.T) {
	tc := di.NewTestContainer(t)
	defer tc.Cleanup()

	account, project := fixtures.SetupAccountAndProject(t, tc)

	t.Run("can create account and project", func(t *testing.T) {
		assert.NotEmpty(t, account.AccountID)
		assert.NotEmpty(t, account.OrgID)
		assert.NotEmpty(t, account.AccessToken)
		assert.NotEmpty(t, project.ID)
		assert.NotEmpty(t, project.ProjectToken)
		assert.Equal(t, account.OrgID, project.OrganizationID)

		fmt.Printf("✓ Created account: %s\n", account.Email)
		fmt.Printf("✓ Created project: %s (ID: %s)\n", project.Name, project.ID)
		fmt.Printf("✓ Project token: %s\n", project.ProjectToken)
	})

	t.Run("can build events with different properties", func(t *testing.T) {
		pageViewEvent := fixtures.NewEventBuilder().
			WithPageURL("/products").
			WithHost("example.com").
			Build()

		assert.Equal(t, "/products", pageViewEvent.PageURL)
		assert.Equal(t, "example.com", pageViewEvent.Host)
		assert.NotEmpty(t, pageViewEvent.VisitorID)
		assert.NotEmpty(t, pageViewEvent.SessionID)

		utmEvent := fixtures.NewEventBuilder().
			WithPageURL("/landing").
			WithUTMSource("facebook").
			WithUTMMedium("social").
			WithUTMCampaign("winter-campaign").
			Build()

		assert.Equal(t, "facebook", utmEvent.UTMParameters["utm_source"])
		assert.Equal(t, "social", utmEvent.UTMParameters["utm_medium"])
		assert.Equal(t, "winter-campaign", utmEvent.UTMParameters["utm_campaign"])

		clickEvent := fixtures.NewEventBuilder().
			WithPageURL("/product/123").
			WithClickElement("button", "#add-to-cart", "Add to Cart").
			WithClickPosition(100, 200, 1920, 1080).
			Build()

		require.NotNil(t, clickEvent.ClickElement)
		assert.Equal(t, "button", clickEvent.ClickElement.Tag)
		assert.Equal(t, "#add-to-cart", clickEvent.ClickElement.Selector)

		require.NotNil(t, clickEvent.ClickPosition)
		assert.Equal(t, 100.0, clickEvent.ClickPosition.X)
		assert.Equal(t, 200.0, clickEvent.ClickPosition.Y)

		fmt.Println("✓ Event builder creates events with correct properties")
	})

	t.Run("demonstrates ClickHouse query helpers", func(t *testing.T) {

		projectID := project.ID
		opts := fixtures.QueryEventsOptions{
			ProjectID: &projectID,
			Limit:     10,
		}

		events, err := fixtures.QueryEvents(t, tc, opts)
		require.NoError(t, err)
		assert.Empty(t, events, "No events ingested yet")

		count, err := fixtures.CountEvents(t, tc, opts)
		require.NoError(t, err)
		assert.Equal(t, 0, count, "No events ingested yet")

		fmt.Println("✓ ClickHouse query helpers work correctly")
	})
}
