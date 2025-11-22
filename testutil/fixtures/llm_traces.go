package fixtures

import (
	"context"
	"testing"
	"time"

	"zori/di"
	"zori/services/llmtraces/types"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

type LLMTraceInsertData struct {
	VisitorID          *string
	CustomerID         *string
	Provider           string
	Model              string
	InputTokens        uint32
	OutputTokens       uint32
	InputPrompt        *string
	OutputPrompt       *string
	InputCost          decimal.Decimal
	OutputCost         decimal.Decimal
	ExtraFee           *decimal.Decimal
	TotalCost          decimal.Decimal
	ProjectID          string
	OrganizationID     string
	ClientTimestampUTC time.Time
}

func InsertLLMTracesDirect(t *testing.T, tc *di.TestContainer, traces []LLMTraceInsertData) error {
	t.Helper()

	ctx := context.Background()

	for _, trace := range traces {
		// Handle nullable ExtraFee - use zero value if nil
		var extraFee interface{}
		if trace.ExtraFee != nil {
			extraFee = *trace.ExtraFee
		} else {
			extraFee = nil
		}

		query := `
			INSERT INTO llm_traces (
				trace_id, visitor_id, customer_id, provider, model,
				input_tokens, output_tokens, input_prompt, output_prompt,
				input_cost, output_cost, extra_fee, total_cost,
				project_id, organization_id, client_timestamp_utc
			) VALUES (
				generateUUIDv4(), ?, ?, ?, ?,
				?, ?, ?, ?,
				?, ?, ?, ?,
				?, ?, ?
			)
		`

		err := tc.ClickHouse.Db().Exec(ctx, query,
			trace.VisitorID, trace.CustomerID, trace.Provider, trace.Model,
			trace.InputTokens, trace.OutputTokens, trace.InputPrompt, trace.OutputPrompt,
			trace.InputCost, trace.OutputCost, extraFee, trace.TotalCost,
			trace.ProjectID, trace.OrganizationID, trace.ClientTimestampUTC,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func CreateLLMTraceRequest(provider, model string, inputTokens, outputTokens uint32, visitorID, customerID *string) *types.LLMTraceRequest {
	return &types.LLMTraceRequest{
		VisitorID:    visitorID,
		CustomerID:   customerID,
		Provider:     provider,
		Model:        model,
		InputTokens:  &inputTokens,
		OutputTokens: &outputTokens,
	}
}

func StrPtr(s string) *string {
	return &s
}

func InsertTestLLMTraces(t *testing.T, tc *di.TestContainer, projectID, orgID string, count int) {
	t.Helper()

	now := time.Now().UTC()
	traces := make([]LLMTraceInsertData, count)

	for i := 0; i < count; i++ {
		customerID := StrPtr("customer-" + string(rune('A'+i%5)))
		traces[i] = LLMTraceInsertData{
			CustomerID:         customerID,
			Provider:           "openai",
			Model:              "gpt-4o",
			InputTokens:        uint32(100 + i*10),
			OutputTokens:       uint32(200 + i*20),
			InputCost:          decimal.NewFromFloat(0.0001 * float64(i+1)),
			OutputCost:         decimal.NewFromFloat(0.0002 * float64(i+1)),
			TotalCost:          decimal.NewFromFloat(0.0003 * float64(i+1)),
			ProjectID:          projectID,
			OrganizationID:     orgID,
			ClientTimestampUTC: now.Add(-time.Duration(i) * time.Hour),
		}
	}

	err := InsertLLMTracesDirect(t, tc, traces)
	require.NoError(t, err)
}
