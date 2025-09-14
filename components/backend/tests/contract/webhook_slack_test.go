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

func TestSlackWebhook(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		payload        map[string]interface{}
		apiKey         string
		expectedStatus int
	}{
		{
			name: "valid_message_webhook",
			payload: map[string]interface{}{
				"type":    "message",
				"channel": "C1234567890",
				"user":    "U0G9QF9C6",
				"text":    "Please analyze the website https://example.com",
				"ts":      "1355517523.000005",
			},
			apiKey:         "test-slack-key-789",
			expectedStatus: http.StatusAccepted,
		},
		{
			name: "rate_limit_exceeded",
			payload: map[string]interface{}{
				"type": "message",
				"text": "Another request",
			},
			apiKey:         "rate-limited-key",
			expectedStatus: http.StatusTooManyRequests,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.POST("/api/v1/webhooks/slack", handleSlackWebhook)

			reqBody, err := json.Marshal(tt.payload)
			require.NoError(t, err)

			req, err := http.NewRequest("POST", "/api/v1/webhooks/slack", bytes.NewBuffer(reqBody))
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
			}
		})
	}
}

// This function doesn't exist yet - this test MUST FAIL
func handleSlackWebhook(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not_implemented"})
}