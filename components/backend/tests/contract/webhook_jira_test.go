package contract

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJiraWebhook(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		payload        map[string]interface{}
		apiKey         string
		expectedStatus int
	}{
		{
			name: "valid_issue_updated_webhook",
			payload: map[string]interface{}{
				"webhookEvent": "jira:issue_updated",
				"issue": map[string]interface{}{
					"key": "PROJ-123",
					"fields": map[string]interface{}{
						"summary":     "Bug fix needed",
						"description": "Details about the bug",
					},
				},
			},
			apiKey:         "test-api-key-456",
			expectedStatus: http.StatusAccepted,
		},
		{
			name: "policy_violation_budget_exceeded",
			payload: map[string]interface{}{
				"webhookEvent": "jira:issue_updated",
				"issue": map[string]interface{}{
					"key": "PROJ-999", // This will trigger budget exceeded
				},
			},
			apiKey:         "budget-exceeded-key",
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.POST("/api/v1/webhooks/jira", handleJiraWebhook)

			reqBody, err := json.Marshal(tt.payload)
			require.NoError(t, err)

			req, err := http.NewRequest("POST", "/api/v1/webhooks/jira", bytes.NewBuffer(reqBody))
			require.NoError(t, err)

			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-API-Key", tt.apiKey)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusAccepted {
				var responseBody map[string]interface{}
				err = json.Unmarshal(w.Body.Bytes(), &responseBody)
				require.NoError(t, err)
				assert.Contains(t, responseBody, "sessionId")
				assert.Contains(t, responseBody, "namespace")
			}
		})
	}
}

// This function doesn't exist yet - this test MUST FAIL
func handleJiraWebhook(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not_implemented"})
}