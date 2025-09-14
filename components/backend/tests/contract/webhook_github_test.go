package contract

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ambient-code-backend/pkg/handlers"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitHubWebhook(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		payload        map[string]interface{}
		apiKey         string
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "valid_pull_request_webhook",
			payload: map[string]interface{}{
				"action": "opened",
				"pull_request": map[string]interface{}{
					"id":    123,
					"title": "Add new feature",
					"body":  "Description of changes",
				},
				"repository": map[string]interface{}{
					"name": "my-repo",
					"owner": map[string]interface{}{
						"login": "my-org",
					},
				},
			},
			apiKey:         "test-api-key-123",
			expectedStatus: http.StatusAccepted,
			expectedBody: map[string]interface{}{
				"sessionId": "550e8400-e29b-41d4-a716-446655440000", // Will be UUID
				"namespace": "team-alpha",
				"status":    "accepted",
			},
		},
		{
			name:           "missing_api_key",
			payload:        map[string]interface{}{"action": "opened"},
			apiKey:         "",
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error":   "missing_api_key",
				"message": "X-API-Key header is required",
			},
		},
		{
			name:           "invalid_api_key",
			payload:        map[string]interface{}{"action": "opened"},
			apiKey:         "invalid-key",
			expectedStatus: http.StatusUnauthorized,
			expectedBody: map[string]interface{}{
				"error":   "invalid_api_key",
				"message": "API key is invalid or expired",
			},
		},
		{
			name:           "invalid_json_payload",
			payload:        nil, // Will send invalid JSON
			apiKey:         "test-api-key-123",
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error":   "invalid_payload",
				"message": "Request body must be valid JSON",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create router with webhook handler
			router := gin.New()
			webhookHandler := handlers.NewWebhookHandler(nil) // nil client for testing
			router.POST("/api/v1/webhooks/github", webhookHandler.HandleGitHubWebhook)

			// Prepare request body
			var reqBody []byte
			if tt.payload != nil {
				var err error
				reqBody, err = json.Marshal(tt.payload)
				require.NoError(t, err)
			} else {
				reqBody = []byte(`{"invalid": json}`) // Invalid JSON
			}

			// Create request
			req, err := http.NewRequest("POST", "/api/v1/webhooks/github", bytes.NewBuffer(reqBody))
			require.NoError(t, err)

			req.Header.Set("Content-Type", "application/json")
			if tt.apiKey != "" {
				req.Header.Set("X-API-Key", tt.apiKey)
			}

			// Record response
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Assert status code
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Assert response body structure
			var responseBody map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &responseBody)
			require.NoError(t, err)

			// Check required fields based on status
			if tt.expectedStatus == http.StatusAccepted {
				assert.Contains(t, responseBody, "sessionId")
				assert.Contains(t, responseBody, "namespace")
				assert.Contains(t, responseBody, "status")
				assert.Equal(t, tt.expectedBody["status"], responseBody["status"])
			} else {
				assert.Equal(t, tt.expectedBody["error"], responseBody["error"])
				assert.Equal(t, tt.expectedBody["message"], responseBody["message"])
			}
		})
	}
}

// Test uses actual webhook handler implementation