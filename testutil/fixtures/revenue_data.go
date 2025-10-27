package fixtures

import (
	"context"
	"fmt"
	"testing"
	"time"

	"zori/di"
	paymentTypes "zori/services/payments/types"

	"github.com/google/uuid"
)

// VisitorPaymentScenario represents a visitor with their attribution and payment behavior
type VisitorPaymentScenario struct {
	VisitorID       string
	SessionID       string
	FirstVisitTime  time.Time
	ReferrerDomain  string
	UTMSource       string
	UTMMedium       string
	UTMCampaign     string
	Payments        []PaymentScenario
	WillPurchase    bool
}

// PaymentScenario represents a single payment
type PaymentScenario struct {
	Amount    int64
	Timestamp time.Time
	Status    string // succeeded, failed, refunded
	Currency  string
}

// RevenueTestDataBuilder helps build complex revenue test scenarios
type RevenueTestDataBuilder struct {
	scenarios []VisitorPaymentScenario
}

// NewRevenueTestDataBuilder creates a new builder
func NewRevenueTestDataBuilder() *RevenueTestDataBuilder {
	return &RevenueTestDataBuilder{
		scenarios: make([]VisitorPaymentScenario, 0),
	}
}

// AddVisitor adds a visitor scenario
func (b *RevenueTestDataBuilder) AddVisitor(scenario VisitorPaymentScenario) *RevenueTestDataBuilder {
	if scenario.VisitorID == "" {
		scenario.VisitorID = uuid.New().String()
	}
	if scenario.SessionID == "" {
		scenario.SessionID = uuid.New().String()
	}
	if scenario.FirstVisitTime.IsZero() {
		scenario.FirstVisitTime = time.Now().UTC()
	}
	b.scenarios = append(b.scenarios, scenario)
	return b
}

// AddSimpleConvertingVisitor adds a visitor who makes a single purchase
func (b *RevenueTestDataBuilder) AddSimpleConvertingVisitor(amount int64, utmSource string) *RevenueTestDataBuilder {
	return b.AddVisitor(VisitorPaymentScenario{
		VisitorID:      uuid.New().String(),
		SessionID:      uuid.New().String(),
		FirstVisitTime: time.Now().UTC().Add(-2 * time.Hour),
		ReferrerDomain: utmSource + ".com",
		UTMSource:      utmSource,
		UTMMedium:      "cpc",
		UTMCampaign:    "test-campaign",
		WillPurchase:   true,
		Payments: []PaymentScenario{
			{
				Amount:    amount,
				Timestamp: time.Now().UTC().Add(-1 * time.Hour),
				Status:    "succeeded",
				Currency:  "USD",
			},
		},
	})
}

// AddRepeatCustomer adds a visitor who makes multiple purchases
func (b *RevenueTestDataBuilder) AddRepeatCustomer(amounts []int64, utmSource string) *RevenueTestDataBuilder {
	payments := make([]PaymentScenario, len(amounts))
	for i, amount := range amounts {
		payments[i] = PaymentScenario{
			Amount:    amount,
			Timestamp: time.Now().UTC().Add(-time.Duration(len(amounts)-i) * time.Hour),
			Status:    "succeeded",
			Currency:  "USD",
		}
	}

	return b.AddVisitor(VisitorPaymentScenario{
		VisitorID:      uuid.New().String(),
		SessionID:      uuid.New().String(),
		FirstVisitTime: time.Now().UTC().Add(-24 * time.Hour),
		ReferrerDomain: utmSource + ".com",
		UTMSource:      utmSource,
		UTMMedium:      "cpc",
		UTMCampaign:    "test-campaign",
		WillPurchase:   true,
		Payments:       payments,
	})
}

// AddNonConvertingVisitor adds a visitor who doesn't make any purchases
func (b *RevenueTestDataBuilder) AddNonConvertingVisitor(utmSource string) *RevenueTestDataBuilder {
	return b.AddVisitor(VisitorPaymentScenario{
		VisitorID:      uuid.New().String(),
		SessionID:      uuid.New().String(),
		FirstVisitTime: time.Now().UTC().Add(-1 * time.Hour),
		ReferrerDomain: utmSource + ".com",
		UTMSource:      utmSource,
		UTMMedium:      "cpc",
		UTMCampaign:    "test-campaign",
		WillPurchase:   false,
		Payments:       []PaymentScenario{},
	})
}

