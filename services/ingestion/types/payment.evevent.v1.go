package types

type PaymentService string

const (
	PaymentServiceStripe       PaymentService = "stripe"
	PaymentServicePayPal       PaymentService = "paypal"
	PaymentServiceLemonSqueezy PaymentService = "lemon_squeezy"
	PaymentServiceSquare       PaymentService = "square"
)

type PaymentEvent struct {
	ID                  string         `json:"id"`
	AttributedVisitorID string         `json:"attributed_visitor_id"`
	PaymentService      PaymentService `json:"payment_service"`
	ItemName            string         `json:"item_name"`
	// the amount is stored in cents
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
	// the timestamp is stored in unix timestamp format
	Timestamp int64 `json:"timestamp"`
}
