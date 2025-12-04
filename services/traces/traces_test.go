package traces_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"zori/di"
	"zori/internal/utils"
	"zori/services/projects/helpers"
	"zori/services/traces/types"
	"zori/testutil/fixtures"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTracesIngestion_BasicAuth(t *testing.T) {
	tc := di.NewTestContainer(t)
	defer tc.Cleanup()

	account, project := fixtures.SetupAccountAndProject(t, tc)
	_ = account

	langfuseKeys, err := helpers.GenerateLangfuseKeys()
	require.NoError(t, err)

	ctx := context.Background()
	_, err = tc.DB.DB.NewUpdate().
		Table("projects").
		Set("langfuse_public_key = ?", langfuseKeys.PublicKey).
		Set("langfuse_secret_key = ?", langfuseKeys.SecretKey).
		Where("id = ?", project.ID).
		Exec(ctx)
	require.NoError(t, err)

	t.Run("missing Authorization header", func(t *testing.T) {
		req, err := http.NewRequest("POST", tc.IngestionServerURL+"/traces/ingest/langfuse", nil)
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer utils.EnsureBodyClosed(resp)

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("invalid Authorization format", func(t *testing.T) {
		req, err := http.NewRequest("POST", tc.IngestionServerURL+"/traces/ingest/langfuse", nil)
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer sometoken")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer utils.EnsureBodyClosed(resp)

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("invalid credentials", func(t *testing.T) {
		invalidAuth := base64.StdEncoding.EncodeToString([]byte("invalid_public:invalid_secret"))
		req, err := http.NewRequest("POST", tc.IngestionServerURL+"/traces/ingest/langfuse", nil)
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Basic "+invalidAuth)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer utils.EnsureBodyClosed(resp)

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("valid credentials", func(t *testing.T) {
		validAuth := base64.StdEncoding.EncodeToString([]byte(langfuseKeys.PublicKey + ":" + langfuseKeys.SecretKey))

		body := types.IngestionRequest{
			Batch: []types.IngestionEvent{},
		}
		bodyBytes, _ := json.Marshal(body)

		req, err := http.NewRequest("POST", tc.IngestionServerURL+"/traces/ingest/langfuse", bytes.NewReader(bodyBytes))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Basic "+validAuth)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer utils.EnsureBodyClosed(resp)

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestTracesIngestion_TraceCreate(t *testing.T) {
	tc := di.NewTestContainer(t)
	defer tc.Cleanup()

	account, project := fixtures.SetupAccountAndProject(t, tc)
	_ = account

	langfuseKeys, err := helpers.GenerateLangfuseKeys()
	require.NoError(t, err)

	ctx := context.Background()
	_, err = tc.DB.DB.NewUpdate().
		Table("projects").
		Set("langfuse_public_key = ?", langfuseKeys.PublicKey).
		Set("langfuse_secret_key = ?", langfuseKeys.SecretKey).
		Where("id = ?", project.ID).
		Exec(ctx)
	require.NoError(t, err)

	validAuth := base64.StdEncoding.EncodeToString([]byte(langfuseKeys.PublicKey + ":" + langfuseKeys.SecretKey))

	t.Run("create single trace", func(t *testing.T) {
		traceID := fmt.Sprintf("trace-%d", time.Now().UnixNano())
		traceName := "test-trace"
		timestamp := time.Now().Format(time.RFC3339Nano)

		traceBody := types.TraceBody{
			ID:        traceID,
			Timestamp: &timestamp,
			Name:      &traceName,
			Input:     json.RawMessage(`{"prompt": "Hello, world!"}`),
			Output:    json.RawMessage(`{"response": "Hi there!"}`),
			Metadata:  map[string]any{"source": "test"},
			Tags:      []string{"test", "integration"},
		}
		traceBodyBytes, _ := json.Marshal(traceBody)

		body := types.IngestionRequest{
			Batch: []types.IngestionEvent{
				{
					ID:        "event-1",
					Timestamp: timestamp,
					Type:      "trace-create",
					Body:      traceBodyBytes,
				},
			},
		}
		bodyBytes, _ := json.Marshal(body)

		req, err := http.NewRequest("POST", tc.IngestionServerURL+"/traces/ingest/langfuse", bytes.NewReader(bodyBytes))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Basic "+validAuth)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer utils.EnsureBodyClosed(resp)

		assert.Equal(t, http.StatusMultiStatus, resp.StatusCode)

		var ingestionResp types.IngestionResponse
		err = json.NewDecoder(resp.Body).Decode(&ingestionResp)
		require.NoError(t, err)

		assert.Len(t, ingestionResp.Successes, 1)
		assert.Len(t, ingestionResp.Errors, 0)
		assert.Equal(t, "event-1", ingestionResp.Successes[0].ID)
		assert.Equal(t, 201, ingestionResp.Successes[0].Status)

		time.Sleep(100 * time.Millisecond)

		var count uint64
		err = tc.ClickHouse.Db().QueryRow(ctx,
			"SELECT count() FROM llm_traces WHERE project_id = ? AND trace_id = ?",
			project.ID, traceID).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, uint64(1), count)
	})

	t.Run("create trace with missing ID", func(t *testing.T) {
		traceBody := types.TraceBody{
			ID: "",
		}
		traceBodyBytes, _ := json.Marshal(traceBody)

		body := types.IngestionRequest{
			Batch: []types.IngestionEvent{
				{
					ID:        "event-missing-id",
					Timestamp: time.Now().Format(time.RFC3339Nano),
					Type:      "trace-create",
					Body:      traceBodyBytes,
				},
			},
		}
		bodyBytes, _ := json.Marshal(body)

		req, err := http.NewRequest("POST", tc.IngestionServerURL+"/traces/ingest/langfuse", bytes.NewReader(bodyBytes))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Basic "+validAuth)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer utils.EnsureBodyClosed(resp)

		if !assert.Equal(t, http.StatusMultiStatus, resp.StatusCode) {
			t.Logf("Unexpected response code: %d", resp.StatusCode)
			t.Logf("Auth header used: Basic %s", validAuth[:20]+"...")
			return
		}

		var ingestionResp types.IngestionResponse
		err = json.NewDecoder(resp.Body).Decode(&ingestionResp)
		require.NoError(t, err)

		assert.Len(t, ingestionResp.Successes, 0)
		if assert.Len(t, ingestionResp.Errors, 1) {
			assert.Equal(t, "event-missing-id", ingestionResp.Errors[0].ID)
			assert.Equal(t, 400, ingestionResp.Errors[0].Status)
		}
	})
}

func TestTracesIngestion_GenerationCreate(t *testing.T) {
	tc := di.NewTestContainer(t)
	defer tc.Cleanup()

	account, project := fixtures.SetupAccountAndProject(t, tc)
	_ = account

	langfuseKeys, err := helpers.GenerateLangfuseKeys()
	require.NoError(t, err)

	ctx := context.Background()
	_, err = tc.DB.DB.NewUpdate().
		Table("projects").
		Set("langfuse_public_key = ?", langfuseKeys.PublicKey).
		Set("langfuse_secret_key = ?", langfuseKeys.SecretKey).
		Where("id = ?", project.ID).
		Exec(ctx)
	require.NoError(t, err)

	validAuth := base64.StdEncoding.EncodeToString([]byte(langfuseKeys.PublicKey + ":" + langfuseKeys.SecretKey))

	t.Run("create generation with usage", func(t *testing.T) {
		traceID := fmt.Sprintf("trace-%d", time.Now().UnixNano())
		generationID := fmt.Sprintf("gen-%d", time.Now().UnixNano())
		startTime := time.Now()
		endTime := startTime.Add(500 * time.Millisecond)
		startTimeStr := startTime.Format(time.RFC3339Nano)
		endTimeStr := endTime.Format(time.RFC3339Nano)

		modelName := "gpt-4"
		genName := "chat-completion"
		inputTokens := 100
		outputTokens := 50
		totalTokens := 150
		inputCost := 0.003
		outputCost := 0.006
		totalCost := 0.009

		genBody := types.GenerationBody{
			ID:        generationID,
			TraceID:   traceID,
			Name:      &genName,
			StartTime: startTimeStr,
			EndTime:   &endTimeStr,
			Model:     &modelName,
			Input:     json.RawMessage(`[{"role": "user", "content": "Hello"}]`),
			Output:    json.RawMessage(`{"role": "assistant", "content": "Hi!"}`),
			Usage: &types.UsageObject{
				Input:      &inputTokens,
				Output:     &outputTokens,
				Total:      &totalTokens,
				InputCost:  &inputCost,
				OutputCost: &outputCost,
				TotalCost:  &totalCost,
			},
			Metadata: map[string]any{"model_provider": "openai"},
		}
		genBodyBytes, _ := json.Marshal(genBody)

		body := types.IngestionRequest{
			Batch: []types.IngestionEvent{
				{
					ID:        "event-gen-1",
					Timestamp: startTimeStr,
					Type:      "generation-create",
					Body:      genBodyBytes,
				},
			},
		}
		bodyBytes, _ := json.Marshal(body)

		req, err := http.NewRequest("POST", tc.IngestionServerURL+"/traces/ingest/langfuse", bytes.NewReader(bodyBytes))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Basic "+validAuth)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer utils.EnsureBodyClosed(resp)

		assert.Equal(t, http.StatusMultiStatus, resp.StatusCode)

		var ingestionResp types.IngestionResponse
		err = json.NewDecoder(resp.Body).Decode(&ingestionResp)
		require.NoError(t, err)

		assert.Len(t, ingestionResp.Successes, 1)
		assert.Len(t, ingestionResp.Errors, 0)
		assert.Equal(t, "event-gen-1", ingestionResp.Successes[0].ID)

		time.Sleep(100 * time.Millisecond)

		var count uint64
		err = tc.ClickHouse.Db().QueryRow(ctx,
			"SELECT count() FROM llm_generations WHERE project_id = ? AND generation_id = ?",
			project.ID, generationID).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, uint64(1), count)

		var model string
		var inputToks, outputToks uint32
		var cost float64
		err = tc.ClickHouse.Db().QueryRow(ctx,
			`SELECT model, input_tokens, output_tokens, total_cost
			 FROM llm_generations
			 WHERE project_id = ? AND generation_id = ?`,
			project.ID, generationID).Scan(&model, &inputToks, &outputToks, &cost)
		require.NoError(t, err)
		assert.Equal(t, "gpt-4", model)
		assert.Equal(t, uint32(100), inputToks)
		assert.Equal(t, uint32(50), outputToks)
		assert.InDelta(t, 0.009, cost, 0.0001)
	})

	t.Run("create generation missing traceId", func(t *testing.T) {
		genBody := types.GenerationBody{
			ID:        "gen-no-trace",
			TraceID:   "",
			StartTime: time.Now().Format(time.RFC3339Nano),
		}
		genBodyBytes, _ := json.Marshal(genBody)

		body := types.IngestionRequest{
			Batch: []types.IngestionEvent{
				{
					ID:        "event-no-trace",
					Timestamp: time.Now().Format(time.RFC3339Nano),
					Type:      "generation-create",
					Body:      genBodyBytes,
				},
			},
		}
		bodyBytes, _ := json.Marshal(body)

		req, err := http.NewRequest("POST", tc.IngestionServerURL+"/traces/ingest/langfuse", bytes.NewReader(bodyBytes))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Basic "+validAuth)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer utils.EnsureBodyClosed(resp)

		assert.Equal(t, http.StatusMultiStatus, resp.StatusCode)

		var ingestionResp types.IngestionResponse
		err = json.NewDecoder(resp.Body).Decode(&ingestionResp)
		require.NoError(t, err)

		assert.Len(t, ingestionResp.Errors, 1)
		assert.Equal(t, "event-no-trace", ingestionResp.Errors[0].ID)
	})
}

func TestTracesIngestion_BatchEvents(t *testing.T) {
	tc := di.NewTestContainer(t)
	defer tc.Cleanup()

	account, project := fixtures.SetupAccountAndProject(t, tc)
	_ = account

	langfuseKeys, err := helpers.GenerateLangfuseKeys()
	require.NoError(t, err)

	ctx := context.Background()
	_, err = tc.DB.DB.NewUpdate().
		Table("projects").
		Set("langfuse_public_key = ?", langfuseKeys.PublicKey).
		Set("langfuse_secret_key = ?", langfuseKeys.SecretKey).
		Where("id = ?", project.ID).
		Exec(ctx)
	require.NoError(t, err)

	validAuth := base64.StdEncoding.EncodeToString([]byte(langfuseKeys.PublicKey + ":" + langfuseKeys.SecretKey))

	t.Run("batch with multiple traces and generations", func(t *testing.T) {
		timestamp := time.Now()
		timestampStr := timestamp.Format(time.RFC3339Nano)
		endTimeStr := timestamp.Add(200 * time.Millisecond).Format(time.RFC3339Nano)

		traceID1 := fmt.Sprintf("trace-batch-1-%d", time.Now().UnixNano())
		traceID2 := fmt.Sprintf("trace-batch-2-%d", time.Now().UnixNano())
		genID1 := fmt.Sprintf("gen-batch-1-%d", time.Now().UnixNano())

		traceName1 := "batch-trace-1"
		traceName2 := "batch-trace-2"
		genName := "batch-gen"
		modelName := "claude-3"

		trace1 := types.TraceBody{ID: traceID1, Name: &traceName1}
		trace2 := types.TraceBody{ID: traceID2, Name: &traceName2}
		gen1 := types.GenerationBody{
			ID:        genID1,
			TraceID:   traceID1,
			Name:      &genName,
			StartTime: timestampStr,
			EndTime:   &endTimeStr,
			Model:     &modelName,
		}

		trace1Bytes, _ := json.Marshal(trace1)
		trace2Bytes, _ := json.Marshal(trace2)
		gen1Bytes, _ := json.Marshal(gen1)

		body := types.IngestionRequest{
			Batch: []types.IngestionEvent{
				{ID: "batch-event-1", Timestamp: timestampStr, Type: "trace-create", Body: trace1Bytes},
				{ID: "batch-event-2", Timestamp: timestampStr, Type: "trace-create", Body: trace2Bytes},
				{ID: "batch-event-3", Timestamp: timestampStr, Type: "generation-create", Body: gen1Bytes},
			},
		}
		bodyBytes, _ := json.Marshal(body)

		req, err := http.NewRequest("POST", tc.IngestionServerURL+"/traces/ingest/langfuse", bytes.NewReader(bodyBytes))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Basic "+validAuth)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer utils.EnsureBodyClosed(resp)

		assert.Equal(t, http.StatusMultiStatus, resp.StatusCode)

		var ingestionResp types.IngestionResponse
		err = json.NewDecoder(resp.Body).Decode(&ingestionResp)
		require.NoError(t, err)

		assert.Len(t, ingestionResp.Successes, 3)
		assert.Len(t, ingestionResp.Errors, 0)

		time.Sleep(100 * time.Millisecond)

		var traceCount uint64
		err = tc.ClickHouse.Db().QueryRow(ctx,
			"SELECT count() FROM llm_traces WHERE project_id = ? AND (trace_id = ? OR trace_id = ?)",
			project.ID, traceID1, traceID2).Scan(&traceCount)
		require.NoError(t, err)
		assert.Equal(t, uint64(2), traceCount)

		var genCount uint64
		err = tc.ClickHouse.Db().QueryRow(ctx,
			"SELECT count() FROM llm_generations WHERE project_id = ? AND generation_id = ?",
			project.ID, genID1).Scan(&genCount)
		require.NoError(t, err)
		assert.Equal(t, uint64(1), genCount)
	})

	t.Run("batch with mixed success and errors", func(t *testing.T) {
		timestampStr := time.Now().Format(time.RFC3339Nano)

		traceID := fmt.Sprintf("trace-mixed-%d", time.Now().UnixNano())
		traceName := "valid-trace"

		validTrace := types.TraceBody{ID: traceID, Name: &traceName}
		invalidTrace := types.TraceBody{ID: ""}

		validBytes, _ := json.Marshal(validTrace)
		invalidBytes, _ := json.Marshal(invalidTrace)

		body := types.IngestionRequest{
			Batch: []types.IngestionEvent{
				{ID: "mixed-event-1", Timestamp: timestampStr, Type: "trace-create", Body: validBytes},
				{ID: "mixed-event-2", Timestamp: timestampStr, Type: "trace-create", Body: invalidBytes},
			},
		}
		bodyBytes, _ := json.Marshal(body)

		req, err := http.NewRequest("POST", tc.IngestionServerURL+"/traces/ingest/langfuse", bytes.NewReader(bodyBytes))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Basic "+validAuth)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer utils.EnsureBodyClosed(resp)

		assert.Equal(t, http.StatusMultiStatus, resp.StatusCode)

		var ingestionResp types.IngestionResponse
		err = json.NewDecoder(resp.Body).Decode(&ingestionResp)
		require.NoError(t, err)

		assert.Len(t, ingestionResp.Successes, 1)
		assert.Len(t, ingestionResp.Errors, 1)
		assert.Equal(t, "mixed-event-1", ingestionResp.Successes[0].ID)
		assert.Equal(t, "mixed-event-2", ingestionResp.Errors[0].ID)
	})
}

func TestTracesIngestion_UnknownEventTypes(t *testing.T) {
	tc := di.NewTestContainer(t)
	defer tc.Cleanup()

	account, project := fixtures.SetupAccountAndProject(t, tc)
	_ = account

	langfuseKeys, err := helpers.GenerateLangfuseKeys()
	require.NoError(t, err)

	ctx := context.Background()
	_, err = tc.DB.DB.NewUpdate().
		Table("projects").
		Set("langfuse_public_key = ?", langfuseKeys.PublicKey).
		Set("langfuse_secret_key = ?", langfuseKeys.SecretKey).
		Where("id = ?", project.ID).
		Exec(ctx)
	require.NoError(t, err)

	validAuth := base64.StdEncoding.EncodeToString([]byte(langfuseKeys.PublicKey + ":" + langfuseKeys.SecretKey))

	t.Run("span-create is accepted (no-op)", func(t *testing.T) {
		body := types.IngestionRequest{
			Batch: []types.IngestionEvent{
				{
					ID:        "span-event-1",
					Timestamp: time.Now().Format(time.RFC3339Nano),
					Type:      "span-create",
					Body:      json.RawMessage(`{}`),
				},
			},
		}
		bodyBytes, _ := json.Marshal(body)

		req, err := http.NewRequest("POST", tc.IngestionServerURL+"/traces/ingest/langfuse", bytes.NewReader(bodyBytes))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Basic "+validAuth)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer utils.EnsureBodyClosed(resp)

		assert.Equal(t, http.StatusMultiStatus, resp.StatusCode)

		var ingestionResp types.IngestionResponse
		err = json.NewDecoder(resp.Body).Decode(&ingestionResp)
		require.NoError(t, err)

		assert.Len(t, ingestionResp.Successes, 1)
		assert.Len(t, ingestionResp.Errors, 0)
	})

	t.Run("completely unknown type returns error", func(t *testing.T) {
		body := types.IngestionRequest{
			Batch: []types.IngestionEvent{
				{
					ID:        "unknown-event-1",
					Timestamp: time.Now().Format(time.RFC3339Nano),
					Type:      "unknown-type",
					Body:      json.RawMessage(`{}`),
				},
			},
		}
		bodyBytes, _ := json.Marshal(body)

		req, err := http.NewRequest("POST", tc.IngestionServerURL+"/traces/ingest/langfuse", bytes.NewReader(bodyBytes))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Basic "+validAuth)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer utils.EnsureBodyClosed(resp)

		assert.Equal(t, http.StatusMultiStatus, resp.StatusCode)

		var ingestionResp types.IngestionResponse
		err = json.NewDecoder(resp.Body).Decode(&ingestionResp)
		require.NoError(t, err)

		assert.Len(t, ingestionResp.Successes, 0)
		assert.Len(t, ingestionResp.Errors, 1)
		assert.Contains(t, ingestionResp.Errors[0].Message, "Unknown event type")
	})
}
