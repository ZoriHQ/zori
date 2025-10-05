package types

type CreateProjectRequest struct {
	Name           string `json:"name" validate:"required"`
	WebsiteURL     string `json:"website_url" validate:"required,url"`
	AllowLocalHost bool   `json:"allow_localhost"`
}

type UpdateProjectRequest struct {
	Name           string `json:"name"`
	WebsiteURL     string `json:"website_url" validate:"omitempty,url"`
	AllowLocalHost bool   `json:"allow_localhost"`
}
