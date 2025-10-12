package types

// ClientEventFrameV1 represents an event sent from a tracking script to Zori for ingestion.
type ClientEventFrameV1 struct {
	*ClientEventV1
	ProjectID      string `json:"project_id"`
	OrganizationID string `json:"organization_id"`

	LocationCountryISO *string `json:"location_country_iso"`
	LocationCity       *string `json:"location_city"`

	BrowserName *string `json:"browser_name"`
	OsName      *string `json:"os_name"`
	DeviceType  *string `json:"device_type"`

	ReferredDomain *string `json:"referred_domain"`
	ReferrerPath   *string `json:"referrer_path"`

	PagePath *string `json:"page_path"`
}
