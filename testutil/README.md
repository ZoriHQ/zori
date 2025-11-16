# Testing Infrastructure

This directory contains the testing infrastructure for Zori, providing a modular and reusable framework for integration tests.

## Structure

```
testutil/
├── fixtures/          # Test fixtures and helpers
│   ├── account.go     # Account and authentication helpers
│   ├── project.go     # Project creation helpers
│   ├── events.go      # Event builder and sender
│   ├── queries.go     # ClickHouse query helpers
│   └── payments.go    # Payment event helpers
└── helpers.go         # Common utility functions (polling, waiting)
```

## Quick Start

### 1. Setup Test Environment

**Simple one-command setup:**
```bash
# Start all test services with docker compose
task test:containers

# Or manually:
docker compose -f docker-compose.test.yaml up -d

# Run migrations (happens automatically with task test:migrate)
task test:migrate

# Run all tests
task test

# Run tests with coverage
task test:coverage
```

**Cleanup:**
```bash
task test:cleanup
# or
docker compose -f docker-compose.test.yaml down -v
```

### 2. Write an Integration Test

**Option A: Isolated Test Container (slower but fully isolated)**
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

    err := fixtures.SendEventToTestServer(t, tc, project, event)
    require.NoError(t, err)

    events, err := fixtures.WaitForEvents(t, tc, fixtures.QueryEventsOptions{
        ProjectID: &project.ID,
    }, 1, 5*time.Second)

    require.NoError(t, err)
    assert.Len(t, events, 1)
}
```

**Option B: Shared Test Container (faster, recommended for most tests)**
```go
import (
    "context"
    "testing"
    "zori/di"
    "zori/testutil/fixtures"
)

func TestMyFeature(t *testing.T) {
    tc := di.GetSharedTestContainer(t)

    // Clean up data after test (not the container!)
    defer func() {
        if err := tc.CleanupTestData(context.Background()); err != nil {
            t.Logf("Warning: failed to cleanup test data: %v", err)
        }
    }()

    account, project := fixtures.SetupAccountAndProject(t, tc)

    event := fixtures.NewEventBuilder().
        WithPageURL("/products").
        WithUTMSource("google").
        WithUTMCampaign("summer-sale").
        Build()

    err := fixtures.SendEventToTestServer(t, tc, project, event)
    require.NoError(t, err)

    // Use polling helpers instead of sleep
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

## Performance Best Practices

### Use Shared Containers
- **Recommended**: Use `di.GetSharedTestContainer(t)` for faster tests
- The container is created once per test package and reused
- Clean up test data with `tc.CleanupTestData(ctx)` instead of `tc.Cleanup()`
- **5-10x faster** than creating a new container per test

### Use Polling Helpers
- **Always use** `WaitForEvents()` instead of `time.Sleep()`
- Use `WaitForCondition()` for custom polling logic
- Use `WaitForClickHouseMaterializedView()` when waiting for ClickHouse updates
- Makes tests faster and more reliable

### Example: Before and After
**Before (slow):**
```go
fixtures.SendEventToTestServer(t, tc, project, event)
time.Sleep(2 * time.Second)  // Hope it's processed
events, _ := fixtures.QueryEvents(t, tc, opts)
```

**After (fast & reliable):**
```go
fixtures.SendEventToTestServer(t, tc, project, event)
events, err := fixtures.WaitForEvents(t, tc, opts, 1, 5*time.Second)
require.NoError(t, err)
```

## Notes

- Tests use `.env.test` for configuration
- Docker Compose manages all test services (Postgres, ClickHouse, NATS, Redis)
- Test containers run on ports: 5434 (Postgres), 9002 (ClickHouse), 4222 (NATS), 6379 (Redis)
- Services include health checks for reliable startup
- Use `GetSharedTestContainer()` for faster tests, `NewTestContainer()` for full isolation
- Cleanup is automatic via defer statements