// AddDirectTrafficVisitor adds a visitor with direct traffic (no attribution)
func (b *RevenueTestDataBuilder) AddDirectTrafficVisitor(amount int64) *RevenueTestDataBuilder {
	return b.AddVisitor(VisitorPaymentScenario{
		VisitorID:      uuid.New().String(),
		SessionID:      uuid.New().String(),
		FirstVisitTime: time.Now().UTC().Add(-2 * time.Hour),
		ReferrerDomain: "",
		UTMSource:      "",
		UTMMedium:      "",
		UTMCampaign:    "",
		WillPurchase:   true,
		Payments: []PaymentScenario{
			{
				Amount:    amount,
				Timestamp: time.Now().UTC().Add(-1 * time.Hour),
				Status:    "succeeded",
				Currency:  "USD",
			},
		},
	})
}

// GetScenarios returns all scenarios
func (b *RevenueTestDataBuilder) GetScenarios() []VisitorPaymentScenario {
	return b.scenarios
}

// GetConversionStats returns expected conversion statistics
func (b *RevenueTestDataBuilder) GetConversionStats() (totalVisitors, payingCustomers int, totalRevenue int64) {
	totalVisitors = len(b.scenarios)
	for _, s := range b.scenarios {
		if s.WillPurchase && len(s.Payments) > 0 {
			payingCustomers++
			for _, p := range s.Payments {
				if p.Status == "succeeded" {
					totalRevenue += p.Amount
				}
			}
		}
	}
	return totalVisitors, payingCustomers, totalRevenue
}

// GetRepeatPurchaseStats returns repeat purchase statistics
func (b *RevenueTestDataBuilder) GetRepeatPurchaseStats() (payingCustomers, repeatCustomers int) {
	for _, s := range b.scenarios {
		successfulPayments := 0
		for _, p := range s.Payments {
			if p.Status == "succeeded" {
				successfulPayments++
			}
		}
		if successfulPayments > 0 {
			payingCustomers++
			if successfulPayments > 1 {
				repeatCustomers++
			}
		}
	}
	return payingCustomers, repeatCustomers
}

// InsertTestData inserts all test data into the database
func (b *RevenueTestDataBuilder) InsertTestData(t *testing.T, tc *di.TestContainer, project *ProjectFixture) error {
	t.Helper()

	// Step 1: Send visitor events
	for _, scenario := range b.scenarios {
		builder := NewEventBuilder().
			WithVisitorID(scenario.VisitorID).
			WithSessionID(scenario.SessionID).
			WithPageURL("/").
			WithHost(project.Domain)

		if scenario.UTMSource != "" {
			builder = builder.
				WithUTMSource(scenario.UTMSource).
				WithUTMMedium(scenario.UTMMedium).
				WithUTMCampaign(scenario.UTMCampaign)
		}

		if scenario.ReferrerDomain != "" {
			builder = builder.WithReferrer("https://" + scenario.ReferrerDomain)
		}

		event := builder.Build()
		err := SendEventToTestServer(t, tc, project, event)
		if err != nil {
			return fmt.Errorf("failed to send visitor event for %s: %w", scenario.VisitorID, err)
		}
	}

	// Wait for events to be processed
	expectedEventCount := len(b.scenarios)
	_, err := WaitForEvents(t, tc, QueryEventsOptions{
		ProjectID: &project.ID,
	}, expectedEventCount, 15*time.Second)
	if err != nil {
		return fmt.Errorf("failed waiting for events: %w", err)
	}

	// Give attribution system time to process
	time.Sleep(2 * time.Second)

	// Step 2: Send payment events
	totalPayments := 0
	for _, scenario := range b.scenarios {
		if !scenario.WillPurchase || len(scenario.Payments) == 0 {
			continue
		}

		for _, payment := range scenario.Payments {
			paymentEvent := NewPaymentBuilder(project.ID, project.OrganizationID).
				WithVisitorID(scenario.VisitorID).
				WithAmount(payment.Amount).
				WithPaymentStatus(payment.Status).
				WithCurrency(payment.Currency).
				WithProductName("Test Product").
				Build()

			err := SendPaymentToNATS(t, tc, paymentEvent)
			if err != nil {
				return fmt.Errorf("failed to send payment for %s: %w", scenario.VisitorID, err)
			}
			totalPayments++
			time.Sleep(50 * time.Millisecond) // Small delay between payments
		}
	}

	// Wait for payments to be processed
	if totalPayments > 0 {
		succeededStatus := "succeeded"
		_, err = WaitForPayments(t, tc, QueryPaymentEventsOptions{
			ProjectID:     &project.ID,
			PaymentStatus: &succeededStatus,
		}, totalPayments, 15*time.Second)
		if err != nil {
			return fmt.Errorf("failed waiting for payments: %w", err)
		}

		// Give revenue aggregation system time to process
		time.Sleep(3 * time.Second)
	}

	return nil
}

