package types

type ConditionType string

const (
	ConditionTypeEvent    ConditionType = "event"
	ConditionTypePageview ConditionType = "pageview"
	ConditionTypeClick    ConditionType = "click"
)

type FilterOperator string

const (
	FilterOperatorEq       FilterOperator = "eq"
	FilterOperatorNeq      FilterOperator = "neq"
	FilterOperatorContains FilterOperator = "contains"
	FilterOperatorRegex    FilterOperator = "regex"
)

type PropertyFilter struct {
	Property string         `json:"property" validate:"required" example:"utm_source"`
	Operator FilterOperator `json:"operator" validate:"required,oneof=eq neq contains regex" example:"eq"`
	Value    string         `json:"value" validate:"required" example:"google"`
}

type StepCondition struct {
	Type ConditionType `json:"type" validate:"required,oneof=event pageview click" example:"pageview"`

	EventName *string `json:"event_name,omitempty" example:"purchase"`

	PagePath    *string `json:"page_path,omitempty" example:"/checkout"`
	PagePattern *string `json:"page_pattern,omitempty" example:"/product/.*"`

	ClickSelector *string `json:"click_selector,omitempty" example:".add-to-cart-btn"`
	ClickText     *string `json:"click_text,omitempty" example:"Add to Cart"`
	ClickCategory *string `json:"click_category,omitempty" example:"cta"`

	Filters []PropertyFilter `json:"filters,omitempty"`
}

type DepthConfig struct {
	EntryCondition StepCondition `json:"entry_condition" validate:"required"`
	CountEvent     string        `json:"count_event" validate:"required" example:"$pageview"`
	MaxDepth       int           `json:"max_depth" validate:"required,min=1,max=100" example:"10"`
	Scope          string        `json:"scope" validate:"required,oneof=session visitor" example:"session"`
}
