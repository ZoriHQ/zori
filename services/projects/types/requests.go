package types

type CreateProjectRequest struct {
	Name           string `json:"name" validate:"required" example:"My Awesome Project"`
	WebsiteURL     string `json:"website_url" validate:"required,url" example:"https://example.com"`
	AllowLocalHost bool   `json:"allow_localhost" example:"false"`
}

type UpdateProjectRequest struct {
	Name           string `json:"name" example:"Updated Project Name"`
	WebsiteURL     string `json:"website_url" validate:"omitempty,url" example:"https://updated-example.com"`
	AllowLocalHost bool   `json:"allow_localhost" example:"true"`
}