// InsertPaymentEventsDirect inserts payment events directly into ClickHouse (bypasses NATS)
func InsertPaymentEventsDirect(t *testing.T, tc *di.TestContainer, project *ProjectFixture, payments []paymentTypes.PaymentEventFrame) error {
	t.Helper()

	ctx := context.Background()
	query := `INSERT INTO payment_events (
		payment_id, visitor_id, provider_type, payment_status, product_name,
		amount, currency, payment_timestamp_utc, project_id, organization_id, metadata
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	for _, p := range payments {
		err := tc.ClickHouse.Db().Exec(ctx, query,
			p.PaymentID,
			p.VisitorID,
			p.ProviderType,
			p.PaymentStatus,
			p.ProductName,
			p.Amount,
			p.Currency,
			p.PaymentTimestamp,
			p.ProjectID,
			p.OrganizationID,
			p.Metadata,
		)
		if err != nil {
			return fmt.Errorf("failed to insert payment %s: %w", p.PaymentID, err)
		}
	}

	return nil
}

// InsertVisitorAttributionDirect inserts visitor attribution data directly into ClickHouse
func InsertVisitorAttributionDirect(t *testing.T, tc *di.TestContainer, projectID, organizationID, visitorID, referrerDomain, utmSource, utmMedium, utmCampaign string, timestamp time.Time) error {
	t.Helper()

	ctx := context.Background()

	// Insert into the aggregating table using argMinState
	query := `INSERT INTO visitor_first_touch_attribution (
		organization_id, project_id, visitor_id,
		first_referrer_domain, first_utm_source, first_utm_medium, first_utm_campaign
	) VALUES (?, ?, ?,
		argMinState(?, ?),
		argMinState(?, ?),
		argMinState(?, ?),
		argMinState(?, ?)
	)`

	err := tc.ClickHouse.Db().Exec(ctx, query,
		organizationID,
		projectID,
		visitorID,
		&referrerDomain, timestamp,
		utmSource, timestamp,
		utmMedium, timestamp,
		utmCampaign, timestamp,
	)
	if err != nil {
		return fmt.Errorf("failed to insert attribution for visitor %s: %w", visitorID, err)
	}

	return nil
}

// CreateTimeDistributedPayments creates payments distributed over a time range
func CreateTimeDistributedPayments(startTime time.Time, endTime time.Time, count int, baseAmount int64) []PaymentScenario {
	payments := make([]PaymentScenario, count)
	duration := endTime.Sub(startTime)
	interval := duration / time.Duration(count)

	for i := 0; i < count; i++ {
		payments[i] = PaymentScenario{
			Amount:    baseAmount + int64(i*100), // Vary amounts slightly
			Timestamp: startTime.Add(interval * time.Duration(i)),
			Status:    "succeeded",
			Currency:  "USD",
		}
	}

	return payments
}
