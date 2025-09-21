package telemetry

import "context"

type WebContext struct {
	OrganizationID string
	UserID         string
	ProjectID      string

	context.Context
}

func NewWebContext(ctx context.Context, organizationID, userID, projectID string) *WebContext {
	return &WebContext{
		OrganizationID: organizationID,
		UserID:         userID,
		ProjectID:      projectID,
		Context:        ctx,
	}
}
