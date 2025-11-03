# Testing Infrastructure

This directory contains the testing infrastructure for Zori, providing a modular and reusable (not really, but hopefully :) ) framework for integration tests.

## Structure

```
testutil/
├── fixtures/          # Test fixtures and helpers
│   ├── account.go     # Account and authentication helpers
│   ├── project.go     # Project creation helpers
│   ├── events.go      # Event builder and sender
│   └── queries.go     # ClickHouse query helpers
└── helpers.go         # Common utility functions
```

## Quick Start

### 1. Setup Test Environment

```bash
# Start test containers (Postgres, ClickHouse, NATS)
task test-containers

# Run migrations
task test-migrate

# Run tests
go test ./integration/...
```

### 2. Write an Integration Test

```go
import (
    "testing"
    "zori/di"
    "zori/testutil/fixtures"
)

func TestMyFeature(t *testing.T) {
    tc := di.NewTestContainer(t)
    defer tc.Cleanup()

    account, project := fixtures.SetupAccountAndProject(t, tc)

    event := fixtures.NewEventBuilder().
        WithPageURL("/products").
        WithUTMSource("google").
        WithUTMCampaign("summer-sale").
        Build()

    events, err := fixtures.WaitForEvents(t, tc, fixtures.QueryEventsOptions{
        ProjectID: &project.ID,
    }, 1, 5*time.Second)

    require.NoError(t, err)
    assert.Len(t, events, 1)
}
```

## Fixtures

### Account Fixtures (`fixtures/account.go`)

- **`CreateAccount(t, tc)`** - Creates a test account with organization
- Returns `AccountFixture` with access tokens and IDs

### Project Fixtures (`fixtures/project.go`)

- **`CreateProject(t, tc, account, name, domain, allowLocalhost)`** - Creates a project
- **`SetupAccountAndProject(t, tc)`** - Convenience function to create both
- Returns `ProjectFixture` with project details and tokens

### Event Fixtures (`fixtures/events.go`)

- **`NewEventBuilder()`** - Creates a fluent event builder with defaults
- Builder methods:
  - `WithEventName(name)`
  - `WithPageURL(url)`
  - `WithUTMSource/Medium/Campaign(value)`
  - `WithClickElement(tag, selector, text)`
  - `WithClickPosition(x, y, width, height)`
  - `WithIdentity(userID, externalID, email)`
  - `WithCustomProperty(key, value)`
- **`SendEvent(t, url, projectToken, event)`** - Sends event to ingestion endpoint
- **`SendEvents(t, url, projectToken, events)`** - Sends multiple events

### Query Helpers (`fixtures/queries.go`)

- **`QueryEvents(t, tc, opts)`** - Queries events from ClickHouse
- **`WaitForEvents(t, tc, opts, expectedCount, timeout)`** - Waits for events to appear
- **`CountEvents(t, tc, opts)`** - Counts events matching criteria
- **`AssertEventExists(t, tc, opts)`** - Asserts at least one event exists
- **`AssertEventCount(t, tc, opts, count)`** - Asserts exact event count

Query options:
- `ProjectID`
- `OrganizationID`
- `VisitorID`
- `SessionID`
- `EventName`
- `Limit`

## Test Container

The `di.TestContainer` provides access to:
- `DB` - PostgreSQL database connection
- `ClickHouse` - ClickHouse database connection
- `NATS` - NATS streaming connection
- `Server` - HTTP server with registered routes
- `Config` - Test configuration

## Example: Complete Integration Test

See `integration/events_ingestion_test.go` for a complete example that demonstrates:
1. Creating accounts and projects
2. Building events with various properties
3. Querying ClickHouse for stored events
4. Verifying event data integrity

## Future Extensions

This framework is designed to be extended for:
- Payment event testing
- Revenue calculation tests
- Session tracking tests
- Identity resolution tests
- Custom event properties tests
- Multi-project scenarios
- Performance testing

## Notes

- Tests use `.env.test` for configuration
- Test containers run on ports 5434 (Postgres), 9002 (ClickHouse), 4222 (NATS)
- Each test gets a fresh TestContainer with isolated dependencies
- Cleanup is automatic via defer statements
