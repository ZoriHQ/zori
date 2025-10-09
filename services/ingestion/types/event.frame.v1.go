package types

// ClientEventFrameV1 represents an event sent from a tracking script to Zori for ingestion.
type ClientEventFrameV1 struct {
	*ClientEventV1
	ProjectID      string `json:"project_id"`
	OrganizationID string `json:"organization_id"`
}
