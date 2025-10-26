# Test Fixtures

This package provides test fixtures and utilities for writing integration tests for the Zori analytics platform.

## Available Fixtures

### Event Fixtures (`events.go`)

Build and send analytics events to the test ingestion server.

**Example:**
```go
event := fixtures.NewEventBuilder().
    WithVisitorID("visitor-123").
    WithPageURL("/products").
    WithUTMSource("google").
    WithUTMMedium("cpc").
    Build()

err := fixtures.SendEventToTestServer(t, tc, project, event)
```

**Query events from ClickHouse:**
```go
events, err := fixtures.QueryEvents(t, tc, fixtures.QueryEventsOptions{
    ProjectID: &projectID,
    VisitorID: &visitorID,
})

// Or wait for events to appear
events, err := fixtures.WaitForEvents(t, tc, opts, 5, 10*time.Second)
```

### Payment Fixtures (`payments.go`)

Build and send payment events to the NATS stream for processing.

**Example:**
```go
payment := fixtures.NewPaymentBuilder(projectID, organizationID).
    WithVisitorID("visitor-123").
    WithAmount(9900). // $99.00 in cents
    WithProductName("Premium Plan").
    Build()

err := fixtures.SendPaymentToNATS(t, tc, payment)
```

**Query payments from ClickHouse:**
```go
payments, err := fixtures.QueryPaymentEvents(t, tc, fixtures.QueryPaymentEventsOptions{
    ProjectID: &projectID,
    VisitorID: &visitorID,
})

// Or wait for payments to be processed
payments, err := fixtures.WaitForPayments(t, tc, opts, 3, 10*time.Second)
```

### Account & Project Fixtures (`account.go`, `project.go`)

Create test accounts and projects.

**Example:**
```go
account, project := fixtures.SetupAccountAndProject(t, tc)

// Now use project.ID, project.ProjectToken, etc.
```

## Test Container

The `TestContainer` (from `di` package) provides access to all dependencies:

```go
tc := di.NewTestContainer(t)
defer tc.Cleanup()

// Access services
tc.ClickHouse   // ClickHouse database
tc.DB           // PostgreSQL database
tc.NATS         // NATS stream
tc.RevenueData  // Revenue data service (for testing revenue attribution)
```

## Complete Example: Revenue Attribution Test

See `revenue_attribution_test.go` for a comprehensive example that:

1. Creates visitors with different UTM parameters
2. Sends analytics events
3. Waits for events to be stored and attribution to be computed
4. Sends payment events for those visitors
5. Queries revenue attribution data to verify:
   - Revenue attributed to correct UTM sources
   - Revenue attributed to correct traffic origins
   - Customer profiles are accurate
   - First-touch attribution works correctly

## Best Practices

### Wait for Data to be Processed

Always wait for data to appear in ClickHouse before asserting:

```go
events, err := fixtures.WaitForEvents(t, tc, opts, expectedCount, 10*time.Second)
require.NoError(t, err)
```

### Give Time for Materialized Views

Revenue attribution uses ClickHouse materialized views that update asynchronously:

```go
// After sending events
time.Sleep(2 * time.Second)
// Now query revenue attribution data
```

### Use Unique IDs

Generate unique visitor/session IDs for each test to avoid conflicts:

```go
visitorID := "visitor-test-" + uuid.New().String()
```

### Clean Up

Always defer cleanup of the test container:

```go
tc := di.NewTestContainer(t)
defer tc.Cleanup()
```
