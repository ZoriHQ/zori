package fixtures

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"zori/di"
	"zori/internal/storage/clickhouse/models"
	"zori/services/payments/types"
	"zori/testutil"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// PaymentBuilder provides a fluent API to build test payment events
type PaymentBuilder struct {
	payment types.PaymentEventFrame
}

// NewPaymentBuilder creates a new PaymentBuilder with sensible defaults
func NewPaymentBuilder(projectID, organizationID string) *PaymentBuilder {
	return &PaymentBuilder{
		payment: types.PaymentEventFrame{
			PaymentID:        uuid.New().String(),
			ProviderType:     "stripe",
			PaymentStatus:    "succeeded",
			ProductName:      "Test Product",
			Amount:           9900, // $99.00 in cents
			Currency:         "USD",
			PaymentTimestamp: time.Now().UTC(),
			ProjectID:        projectID,
			OrganizationID:   organizationID,
			Metadata:         make(map[string]string),
		},
	}
}

// WithPaymentID sets the payment ID
func (b *PaymentBuilder) WithPaymentID(paymentID string) *PaymentBuilder {
	b.payment.PaymentID = paymentID
	return b
}

// WithVisitorID sets the visitor ID
func (b *PaymentBuilder) WithVisitorID(visitorID string) *PaymentBuilder {
	b.payment.VisitorID = &visitorID
	return b
}

// WithProviderType sets the payment provider type
func (b *PaymentBuilder) WithProviderType(providerType string) *PaymentBuilder {
	b.payment.ProviderType = providerType
	return b
}

// WithPaymentStatus sets the payment status
func (b *PaymentBuilder) WithPaymentStatus(status string) *PaymentBuilder {
	b.payment.PaymentStatus = status
	return b
}

// WithProductName sets the product name
func (b *PaymentBuilder) WithProductName(productName string) *PaymentBuilder {
	b.payment.ProductName = productName
	return b
}

// WithAmount sets the payment amount (in cents)
func (b *PaymentBuilder) WithAmount(amount int64) *PaymentBuilder {
	b.payment.Amount = amount
	return b
}

// WithCurrency sets the currency code
func (b *PaymentBuilder) WithCurrency(currency string) *PaymentBuilder {
	b.payment.Currency = currency
	return b
}

// WithPaymentTimestamp sets the payment timestamp
func (b *PaymentBuilder) WithPaymentTimestamp(timestamp time.Time) *PaymentBuilder {
	b.payment.PaymentTimestamp = timestamp
	return b
}

// WithMetadata adds metadata to the payment
func (b *PaymentBuilder) WithMetadata(key, value string) *PaymentBuilder {
	if b.payment.Metadata == nil {
		b.payment.Metadata = make(map[string]string)
	}
	b.payment.Metadata[key] = value
	return b
}

// Build returns the built payment event
func (b *PaymentBuilder) Build() types.PaymentEventFrame {
	return b.payment
}

// SendPaymentToNATS sends a payment event to the NATS stream for processing
func SendPaymentToNATS(t *testing.T, tc *di.TestContainer, payment types.PaymentEventFrame) error {
	t.Helper()

	paymentJSON, err := json.Marshal(payment)
	require.NoError(t, err)

	err = tc.NATS.GetConnection().Publish("payments:raw", paymentJSON)
	if err != nil {
		return fmt.Errorf("failed to publish payment to NATS: %w", err)
	}

	return nil
}

// SendPaymentsToNATS sends multiple payment events to the NATS stream
func SendPaymentsToNATS(t *testing.T, tc *di.TestContainer, payments []types.PaymentEventFrame) error {
	t.Helper()

	for i, payment := range payments {
		if err := SendPaymentToNATS(t, tc, payment); err != nil {
			return fmt.Errorf("failed to send payment %d: %w", i, err)
		}
		// Small delay between payments to avoid overwhelming the system
		time.Sleep(10 * time.Millisecond)
	}

	return nil
}

// QueryPaymentEventsOptions holds options for querying payment events
type QueryPaymentEventsOptions struct {
	ProjectID      *string
	OrganizationID *string
	VisitorID      *string
	PaymentID      *string
	PaymentStatus  *string
	Limit          int
}

// QueryPaymentEvents retrieves payment events from ClickHouse based on the provided options
func QueryPaymentEvents(t *testing.T, tc *di.TestContainer, opts QueryPaymentEventsOptions) ([]models.PaymentEvent, error) {
	t.Helper()

	query := `SELECT
		payment_id, visitor_id, provider_type, payment_status, product_name,
		amount, currency, payment_timestamp_utc, server_timestamp_utc,
		project_id, organization_id, metadata, created_at
	FROM payment_events WHERE 1=1`
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

	if opts.PaymentID != nil {
		query += " AND payment_id = ?"
		args = append(args, *opts.PaymentID)
	}

	if opts.PaymentStatus != nil {
		query += " AND payment_status = ?"
		args = append(args, *opts.PaymentStatus)
	}

	query += " ORDER BY payment_timestamp_utc DESC"

	if opts.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", opts.Limit)
	}

	var payments []models.PaymentEvent
	err := tc.ClickHouse.Db().Select(context.Background(), &payments, query, args...)
	if err != nil {
		return nil, err
	}

	return payments, nil
}

// WaitForPayments waits for payment events to appear in ClickHouse with the given options
// Returns the payments once they appear or times out after the specified duration
func WaitForPayments(t *testing.T, tc *di.TestContainer, opts QueryPaymentEventsOptions, expectedCount int, timeout time.Duration) ([]models.PaymentEvent, error) {
	t.Helper()

	var payments []models.PaymentEvent
	ctx := context.Background()

	err := testutil.WaitForCondition(ctx, timeout, 100*time.Millisecond, func() (bool, error) {
		var err error
		payments, err = QueryPaymentEvents(t, tc, opts)
		if err != nil {
			return false, err
		}

		return len(payments) >= expectedCount, nil
	})

	if err != nil {
		return payments, fmt.Errorf("timeout waiting for payments: %w (got %d payments, expected %d)", err, len(payments), expectedCount)
	}

	return payments, nil
}

// CountPaymentEvents counts the number of payment events matching the given options
func CountPaymentEvents(t *testing.T, tc *di.TestContainer, opts QueryPaymentEventsOptions) (int, error) {
	t.Helper()

	query := "SELECT count(*) FROM payment_events WHERE 1=1"
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

	if opts.PaymentID != nil {
		query += " AND payment_id = ?"
		args = append(args, *opts.PaymentID)
	}

	if opts.PaymentStatus != nil {
		query += " AND payment_status = ?"
		args = append(args, *opts.PaymentStatus)
	}

	var count uint64
	err := tc.ClickHouse.Db().QueryRow(context.Background(), query, args...).Scan(&count)
	if err != nil {
		return 0, err
	}

	return int(count), nil
}
