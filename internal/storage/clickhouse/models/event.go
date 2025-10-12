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

	// Timestamps
	ClientTimestampUTC time.Time `ch:"client_timestamp_utc"`
	ServerTimestampUTC time.Time `ch:"server_timestamp_utc,default:now()"`

	// Request metadata
	UserAgent   string `ch:"user_agent"`
	IP          string `ch:"ip"`
	ReferrerURL string `ch:"referrer_url"`
	PageURL     string `ch:"page_url"`
	PathPath    string `ch:"path_path"`

	// Processed Request Medata
	ReferrerDomain *string `ch:"referrer_domain"`
	ReferrerPath   *string `ch:"referrer_path"`

	BrowserName *string `ch:"browser_name"`
	OsName      *string `ch:"os_name"`
	DeviceType  *string `ch:"device_type"`

	// Interaction data
	ClickOn        *string  `ch:"click_on"`
	ClickPositionX *float64 `ch:"click_position_x"`
	ClickPositionY *float64 `ch:"click_position_y"`

	//UTM parameters
	UTMParameters map[string]string `ch:"utm_parameters"`

	// Custom properties
	CustomProperties string `ch:"custom_properties"`

	// Organization hierarchy
	ProjectID      string `ch:"project_id"`
	OrganizationID string `ch:"organization_id"`

	// Location
	LocationCountryISO *string `ch:"location_country_iso"`
	LocationCity       *string `ch:"location_city"`

	// Metadata
	CreatedAt time.Time `ch:"created_at,type:DateTime,default:now()"`

	// Materialized columns for common UTM parameters
	UTMSource   string `ch:"utm_source,materialized:utm_parameters['utm_source'],scanonly"`
	UTMMedium   string `ch:"utm_medium,materialized:utm_parameters['utm_medium'],scanonly"`
	UTMCampaign string `ch:"utm_campaign,materialized:utm_parameters['utm_campaign'],scanonly"`
}
