# Zori Codebase Analysis Report

**Project**: Zori Analytics Platform (Go)  
**Analysis Date**: November 16, 2025  
**Total Go Files**: 148  
**Application Type**: Multi-service analytics and event ingestion platform

---

## Executive Summary

Zori is a distributed analytics platform with two main server applications:
1. **Main Server** (Port 1323) - REST API for analytics, organizations, projects, payments, and revenue tracking
2. **Ingestion Server** (Port 1324) - High-performance event ingestion using fasthttp

The architecture uses NATS JetStream for event streaming, PostgreSQL for relational data, and ClickHouse for analytical data. The codebase implements custom context wrapping (`ctx.Ctx`) for authentication and organizational context propagation through HTTP handlers.

---

## 1. Project Structure & Packages

### Root-Level Organization
```
/home/user/zori/
├── main.go                    # Entry point with two CLI commands
├── di/                        # Dependency injection containers
├── internal/                  # Core infrastructure
├── services/                  # Business logic organized by domain
└── testutil/                  # Testing utilities
```

### Internal Packages (`/home/user/zori/internal/`)
| Package | Purpose |
|---------|---------|
| **ctx/** | Custom context wrapper for authentication and org context |
| **config/** | Configuration management |
| **server/** | Echo HTTP server setup and middleware |
| **server/middlewares/** | Auth middleware (Clerk, OSS JWT) and caching |
| **server/validators/** | Request validation |
| **storage/postgres/** | PostgreSQL connection and data models |
| **storage/clickhouse/** | ClickHouse connection and event models |
| **natsstream/** | NATS JetStream client wrapper |
| **cache/** | Redis-based caching service |
| **metrics/** | Prometheus metrics collection |
| **telemetry/** | Telemetry context (for OpenTelemetry integration) |
| **init/** | OSS initialization |
| **utils/** | Encryption and validation utilities |

### Services Packages (`/home/user/zori/services/`)
| Service | Purpose |
|---------|---------|
| **analytics/** | Analytics tiles, visitors tracking, event queries |
| **auth/** | Authentication service (Clerk integration + OSS JWT) |
| **events/** | Event processing pipeline with NATS consumers |
| **ingestion/** | HTTP ingestion server for client events |
| **organizations/** | Organization management |
| **projects/** | Project management |
| **payments/** | Payment provider integration (Stripe) |
| **revenue/** | Revenue analytics and attribution |

**Full directory structure:**
```
services/
├── analytics/               # Analytics service
│   ├── tiles/              # Tile view components (15+ files)
│   ├── data/               # Data access layer
│   ├── services/           # Business logic
│   ├── filters/            # Query filters
│   ├── types/              # Type definitions
│   ├── web/                # HTTP routes
│   └── container.go        # DI container
│
├── auth/                   # Authentication service
│   ├── web/                # Routes
│   └── container.go
│
├── events/                 # Event processing
│   ├── services/           # Processors (identify, location, user-agent, etc.)
│   ├── classifier/         # Click classification
│   ├── web/                # Routes
│   └── container.go
│
├── ingestion/              # Ingestion server
│   ├── web/                # FastHTTP handlers
│   ├── services/           # Ingestor, Identifier
│   ├── data/               # Data access
│   ├── types/              # Event types
│   └── container.go
│
├── organizations/          # Organization service
│   ├── services/
│   ├── data/
│   ├── web/
│   └── container.go
│
├── projects/               # Project service
│   ├── services/
│   ├── data/
│   ├── web/
│   └── container.go
│
├── payments/               # Payment service
│   ├── services/           # Provider management, webhooks, processor
│   ├── data/
│   ├── web/                # Routes & webhook routes
│   ├── types/
│   └── container.go
│
└── revenue/                # Revenue analytics
    ├── services/
    ├── data/
    ├── web/
    ├── types/
    └── container.go
```

---

## 2. Main Operations Performed

### 2.1 HTTP Endpoints & Handlers

#### **Main Server (Echo Framework)**
**Location**: `/home/user/zori/internal/server/server.go`

The server uses a custom handler pattern with type safety:
- **Generic handler functions**: `HandlerFunc[T]` and `HandlerFuncWithFilter[T, F]`
- **Handler wrapper**: Extracts `ctx.Ctx` from Echo context, handles errors, returns JSON
- **Server methods**: `GET`, `POST`, `PUT`, `DELETE`, `PATCH` (both global and group-scoped)

**Analytics Endpoints** (`/home/user/zori/services/analytics/web/routes.go`, lines 30-73)
```go
GET  /api/v1/analytics/visitors/device
GET  /api/v1/analytics/visitors/top
POST /api/v1/analytics/visitors/identify
GET  /api/v1/analytics/visitors/profile
GET  /api/v1/analytics/timeline
GET  /api/v1/analytics/tiles/unique-visitors
GET  /api/v1/analytics/tiles/unique-sessions
GET  /api/v1/analytics/tiles/bounce-rate
GET  /api/v1/analytics/tiles/session-duration
GET  /api/v1/analytics/tiles/pages-per-session
GET  /api/v1/analytics/tiles/dau
GET  /api/v1/analytics/tiles/wau
GET  /api/v1/analytics/tiles/mau
GET  /api/v1/analytics/tiles/return-rate
GET  /api/v1/analytics/tiles/time-between-visits
GET  /api/v1/analytics/tiles/traffic-by-country
GET  /api/v1/analytics/tiles/traffic-by-referer
GET  /api/v1/analytics/tiles/traffic-by-utm
GET  /api/v1/analytics/events/recent
GET  /api/v1/analytics/events/filter-options
```
**Cache Levels**: High frequency (1min), Medium (2min), Low (5min)

**Organization Endpoints** (`/home/user/zori/services/organizations/web/routes.go`, lines 9-18)
```go
GET /api/v1/organization/
```

**Projects Endpoints** (`/home/user/zori/services/projects/web/routes.go`, lines 9-22)
```go
GET    /api/v1/projects/list
GET    /api/v1/projects/:id
POST   /api/v1/projects
PUT    /api/v1/projects/:id
DELETE /api/v1/projects/:id
```

**Payments Endpoints** (`/home/user/zori/services/payments/web/routes.go`, lines 9-27)
```go
GET  /api/v1/payment-providers/instructions
POST /api/v1/payment-providers
GET  /api/v1/payment-providers
GET  /api/v1/payment-providers/:id
PUT  /api/v1/payment-providers/:id
DELETE /api/v1/payment-providers/:id
GET  /api/v1/payment-providers/stripe/app/callback
```

**Webhook Routes** (`/home/user/zori/services/payments/web/webhook_routes.go`, lines 8-13)
```go
POST /webhooks/stripe/app/lifecycle
POST /webhooks/stripe/cloud/app
POST /webhooks/stripe/:project_id
```

**Revenue Endpoints** (`/home/user/zori/services/revenue/web/routes.go`, lines 11-70)
```go
GET  /api/v1/revenue/dashboard
GET  /api/v1/revenue/attribution/origin
GET  /api/v1/revenue/attribution/utm
GET  /api/v1/revenue/timeline
GET  /api/v1/revenue/customers/top
GET  /api/v1/revenue/customers/profile
GET  /api/v1/revenue/conversion/metrics
POST /api/v1/revenue/cohort/metrics
```

**Health & System Endpoints**
- `GET /health` - Server health check

#### **Ingestion Server (FastHTTP)**
**Location**: `/home/user/zori/services/ingestion/web/server.go`

```go
POST   /ingest      # Main event ingestion endpoint
POST   /identify    # User identification endpoint
GET    /health      # Health check
```

**Handler**: `IngestionServer.HandleRequest()` (line 33) - Route dispatcher
- `Injest()` method (line 60) - Event ingestion
- `Identify()` method (line 166) - User identification
- CORS headers support
- Support for OPTIONS requests

---

### 2.2 Data Ingestion Operations

#### **Event Ingestion Pipeline** 
**Entry Point**: `POST /ingest` → `/home/user/zori/services/ingestion/web/server.go:Injest()`

**Processing Flow**:
1. **Validation** (lines 61-95):
   - Verify POST method
   - Parse JSON event payload
   - Validate visitor ID cookie
   - Validate X-Zori-PT project token header
   - Validate host origin (localhost support)

2. **Caching** (lines 93-119):
   - Project lookup with cache hit/miss handling
   - `cache.ProjectCacheKey.FromValue(projectToken)`
   - TTL: 1 minute

3. **Enrichment** (lines 150-159):
   - User-Agent extraction
   - IP address extraction (CloudFlare, X-Forwarded-For, or direct)
   - Project first-event tracking (async, line 122)

4. **Async Publishing** (line 161):
   ```go
   go h.ingestor.Ingest(&project, &clientEvent)
   ```
   - Calls `Ingestor.Ingest()` in goroutine

**Ingestor Service** (`/home/user/zori/services/ingestion/services/ingestor.go`, lines 33-89)
```go
func (i *Ingestor) Ingest(project *models.Project, clientEvent *types.ClientEventV1) error
```
- **Event Deduplication** (lines 40-55):
  - Key: `project.ID:clientEvent.ClientGeneratedEventID`
  - Window: 5 seconds (line 16)
  - Cache operation: `SetNX()`
  - Metric recording: `RecordEventDedupe()`

- **Event Framing** (lines 57-61):
  - Wraps with project/org context

- **NATS Publishing** (lines 70-74):
  - Stream: `events:raw`
  - Async publishing

- **Metrics Recording** (lines 85-86):
  - Event type tracking (track vs identify)
  - Request timing
  - Success/error status

#### **User Identification**
**Entry Point**: `POST /identify` → `/home/user/zori/services/ingestion/web/server.go:Identify()`

**Processing** (lines 166-256):
- Similar validation and enrichment as Injest
- Async identifier call (line 248):
  ```go
  go func() {
      err := h.identifier.Identify(ctx, &project, &identifyEvent)
  }()
  ```

---

### 2.3 Background Jobs & Event Processing Workers

#### **Event Processing Pipeline (NATS JetStream)**
**Location**: `/home/user/zori/services/events/services/processor.go`

**Consumer Setup** (lines 38-84):
- Stream: `events:raw`
- Consumer: `event-enricher` (durable)
- JetStream context initialization

**Event Processing Loop** (lines 86-208):
```go
func (p *Processor) Start() error {
    _, err := p.consumer.Consume(func(msg jetstream.Msg) {
        // Process event
    })
}
```

**Processing Stages** (lines 216-238):
Pipeline with 6 sequential enrichment stages:
1. **Location** - IP geolocation lookup
2. **Page** - URL parsing  
3. **UserAgent** - Browser/OS detection
4. **Referrer** - Referrer domain extraction
5. **Identity** - User identity linking
6. **ClickClassification** - CTA/link classification

Each stage is timed and errors are tracked with metrics.

**ClickHouse Insert** (lines 134-192):
- `AsyncInsert()` to events table
- 40 columns inserted including:
  - Event metadata (visitor_id, session_id, event_name)
  - Browser/device info (browser_name, os_name, device_type)
  - Location (country, city, lat/long)
  - Page info (url, path, referrer)
  - UTM parameters
  - Click metadata (position, element, selector, text)
  - User/external IDs, email hash

**Message Acknowledgment**:
- Success: `msg.Ack()`
- Error: `msg.Nak()` (negative acknowledgment)

#### **Identify Event Processing**
**Location**: `/home/user/zori/services/events/services/identify_processor.go`

**Consumer Setup** (lines 38-74):
- Stream: `events:identify`
- Consumer: `identify-processor` (durable)

**Processing Loop** (lines 76-101):
```go
func (p *IdentifyProcessor) Start() error {
    _, err := p.consumer.Consume(func(msg jetstream.Msg) {
        // Unmarshal identify frame
        // Update visitor events with identity
        // Ack/Nak
    })
}
```

**Visitor Update Operations** (line 89):
- Updates ClickHouse events table with user identity
- Records email hash (SHA256)
- Updates user_id and external_id fields

#### **Payment Event Processing**
**Location**: `/home/user/zori/services/payments/services/payment_processor.go`

**Consumer Setup** (lines 34-70):
- Stream: `payments:raw`
- Consumer: `payment-processor` (durable)

**Processing Loop** (lines 72-91):
```go
func (p *PaymentProcessor) Start() error {
    _, err := p.consumer.Consume(func(msg jetstream.Msg) {
        // Unmarshal payment frame
        // Process payment
        // Ack/Nak
    })
}
```

#### **Application Lifecycle Management (fx)**
**Location**: `/home/user/zori/di/app_container.go`, `/home/user/zori/di/ingestion_container.go`

**Main Server Lifecycle** (lines 92-119):
```go
fx.Invoke(func(lc fx.Lifecycle, metricsServer *metrics.MetricsServer) {
    lc.Append(fx.Hook{
        OnStart: func(ctx context.Context) error { ... },
        OnStop: func(ctx context.Context) error { ... },
    })
})
```

Manages:
- Metrics server startup/shutdown
- HTTP server startup/shutdown (goroutine-based)
- Database connection cleanup
- OSS initialization

**Ingestion Server Lifecycle** (lines 46-74):
- Metrics server hooks
- FastHTTP server startup/shutdown
- Signal handling

---

### 2.4 Database Operations

#### **PostgreSQL Operations**
**Connection**: `/home/user/zori/internal/storage/postgres/connection.go`

**Query Examples** (analytics tiles):

**UniqueVisitorsTile** (`/home/user/zori/services/analytics/tiles/visitors.tile.view.go`):
```go
row := t.db.Db().QueryRow(ctx, query, ctx.OrgID(), filter.ProjectID)
```

**TrafficByCountry** (`/home/user/zori/services/analytics/tiles/traffic.bycountry.tile.view.go`):
```go
queryResult, err := t.db.Db().Query(ctx, query, ctx.OrgID(), filter.ProjectID)
```

**All Analytics Queries** use:
- `QueryRow()` for single results
- `Query()` for result sets
- Parameters: `ctx.OrgID()`, `filter.ProjectID` (organization/project isolation)

**Organization Queries** (`/home/user/zori/services/organizations/data/organization.go`):
```go
err := o.db.NewSelect().Model(&model).Where("id = ?", id).Scan(c, &model)
```
- Uses bun ORM (scanned with context)

#### **ClickHouse Operations**
**Connection**: `/home/user/zori/internal/storage/clickhouse/connection.go`

**Async Inserts** (`/home/user/zori/services/events/services/processor.go`, lines 134-186):
```go
p.clickDb.Db().AsyncInsert(context.Background(),
    `INSERT INTO events (...) VALUES (...)`, true,
    // 40 parameters
)
```

**Query Operations** (`/home/user/zori/services/analytics/data/analytics.go`):
```go
rows, err := a.clickDb.Db().Query(ctx, query, filter.ProjectID, filter.TimeRange.Start)
```

**Tables**:
- `events` - Main event table
- Queried for analytics aggregations

---

### 2.5 Caching Layer

**Location**: `/home/user/zori/internal/cache/cache.go`

**Cache Types**:
- **Project Cache** (`cache.ProjectCacheKey`)
- **Event Dedupe Cache** (`cache.EventDedupeCacheKey`)
- **Analytics Cache** (prefixed with `analytics`)
- **Revenue Cache** (prefixed with `revenue`)

**Operations**:
- `Get(ctx, key)` - Retrieve value
- `Set(ctx, key, value, ttl)` - Store value with TTL
- `SetNX(ctx, key, value, ttl)` - Set if not exists (for deduplication)

**Middleware Integration** (`/home/user/zori/internal/server/middlewares/cache.go`):
```go
func (m *CacheMiddleware) Middleware(config CacheConfig) echo.MiddlewareFunc
```
- Cache key generation using MD5 hash of path + query params
- Project-ID-based key prefix isolation
- Response capture and serialization

---

## 3. ctx.Ctx Object Usage Throughout Codebase

### Definition
**Location**: `/home/user/zori/internal/ctx/context.go`, lines 10-52

```go
type Ctx struct {
    context.Context      // Embedded standard context
    Echo  echo.Context   // Echo HTTP context
    User  *models.Account // Authenticated user
    orgID string         // Organization ID
}
```

**Methods**:
- `NewCtx(c echo.Context)` - Constructor from Echo context
- `SetUser(user *models.Account)` - Set authenticated user
- `SetOrgID(orgID string)` - Set organization ID
- `IsAuthenticated() bool` - Check if user set
- `HasOrg() bool` - Check if org ID set
- `UserID() string` - Get user ID
- `OrgID() string` - Get organization ID

### Usage Patterns

#### **Creation in Handler Wrappers**
**Location**: `/home/user/zori/internal/server/server.go`, lines 94-115

```go
func wrapHandler[T any](s *Server, handler HandlerFunc[T]) echo.HandlerFunc {
    return func(c echo.Context) error {
        appctx, ok := c.Get("ctx").(*ctx.Ctx)
        if !ok {
            appctx = ctx.NewCtx(c)
            c.Set("ctx", appctx)
        }
        result, err := handler(appctx)
        // ...
    }
}
```

#### **Auth Middleware Setup - Clerk**
**Location**: `/home/user/zori/internal/server/middlewares/clerk_auth.go`, lines 40-81

```go
func (m *ClerkAuthMiddleware) Middleware() echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            // JWT verification...
            userID := claims.Subject
            
            reqCtx := ctx.NewCtx(c)
            reqCtx.SetUser(&models.Account{
                ID:    userID,
                Email: "",
            })
            if claims.ActiveOrganizationID != "" {
                reqCtx.SetOrgID(claims.ActiveOrganizationID)
            }
            c.Set("ctx", reqCtx)
            return next(c)
        }
    }
}
```

#### **Auth Middleware Setup - OSS JWT**
**Location**: `/home/user/zori/internal/server/middlewares/oss_auth.go`, lines 31-79

```go
func (m *OSSAuthMiddleware) Middleware() echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            // JWT parsing...
            reqCtx := ctx.NewCtx(c)
            reqCtx.SetUser(&models.Account{
                ID:    "oss-admin",
                Email: "admin@localhost",
            })
            reqCtx.SetOrgID(claims.OrgID)
            c.Set("ctx", reqCtx)
            return next(c)
        }
    }
}
```

#### **Service Layer Usage**
**Location**: `/home/user/zori/services/organizations/services/organization.go`, lines 19-26

```go
func (s *OrganizationService) GetOrganization(c *ctx.Ctx) (*models.Organization, error) {
    orgId := c.OrgID()
    return s.GetOrganizationByID(c, orgId)
}

func (s *OrganizationService) GetOrganizationByID(c *ctx.Ctx, id string) (*models.Organization, error) {
    return s.data.GetOrganizationByID(c, id)
}
```

#### **Data Layer Usage**
**Location**: `/home/user/zori/services/organizations/data/organization.go`

```go
func (o *OrganizationData) GetOrganizationByID(c *ctx.Ctx, id string) (*models.Organization, error) {
    var model models.Organization
    err := o.db.NewSelect().Model(&model).Where("id = ?", id).Scan(c, &model)
    return &model, err
}
```

#### **Analytics Tiles Usage**
All analytics tiles use the context for:
- **User isolation**: `ctx.OrgID()` - ensures organization-level access control
- **Project filtering**: `filter.ProjectID` - filters analytics to specific project

**Example**: `/home/user/zori/services/analytics/tiles/visitors.tile.view.go`

```go
func (t *UniqueVisitorsTile) Fetch(ctx *ctx.Ctx, filter *filters.SectionFilter) (*UniqueVisitorsResponse, error) {
    row := t.db.Db().QueryRow(ctx, query, ctx.OrgID(), filter.ProjectID)
}
```

**Usage in 50+ Handler Functions**:
- All tile/analytics handlers: `func(*ctx.Ctx, *filters.SectionFilter)`
- Service handlers: `func(*ctx.Ctx) (T, error)`
- Data layer: Always receives ctx for cancellation and isolation

---

## 4. Logging Implementation

### Current Logging Approach

**Log Packages Used**:
- `log` - Standard Go logging (most common)
- `fmt.Printf` - Standard printing to stdout

**No structured logging** - No JSON logging or structured log libraries (slog, logrus, zap)

### Logging Locations

#### **Event Processing Errors**
**File**: `/home/user/zori/services/events/services/processor.go`, lines 128-131

```go
if err := p.clickDb.Ping(context.Background()); err != nil {
    fmt.Println(err)
    log.Printf("Error pinging database: %v", err)
}
```

#### **Identify Processing**
**File**: `/home/user/zori/services/events/services/identify_processor.go`

```go
log.Printf("Failed to unmarshal identify event: %v", err)
log.Printf("Failed to update visitor events: %v", err)
log.Printf("Updated events for visitor %s with identity information", identifyFrame.VisitorID)
```

#### **Payment Processing**
**File**: `/home/user/zori/services/payments/services/payment_processor.go`

```go
log.Printf("Failed to unmarshal payment event: %v", err)
log.Printf("Failed to process payment: %v", err)
log.Printf("Successfully processed payment %s (visitor: %v, amount: %d %s)",
    paymentFrame.PaymentID, visitorID, paymentFrame.Amount, paymentFrame.Currency)
```

#### **Location Stage Warning**
**File**: `/home/user/zori/services/events/services/stage_location.go`

```go
log.Printf("Warning: Could not open ipdb.mmdb: %v. Location enrichment will be disabled.", err)
```

#### **Event Deduplication Warning**
**File**: `/home/user/zori/services/ingestion/services/ingestor.go`, line 47

```go
fmt.Printf("Warning: event deduplication check failed: %v\n", err)
```

#### **Ingestion Server Errors**
**File**: `/home/user/zori/services/ingestion/web/server.go`

```go
fmt.Println("Identify error: ", err)
```

#### **Metrics Server**
**File**: `/home/user/zori/internal/metrics/server.go`

```go
log.Println("Metrics server is disabled")
log.Printf("Starting metrics server on :%s", s.config.MetricsPort)
```

#### **Payments Webhook Handling**
**File**: `/home/user/zori/services/payments/services/webhook_handler.go`

```go
fmt.Printf("Stripe App authorized for account: %s\n", event.Account)
fmt.Printf("Provider not found for account %s: %v\n", accountID, err)
```

#### **Backfill Operations**
**File**: `/home/user/zori/services/payments/services/backfill_service.go`

```go
log.Printf("Starting backfill for provider %s from %s", provider.ID, startDate.Format(time.RFC3339))
log.Printf("Completed backfill for provider %s", provider.ID)
log.Printf("Failed to marshal payment frame for charge %s: %v", charge.ID, err)
```

---

## 5. Key Entry Points for Adding Tracing

### 5.1 HTTP Request Entry Points

#### **Main Server (Echo)**
**File**: `/home/user/zori/internal/server/server.go`, lines 94-115

All HTTP request handlers pass through `wrapHandler()`:
```go
func wrapHandler[T any](s *Server, handler HandlerFunc[T]) echo.HandlerFunc {
    return func(c echo.Context) error {
        appctx, ok := c.Get("ctx").(*ctx.Ctx)
        if !ok {
            appctx = ctx.NewCtx(c)
            c.Set("ctx", appctx)
        }
        result, err := handler(appctx)
        // ...
        return c.JSON(statusCode, result)
    }
}
```

**Recommendation**: Add tracing span here - captures all HTTP requests to main server

**Entry Point Method**:
- Modify `wrapHandler()` to create and attach trace span to `ctx.Ctx`
- Instrument handler execution
- Record error status

#### **Ingestion Server (FastHTTP)**
**File**: `/home/user/zori/services/ingestion/web/server.go`, line 33

```go
func (h *IngestionServer) HandleRequest(ctx *fasthttp.RequestCtx) {
    // Route dispatcher
    path := string(ctx.Path())
    switch path {
        case "/ingest": h.Injest(ctx)
        case "/identify": h.Identify(ctx)
    }
}
```

**Recommendation**: Add tracing span creation in `HandleRequest()`

**Child Spans**:
- `Injest()` - lines 60-164
- `Identify()` - lines 166-256

---

### 5.2 Async Background Job Entry Points

#### **Event Ingestion to Processing**
**File**: `/home/user/zori/services/ingestion/services/ingestor.go`, line 70

```go
if err = i.natsStream.GetConnection().Publish("events:raw", eventFrameBytes); err != nil {
    // ...
}
```

**Span Should Track**:
- NATS publish operation
- Event deduplication check (lines 40-55)
- Event frame marshaling (lines 63-68)

#### **Event Processing Worker - Start**
**File**: `/home/user/zori/services/events/services/processor.go`, line 86

```go
func (p *Processor) Start() error {
    _, err := p.consumer.Consume(func(msg jetstream.Msg) {
        // Process event
    })
}
```

**Spans Should Include**:
- Message consumption (outer loop)
- Event frame unmarshaling (lines 90-96)
- Event processing pipeline (lines 98-103)
- Each processing stage (lines 216-238):
  - Location stage
  - Page stage
  - User-agent stage
  - Referrer stage
  - Identity stage
  - Click classification stage
- ClickHouse insert (lines 134-186)
- Message acknowledgment (lines 195-197)

#### **Identify Event Processing Worker**
**File**: `/home/user/zori/services/events/services/identify_processor.go`, line 76

```go
func (p *IdentifyProcessor) Start() error {
    _, err := p.consumer.Consume(func(msg jetstream.Msg) {
        // Process identify event
    })
}
```

**Spans Should Track**:
- Message unmarshaling
- Visitor events update
- Email hash calculation

#### **Payment Event Processing Worker**
**File**: `/home/user/zori/services/payments/services/payment_processor.go`, line 72

```go
func (p *PaymentProcessor) Start() error {
    _, err := p.consumer.Consume(func(msg jetstream.Msg) {
        // Process payment
    })
}
```

**Spans Should Track**:
- Message unmarshaling
- Payment processing
- ClickHouse insert

---

### 5.3 Database Operation Entry Points

#### **PostgreSQL Queries**
**File**: `/home/user/zori/services/organizations/data/organization.go`

```go
err := o.db.NewSelect().Model(&model).Where("id = ?", id).Scan(c, &model)
```

All data layer files follow pattern:
- `/home/user/zori/services/*/data/*.go`

**Recommendation**: 
- Add DB connection pooling metrics
- Instrument `Query()` and `QueryRow()` calls
- Track query duration by table

#### **ClickHouse Inserts**
**File**: `/home/user/zori/services/events/services/processor.go`, lines 134-186

```go
err := p.clickDb.Db().AsyncInsert(context.Background(), 
    `INSERT INTO events (...) VALUES (...)`, true, /* 40 parameters */
)
```

**Recommendation**:
- Instrument `AsyncInsert()` calls
- Track batch sizes
- Monitor insert latency
- Track failures by table

#### **ClickHouse Queries**
**File**: `/home/user/zori/services/analytics/data/analytics.go`

```go
rows, err := a.clickDb.Db().Query(ctx, query, filter.ProjectID, filter.TimeRange.Start)
```

**All Analytics Query Locations**:
- `/home/user/zori/services/analytics/tiles/*.go` - 15+ tile views
- Each calls `t.db.Db().Query()` or `t.db.Db().QueryRow()`

---

### 5.4 NATS Stream Entry Points

#### **Event Publishing**
**File**: `/home/user/zori/services/ingestion/services/ingestor.go`, line 70

```go
if err = i.natsStream.GetConnection().Publish("events:raw", eventFrameBytes); err != nil {
```

**Recommendation**: Instrument NATS publish operations

#### **NATS Consumer Creation & Consumption**
**Files**:
- `/home/user/zori/services/events/services/processor.go`, lines 86-207
- `/home/user/zori/services/events/services/identify_processor.go`, lines 76-101
- `/home/user/zori/services/payments/services/payment_processor.go`, lines 72-91

**Recommendation**:
- Create spans for consumer creation
- Create child spans for each message consumption
- Track consumer lag
- Monitor Ack/Nak rates

---

### 5.5 Cache Operation Entry Points

#### **Cache Middleware**
**File**: `/home/user/zori/internal/server/middlewares/cache.go`, lines 33-73

```go
func (m *CacheMiddleware) Middleware(config CacheConfig) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            cachedValue, err := m.cacheService.Get(c.Request().Context(), cacheKey)
            // ...
            _ = m.cacheService.Set(c.Request().Context(), cacheKey, responseObj, config.TTL)
        }
    }
}
```

**Recommendation**: 
- Instrument cache hits/misses
- Track cache key generation time
- Monitor cache operation latency

#### **Event Deduplication Cache**
**File**: `/home/user/zori/services/ingestion/services/ingestor.go`, lines 40-55

```go
isNew, err := i.cacheService.SetNX(
    context.Background(),
    dedupeKey,
    true,
    EventDeduplicationWindow,
)
```

**Recommendation**: Track deduplication success rate

---

### 5.6 Critical Service Initialization Points

#### **Application Startup**
**File**: `/home/user/zori/di/app_container.go`, lines 45-123

**Main Server DI Container** - All services initialized here:
- Config loading
- Database connections (PostgreSQL, ClickHouse)
- Server creation
- NATS stream
- Cache service
- Metrics collectors
- Auth middleware
- All business services

**Recommendation**: Add startup tracing span covering:
- Component initialization duration
- Dependency resolution
- Service startup hooks

#### **Ingestion Server Startup**
**File**: `/home/user/zori/di/ingestion_container.go`, lines 23-78

- FastHTTP server setup
- Ingestion-specific service initialization

---

### 5.7 Authentication Entry Points

#### **Clerk Auth Middleware**
**File**: `/home/user/zori/internal/server/middlewares/clerk_auth.go`, lines 40-81

```go
func (m *ClerkAuthMiddleware) Middleware() echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            claims, err := jwt.Verify(context.Background(), &jwt.VerifyParams{
                Token: tokenString,
            })
            // Set user and org in ctx
        }
    }
}
```

**Recommendation**: Add span for JWT verification time

#### **OSS Auth Middleware**
**File**: `/home/user/zori/internal/server/middlewares/oss_auth.go`, lines 31-79

**Recommendation**: Add span for JWT parsing time

---

### 5.8 Webhook Entry Points (External Events)

#### **Stripe Webhook Handlers**
**File**: `/home/user/zori/services/payments/services/webhook_handler.go`

Methods:
- `HandleStripeAppLifecycleWebhook()`
- `HandleStripeAppWebhook()`
- `HandleStripeWebhook()`

**Recommendation**: 
- Create span for each webhook reception
- Track webhook event type
- Record processing latency

---

## 6. Summary Table: Tracing Instrumentation Points

| **Category** | **File Path** | **Function/Method** | **Line(s)** | **Span Type** |
|---|---|---|---|---|
| **HTTP Request** | `internal/server/server.go` | `wrapHandler()` | 94-115 | Request (main) |
| **HTTP Ingestion** | `services/ingestion/web/server.go` | `HandleRequest()` | 33 | Request (ingestion) |
| **Event Ingest** | `services/ingestion/web/server.go` | `Injest()` | 60 | Child of HTTP request |
| **User Identify** | `services/ingestion/web/server.go` | `Identify()` | 166 | Child of HTTP request |
| **Ingestor** | `services/ingestion/services/ingestor.go` | `Ingest()` | 33 | Async operation |
| **Event Processing** | `services/events/services/processor.go` | `Start()` / message loop | 86 | Stream consumer |
| **Identify Processing** | `services/events/services/identify_processor.go` | `Start()` / message loop | 76 | Stream consumer |
| **Payment Processing** | `services/payments/services/payment_processor.go` | `Start()` / message loop | 72 | Stream consumer |
| **Processing Stages** | `services/events/services/processor.go` | `processEvent()` | 216 | Child spans (6 stages) |
| **ClickHouse Insert** | `services/events/services/processor.go` | Message handler | 134-186 | Database operation |
| **Analytics Query** | `services/analytics/tiles/*.go` | `Fetch()` methods | - | Database operation |
| **Cache Get** | `internal/server/middlewares/cache.go` | `Middleware()` | 42 | Cache operation |
| **Cache Set** | `internal/server/middlewares/cache.go` | `Middleware()` | 66 | Cache operation |
| **Dedup Check** | `services/ingestion/services/ingestor.go` | `Ingest()` | 40 | Cache operation |
| **Auth Verify** | `internal/server/middlewares/clerk_auth.go` | `Middleware()` | 50 | Auth operation |
| **Auth Parse** | `internal/server/middlewares/oss_auth.go` | `Middleware()` | 41 | Auth operation |
| **Stripe Webhook** | `services/payments/services/webhook_handler.go` | `HandleStripe*()` | - | External event |
| **App Startup** | `di/app_container.go` | `NewApplication()` | 45-123 | Startup |
| **Ingestion Startup** | `di/ingestion_container.go` | `NewIngestionApplication()` | 23-78 | Startup |

---

## 7. Context Flow Through Application

### Request Context Propagation
```
HTTP Request
    ↓
[CORS Middleware]
    ↓
[Logger Middleware]
    ↓
[Recovery Middleware]
    ↓
[Auth Middleware] ← Sets ctx.Ctx with User & OrgID
    ↓
[Cache Middleware] ← Can skip cache based on context
    ↓
[Handler Wrapper] ← wrapHandler() extracts ctx.Ctx
    ↓
[Service Handler] ← Receives ctx.Ctx
    ↓
[Data Layer] ← Uses ctx for isolation & cancellation
    ↓
[Database/Cache]
    ↓
[Response]
```

### Async Event Context
```
HTTP /ingest or /identify Request
    ↓
[Validation & Enrichment]
    ↓
[go h.ingestor.Ingest() or h.identifier.Identify()]
    ↓
[NATS Publish to events:raw or events:identify]
    ↓
[Background Consumer] ← Separate context, started in DI
    ↓
[Message Processing with enrichment stages]
    ↓
[ClickHouse AsyncInsert]
    ↓
[NATS Ack/Nak]
```

---

## 8. Metrics Currently Collected

**Location**: `/home/user/zori/internal/metrics/metrics.go`

### Ingestion Metrics
- `zori_ingest_requests_total` - Total ingestion requests by project/org/status
- `zori_ingest_request_duration_seconds` - Request duration histogram
- `zori_ingest_errors_total` - Errors by type
- `zori_event_dedupe_total` - Event deduplication results (new/duplicate/error)
- `zori_events_total` - Event count by type and name

### NATS Processing Metrics
- `zori_nats_messages_processed_total` - Messages by stream/consumer/status
- `zori_nats_message_processing_duration_seconds` - Processing duration
- `zori_nats_message_ack_total` - Ack/Nak counts
- `zori_nats_consumer_lag_messages` - Pending messages in consumer
- `zori_nats_stage_duration_seconds` - Per-stage duration (6 stages)
- `zori_nats_stage_errors_total` - Errors per processing stage

### ClickHouse Metrics
- `zori_clickhouse_insert_duration_seconds` - Insert latency by table
- `zori_clickhouse_insert_errors_total` - Insert errors by type

**Metrics Server**: `/home/user/zori/internal/metrics/server.go` (Prometheus format on port 9090)

---

## Conclusion

Zori is a well-architected analytics platform with clear separation of concerns:

1. **Ingestion Path** - High-performance fasthttp server for collecting events
2. **Processing Path** - NATS JetStream-based async processing pipeline with enrichment stages
3. **Query Path** - Echo REST API with caching and PostgreSQL/ClickHouse backends
4. **Authentication** - Flexible auth supporting both Clerk and OSS JWT

**Key Opportunities for Tracing**:
- HTTP request entry points (`wrapHandler`)
- NATS message consumption loops
- Processing stages in event enrichment
- Database operations (Query/Insert)
- Cache operations
- Authentication verification
- Background async operations

The architecture is ready for comprehensive distributed tracing implementation starting with the entry points listed above.

