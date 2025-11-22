package web_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"zori/di"
	"zori/testutil/fixtures"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIngestLLMTrace(t *testing.T) {
	tc := di.NewTestContainer(t)
	defer tc.Cleanup()

	_, project := fixtures.SetupAccountAndProject(t, tc)

	time.Sleep(500 * time.Millisecond)

	t.Run("should ingest LLM trace with token counts", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"visitor_id":    "visitor-123",
			"customer_id":   "customer-456",
			"provider":      "openai",
			"model":         "gpt-4o",
			"input_tokens":  100,
			"output_tokens": 200,
		}

		body, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", tc.IngestionServerURL+"/llm-trace", bytes.NewReader(body))
		require.NoError(t, err)

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-zori-pt", project.ProjectToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		assert.Contains(t, string(respBody), "ACCEPTED")
	})

	t.Run("should ingest LLM trace with prompts and calculate tokens", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"customer_id":   "customer-789",
			"provider":      "openai",
			"model":         "gpt-4o",
			"input_prompt":  "Hello, how are you today?",
			"output_prompt": "I'm doing well, thank you for asking!",
		}

		body, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", tc.IngestionServerURL+"/llm-trace", bytes.NewReader(body))
		require.NoError(t, err)

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-zori-pt", project.ProjectToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("should reject request without model", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"provider":     "openai",
			"input_tokens": 100,
		}

		body, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", tc.IngestionServerURL+"/llm-trace", bytes.NewReader(body))
		require.NoError(t, err)

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-zori-pt", project.ProjectToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("should reject request without authentication token", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"provider":      "openai",
			"model":         "gpt-4o",
			"input_tokens":  100,
			"output_tokens": 200,
		}

		body, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", tc.IngestionServerURL+"/llm-trace", bytes.NewReader(body))
		require.NoError(t, err)

		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}
