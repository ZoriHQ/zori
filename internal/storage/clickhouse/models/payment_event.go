package models

import (
	"time"

	"github.com/uptrace/go-clickhouse/ch"
)

type PaymentStatus string

const (
	PaymentStatusSucceeded PaymentStatus = "succeeded"
	PaymentStatusFailed    PaymentStatus = "failed"
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusRefunded  PaymentStatus = "refunded"
)

type PaymentEvent struct {
	ch.CHModel `ch:"payment_events,partition:toYYYYMM(payment_timestamp_utc),order:organization_id,order:project_id,order:payment_timestamp_utc,order:payment_id"`

	// Payment identification
	PaymentID string  `ch:"payment_id"`
	VisitorID *string `ch:"visitor_id"` // Nullable for unattributed payments

	// Payment details
	ProviderType  string `ch:"provider_type"`  // LowCardinality
	PaymentStatus string `ch:"payment_status"` // LowCardinality
	ProductName   string `ch:"product_name"`

	// Amounts (stored in smallest currency unit, e.g., cents)
	Amount   int64  `ch:"amount"`
	Currency string `ch:"currency"` // ISO 4217 currency code

	// Timestamps
	PaymentTimestampUTC time.Time `ch:"payment_timestamp_utc"`
	ServerTimestampUTC  time.Time `ch:"server_timestamp_utc"`

	// Organization hierarchy
	ProjectID      string `ch:"project_id"`
	OrganizationID string `ch:"organization_id"`

	// Additional metadata
	Metadata map[string]string `ch:"metadata"`

	// Metadata
	CreatedAt time.Time `ch:"created_at"`
}
