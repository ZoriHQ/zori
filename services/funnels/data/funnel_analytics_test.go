package data

import (
	"encoding/json"
	"testing"
	"zori/internal/storage/postgres/models"
	"zori/services/funnels/types"
)

func TestFunnelAnalyticsData_BuildConditionFromStepCondition(t *testing.T) {
	d := &FunnelAnalyticsData{}

	tests := []struct {
		name      string
		condition types.StepCondition
		want      string
		wantErr   bool
	}{
		{
			name: "event condition",
			condition: types.StepCondition{
				Type:      types.ConditionTypeEvent,
				EventName: strPtr("purchase"),
			},
			want:    "event_name = 'purchase'",
			wantErr: false,
		},
		{
			name: "event condition missing event_name",
			condition: types.StepCondition{
				Type: types.ConditionTypeEvent,
			},
			wantErr: true,
		},
		{
			name: "pageview condition with exact path",
			condition: types.StepCondition{
				Type:     types.ConditionTypePageview,
				PagePath: strPtr("/checkout"),
			},
			want:    "event_name = '$pageview' AND page_path = '/checkout'",
			wantErr: false,
		},
		{
			name: "pageview condition with pattern",
			condition: types.StepCondition{
				Type:        types.ConditionTypePageview,
				PagePattern: strPtr("/product/.*"),
			},
			want:    "event_name = '$pageview' AND match(page_path, '/product/.*')",
			wantErr: false,
		},
		{
			name: "pageview condition without path",
			condition: types.StepCondition{
				Type: types.ConditionTypePageview,
			},
			want:    "event_name = '$pageview'",
			wantErr: false,
		},
		{
			name: "click condition with selector",
			condition: types.StepCondition{
				Type:          types.ConditionTypeClick,
				ClickSelector: strPtr(".add-to-cart"),
			},
			want:    "event_name = '$click' AND click_element_selector = '.add-to-cart'",
			wantErr: false,
		},
		{
			name: "click condition with text",
			condition: types.StepCondition{
				Type:      types.ConditionTypeClick,
				ClickText: strPtr("Add to Cart"),
			},
			want:    "event_name = '$click' AND click_element_text = 'Add to Cart'",
			wantErr: false,
		},
		{
			name: "click condition with category",
			condition: types.StepCondition{
				Type:          types.ConditionTypeClick,
				ClickCategory: strPtr("cta"),
			},
			want:    "event_name = '$click' AND click_element_category = 'cta'",
			wantErr: false,
		},
		{
			name: "click condition with multiple fields",
			condition: types.StepCondition{
				Type:          types.ConditionTypeClick,
				ClickSelector: strPtr(".btn"),
				ClickText:     strPtr("Buy Now"),
			},
			want:    "event_name = '$click' AND click_element_selector = '.btn' AND click_element_text = 'Buy Now'",
			wantErr: false,
		},
		{
			name: "condition with eq filter",
			condition: types.StepCondition{
				Type:      types.ConditionTypeEvent,
				EventName: strPtr("signup"),
				Filters: []types.PropertyFilter{
					{Property: "utm_source", Operator: types.FilterOperatorEq, Value: "google"},
				},
			},
			want:    "event_name = 'signup' AND utm_source = 'google'",
			wantErr: false,
		},
		{
			name: "condition with neq filter",
			condition: types.StepCondition{
				Type:      types.ConditionTypeEvent,
				EventName: strPtr("pageview"),
				Filters: []types.PropertyFilter{
					{Property: "device_type", Operator: types.FilterOperatorNeq, Value: "bot"},
				},
			},
			want:    "event_name = 'pageview' AND device_type != 'bot'",
			wantErr: false,
		},
		{
			name: "condition with contains filter",
			condition: types.StepCondition{
				Type:      types.ConditionTypeEvent,
				EventName: strPtr("click"),
				Filters: []types.PropertyFilter{
					{Property: "page_path", Operator: types.FilterOperatorContains, Value: "product"},
				},
			},
			want:    "event_name = 'click' AND position(page_path, 'product') > 0",
			wantErr: false,
		},
		{
			name: "condition with regex filter",
			condition: types.StepCondition{
				Type:      types.ConditionTypeEvent,
				EventName: strPtr("view"),
				Filters: []types.PropertyFilter{
					{Property: "page_path", Operator: types.FilterOperatorRegex, Value: "^/blog/.*"},
				},
			},
			want:    "event_name = 'view' AND match(page_path, '^/blog/.*')",
			wantErr: false,
		},
		{
			name: "condition with multiple filters",
			condition: types.StepCondition{
				Type:      types.ConditionTypeEvent,
				EventName: strPtr("purchase"),
				Filters: []types.PropertyFilter{
					{Property: "utm_source", Operator: types.FilterOperatorEq, Value: "google"},
					{Property: "device_type", Operator: types.FilterOperatorEq, Value: "mobile"},
				},
			},
			want:    "event_name = 'purchase' AND utm_source = 'google' AND device_type = 'mobile'",
			wantErr: false,
		},
		{
			name: "unknown condition type",
			condition: types.StepCondition{
				Type: "unknown",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := d.buildConditionFromStepCondition(&tt.condition)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildConditionFromStepCondition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("buildConditionFromStepCondition() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFunnelAnalyticsData_BuildStepCondition(t *testing.T) {
	d := &FunnelAnalyticsData{}

	condition := types.StepCondition{
		Type:     types.ConditionTypePageview,
		PagePath: strPtr("/checkout"),
	}
	conditionJSON, _ := json.Marshal(condition)

	step := &models.FunnelStep{
		ID:        "step-1",
		FunnelID:  "funnel-1",
		StepOrder: 1,
		Name:      "Checkout Page",
		Condition: conditionJSON,
	}

	got, err := d.buildStepCondition(step)
	if err != nil {
		t.Errorf("buildStepCondition() error = %v", err)
		return
	}

	want := "event_name = '$pageview' AND page_path = '/checkout'"
	if got != want {
		t.Errorf("buildStepCondition() = %q, want %q", got, want)
	}
}

func TestFunnelAnalyticsData_EscapeString(t *testing.T) {
	d := &FunnelAnalyticsData{}

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple string",
			input: "hello",
			want:  "hello",
		},
		{
			name:  "string with single quote",
			input: "it's working",
			want:  "it\\'s working",
		},
		{
			name:  "string with backslash",
			input: "path\\to\\file",
			want:  "path\\\\to\\\\file",
		},
		{
			name:  "string with both",
			input: "it's a path\\test",
			want:  "it\\'s a path\\\\test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := d.escapeString(tt.input)
			if got != tt.want {
				t.Errorf("escapeString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFunnelAnalyticsData_BuildFilterCondition(t *testing.T) {
	d := &FunnelAnalyticsData{}

	tests := []struct {
		name    string
		filter  types.PropertyFilter
		want    string
		wantErr bool
	}{
		{
			name:   "eq operator",
			filter: types.PropertyFilter{Property: "utm_source", Operator: types.FilterOperatorEq, Value: "google"},
			want:   "utm_source = 'google'",
		},
		{
			name:   "neq operator",
			filter: types.PropertyFilter{Property: "device_type", Operator: types.FilterOperatorNeq, Value: "bot"},
			want:   "device_type != 'bot'",
		},
		{
			name:   "contains operator",
			filter: types.PropertyFilter{Property: "page_url", Operator: types.FilterOperatorContains, Value: "checkout"},
			want:   "position(page_url, 'checkout') > 0",
		},
		{
			name:   "regex operator",
			filter: types.PropertyFilter{Property: "page_path", Operator: types.FilterOperatorRegex, Value: "^/product/.*"},
			want:   "match(page_path, '^/product/.*')",
		},
		{
			name:    "unknown operator",
			filter:  types.PropertyFilter{Property: "field", Operator: "unknown", Value: "value"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := d.buildFilterCondition(&tt.filter)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildFilterCondition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("buildFilterCondition() = %q, want %q", got, tt.want)
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}
