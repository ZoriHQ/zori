package types

// ClientEventFrameV1 represents an event sent from a tracking script to Zori for ingestion.
type ClientEventFrameV1 struct {
	*ClientEventV1
	ProjectID      string `json:"project_id"`
	OrganizationID string `json:"organization_id"`

	LocationCountryISO *string  `json:"location_country_iso"`
	LocationCity       *string  `json:"location_city"`
	LocationLatitude   *float64 `json:"location_latitude"`
	LocationLongitude  *float64 `json:"location_longitude"`

	BrowserName *string `json:"browser_name"`
	OsName      *string `json:"os_name"`
	DeviceType  *string `json:"device_type"`

	ReferredDomain *string `json:"referred_domain"`
	ReferrerPath   *string `json:"referrer_path"`

	PagePath *string `json:"page_path"`

	// Identity fields (processed from ClientEventV1)
	EmailHash *string `json:"email_hash"` // SHA256 hash of email for privacy-safe storage

	// Click element classification (computed during processing)
	ClickElementType     *string `json:"click_element_type"`
	ClickElementCategory *string `json:"click_element_category"`
	IsCTAClick           *bool   `json:"is_cta_click"`
	LinkDestination      *string `json:"link_destination"`
	IsExternalLink       *bool   `json:"is_external_link"`
	IsDownloadLink       *bool   `json:"is_download_link"`

	PagePattern *string `json:"page_pattern"`
}
