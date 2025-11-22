package types

import (
	"encoding/json"
	"time"
	"zori/internal/storage/postgres/models"
)

type FunnelResponse struct {
	ID             string            `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ProjectID      string            `json:"project_id" example:"660e8400-e29b-41d4-a716-446655440001"`
	OrganizationID string            `json:"organization_id" example:"770e8400-e29b-41d4-a716-446655440002"`
	Name           string            `json:"name" example:"Checkout Funnel"`
	Description    *string           `json:"description" example:"Tracks user checkout flow"`
	FunnelType     models.FunnelType `json:"funnel_type" example:"sequential"`
	WindowSeconds  int               `json:"window_seconds" example:"86400"`
	IsStrict       bool              `json:"is_strict" example:"false"`
	DepthConfig    *DepthConfig      `json:"depth_config,omitempty"`
	Steps          []StepResponse    `json:"steps"`
	CreatedAt      time.Time         `json:"created_at" example:"2024-01-15T10:30:00Z"`
	UpdatedAt      time.Time         `json:"updated_at" example:"2024-01-15T10:30:00Z"`
}

type StepResponse struct {
	ID        string        `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	StepOrder int           `json:"step_order" example:"1"`
	Name      string        `json:"name" example:"View Product"`
	Condition StepCondition `json:"condition"`
}

type ListFunnelsResponse struct {
	Funnels []*FunnelResponse `json:"funnels"`
	Total   int               `json:"total" example:"5"`
}

type FunnelStepResult struct {
	StepOrder      int     `json:"step_order" example:"1"`
	Name           string  `json:"name" example:"View Product"`
	Visitors       uint64  `json:"visitors" example:"1000"`
	ConversionRate float64 `json:"conversion_rate" example:"0.85"`
	DropoffRate    float64 `json:"dropoff_rate" example:"0.15"`
}

type FunnelAnalysisResponse struct {
	FunnelID           string             `json:"funnel_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	FunnelName         string             `json:"funnel_name" example:"Checkout Funnel"`
	TotalVisitors      uint64             `json:"total_visitors" example:"10000"`
	ConvertedVisitors  uint64             `json:"converted_visitors" example:"1500"`
	OverallConversion  float64            `json:"overall_conversion" example:"0.15"`
	Steps              []FunnelStepResult `json:"steps"`
	AnalyzedAt         time.Time          `json:"analyzed_at" example:"2024-01-15T10:30:00Z"`
}

type DepthAnalysisResult struct {
	Depth    int    `json:"depth" example:"3"`
	Visitors uint64 `json:"visitors" example:"4200"`
}

type DepthFunnelAnalysisResponse struct {
	FunnelID          string                `json:"funnel_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	FunnelName        string                `json:"funnel_name" example:"Pricing Page Engagement"`
	TotalSessions     uint64                `json:"total_sessions" example:"10000"`
	DepthDistribution []DepthAnalysisResult `json:"depth_distribution"`
	AnalyzedAt        time.Time             `json:"analyzed_at" example:"2024-01-15T10:30:00Z"`
}

type UnifiedFunnelAnalysisResponse struct {
	FunnelID   string            `json:"funnel_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	FunnelName string            `json:"funnel_name" example:"Checkout Funnel"`
	FunnelType models.FunnelType `json:"funnel_type" example:"sequential"`
	AnalyzedAt time.Time         `json:"analyzed_at" example:"2024-01-15T10:30:00Z"`

	TotalVisitors     uint64             `json:"total_visitors,omitempty" example:"10000"`
	ConvertedVisitors uint64             `json:"converted_visitors,omitempty" example:"1500"`
	OverallConversion float64            `json:"overall_conversion,omitempty" example:"0.15"`
	Steps             []FunnelStepResult `json:"steps,omitempty"`

	TotalSessions     uint64                `json:"total_sessions,omitempty" example:"10000"`
	DepthDistribution []DepthAnalysisResult `json:"depth_distribution,omitempty"`
}

func ToFunnelResponse(funnel *models.Funnel) *FunnelResponse {
	response := &FunnelResponse{
		ID:             funnel.ID,
		ProjectID:      funnel.ProjectID,
		OrganizationID: funnel.OrganizationID,
		Name:           funnel.Name,
		Description:    funnel.Description,
		FunnelType:     funnel.FunnelType,
		WindowSeconds:  funnel.WindowSeconds,
		IsStrict:       funnel.IsStrict,
		CreatedAt:      funnel.CreatedAt,
		UpdatedAt:      funnel.UpdatedAt,
		Steps:          make([]StepResponse, 0),
	}

	if funnel.DepthConfig != nil && len(funnel.DepthConfig) > 0 {
		var depthConfig DepthConfig
		if err := json.Unmarshal(funnel.DepthConfig, &depthConfig); err == nil {
			response.DepthConfig = &depthConfig
		}
	}

	if funnel.Steps != nil {
		for _, step := range funnel.Steps {
			stepResponse := StepResponse{
				ID:        step.ID,
				StepOrder: step.StepOrder,
				Name:      step.Name,
			}
			if step.Condition != nil {
				var condition StepCondition
				if err := json.Unmarshal(step.Condition, &condition); err == nil {
					stepResponse.Condition = condition
				}
			}
			response.Steps = append(response.Steps, stepResponse)
		}
	}

	return response
}
