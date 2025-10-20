package types

import "time"

// PaymentEventFrame represents a payment event ready for processing
type PaymentEventFrame struct {
	// Payment identification
	PaymentID string `json:"payment_id"`
	VisitorID *string `json:"visitor_id"` // From metadata

	// Payment details
	ProviderType   string `json:"provider_type"`
	PaymentStatus  string `json:"payment_status"`
	ProductName    string `json:"product_name"`

	// Amounts (in smallest currency unit, e.g., cents)
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`

	// Timestamps
	PaymentTimestamp time.Time `json:"payment_timestamp"`

	// Organization hierarchy
	ProjectID      string `json:"project_id"`
	OrganizationID string `json:"organization_id"`

	// Additional metadata
	Metadata map[string]string `json:"metadata"`
}
