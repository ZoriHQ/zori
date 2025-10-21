package types

import "time"

// IdentifyEventV1 represents user identification data sent from the tracking script
type IdentifyEventV1 struct {
	// VisitorID is the anonymous identifier that we're attaching identity to
	VisitorID string `json:"visitor_id"`
	// SessionID is the current session identifier
	SessionID string `json:"session_id"`

	// ClientTimeStampUTC is when the identify call was made on the client
	ClientTimestampUTC time.Time `json:"client_timestamp_utc"`

	// UserAgent and IP will be overridden by the server
	UserAgent string `json:"user_agent"`
	IP        string `json:"ip"`

	// PageURL is the path where the identify call was made
	PageURL string `json:"page_url"`
	// Host is the domain name where the identify call was made
	Host string `json:"host"`

	// Identity fields
	AppID    *string `json:"app_id"`     // Customer's internal app/user ID
	Email    *string `json:"email"`      // User email address
	Fullname *string `json:"fullname"`   // User's full name
	Phone    *string `json:"phone"`      // User's phone number

	// AdditionalProperties contains any custom traits/properties
	AdditionalProperties map[string]any `json:"additional_properties"`
}

// IdentifyEventFrameV1 wraps the identify event with project/org context
type IdentifyEventFrameV1 struct {
	*IdentifyEventV1
	ProjectID      string `json:"project_id"`
	OrganizationID string `json:"organization_id"`
}
