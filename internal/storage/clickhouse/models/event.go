package models

import (
	"time"

	"github.com/uptrace/go-clickhouse/ch"
)

type Event struct {
	ch.CHModel `ch:"events,partition:toYYYYMM(client_timestamp_utc),order:organization_id,order:project_id,order:client_timestamp_utc,order:visitor_id"`

	// Event identification
	EventName              *string `ch:"event_name"`
	ClientGeneratedEventID string  `ch:"client_generated_event_id"`
	VisitorID              string  `ch:"visitor_id"`
	SessionID              string  `ch:"session_id"`

	// Identity fields (set when visitor is identified)
	UserID     *string `ch:"user_id"`     // Authenticated user ID
	ExternalID *string `ch:"external_id"` // Customer-provided external ID
	EmailHash  *string `ch:"email_hash"`  // SHA256 hash of email for privacy-safe analytics

	// Timestamps
	ClientTimestampUTC time.Time `ch:"client_timestamp_utc"`
	ServerTimestampUTC time.Time `ch:"server_timestamp_utc"`

	// Request metadata
	UserAgent   string `ch:"user_agent"`
	IP          string `ch:"ip"`
	ReferrerURL string `ch:"referrer_url"`
	PageURL     string `ch:"page_url"`
	PagePath    string `ch:"page_path"`
	Host        string `ch:"host"` // Domain name (e.g., "mention.click")

	// Processed Request Medata
	ReferrerDomain *string `ch:"referrer_domain"`
	ReferrerPath   *string `ch:"referrer_path"`

	BrowserName *string `ch:"browser_name"`
	OsName      *string `ch:"os_name"`
	DeviceType  *string `ch:"device_type"`

	// Interaction data - Click element details
	ClickElementTag       *string `ch:"click_element_tag"`       // HTML tag of clicked element
	ClickElementSelector  *string `ch:"click_element_selector"`  // CSS selector of clicked element
	ClickElementText      *string `ch:"click_element_text"`      // Text content of clicked element
	ClickPositionX        *float64 `ch:"click_position_x"`        // X coordinate of click
	ClickPositionY        *float64 `ch:"click_position_y"`        // Y coordinate of click
	ClickScreenWidth      *uint16  `ch:"click_screen_width"`      // Browser viewport width
	ClickScreenHeight     *uint16  `ch:"click_screen_height"`     // Browser viewport height

	// Click element classification
	ClickElementType     *string `ch:"click_element_type"`     // Classified element type (button, link, input, etc.)
	ClickElementCategory *string `ch:"click_element_category"` // High-level category (cta, navigation, form, etc.)
	IsCTAClick           *bool   `ch:"is_cta_click"`           // Whether this is a CTA click
	LinkDestination      *string `ch:"link_destination"`       // Destination URL for link clicks
	IsExternalLink       *bool   `ch:"is_external_link"`       // Whether link is external
	IsDownloadLink       *bool   `ch:"is_download_link"`       // Whether link is a download

	//UTM parameters
	UTMParameters map[string]string `ch:"utm_parameters"`

	// Custom properties
	CustomProperties string `ch:"custom_properties"`

	// Organization hierarchy
	ProjectID      string `ch:"project_id"`
	OrganizationID string `ch:"organization_id"`

	// Location
	LocationCountryISO *string  `ch:"location_country_iso"`
	LocationCity       *string  `ch:"location_city"`
	LocationLatitude   *float64 `ch:"location_latitude"`
	LocationLongitude  *float64 `ch:"location_longitude"`

	// Metadata
	CreatedAt time.Time `ch:"created_at,type:DateTime,default:now()"`

	// Materialized columns for common UTM parameters
	UTMSource   string `ch:"utm_source,materialized:utm_parameters['utm_source'],scanonly"`
	UTMMedium   string `ch:"utm_medium,materialized:utm_parameters['utm_medium'],scanonly"`
	UTMCampaign string `ch:"utm_campaign,materialized:utm_parameters['utm_campaign'],scanonly"`
}
