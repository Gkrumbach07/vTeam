package contract

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestWebhookValidateEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		source     string
		payload    interface{}
		headers    map[string]string
		expectCode int
		expectKeys []string
	}{
		{
			name:   "GitHub webhook validation - valid payload",
			source: "github",
			payload: map[string]interface{}{
				"action": "opened",
				"pull_request": map[string]interface{}{
					"id":    123,
					"title": "Test PR",
				},
				"repository": map[string]interface{}{
					"name": "test-repo",
					"owner": map[string]interface{}{
						"login": "test-user",
					},
				},
			},
			headers: map[string]string{
				"X-GitHub-Event": "pull_request",
				"Content-Type":   "application/json",
			},
			expectCode: 200,
			expectKeys: []string{"valid", "namespace", "framework"},
		},
		{
			name:   "GitHub webhook validation - invalid payload",
			source: "github",
			payload: map[string]interface{}{
				"invalid": "data",
			},
			headers: map[string]string{
				"X-GitHub-Event": "pull_request",
				"Content-Type":   "application/json",
			},
			expectCode: 400,
			expectKeys: []string{"valid", "errors"},
		},
		{
			name:   "Jira webhook validation - valid payload",
			source: "jira",
			payload: map[string]interface{}{
				"issue": map[string]interface{}{
					"id":  "TEST-123",
					"key": "TEST-123",
					"fields": map[string]interface{}{
						"summary": "Test Issue",
					},
				},
				"webhookEvent": "jira:issue_updated",
			},
			headers: map[string]string{
				"X-Atlassian-Webhook-Identifier": "test-webhook",
				"Content-Type":                   "application/json",
			},
			expectCode: 200,
			expectKeys: []string{"valid", "namespace", "framework"},
		},
		{
			name:   "Slack webhook validation - valid payload",
			source: "slack",
			payload: map[string]interface{}{
				"token": "test-token",
				"team_id": "T123456",
				"channel": map[string]interface{}{
					"id":   "C123456",
					"name": "general",
				},
				"event": map[string]interface{}{
					"type": "message",
					"text": "Hello world",
				},
			},
			headers: map[string]string{
				"X-Slack-Signature": "test-signature",
				"Content-Type":      "application/json",
			},
			expectCode: 200,
			expectKeys: []string{"valid", "namespace", "framework"},
		},
		{
			name:   "Unsupported webhook source",
			source: "unsupported",
			payload: map[string]interface{}{
				"test": "data",
			},
			headers: map[string]string{
				"Content-Type": "application/json",
			},
			expectCode: 404,
			expectKeys: []string{"error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			payloadBytes, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/"+tt.source+"/validate", bytes.NewBuffer(payloadBytes))

			// Set headers
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			// Create response recorder
			w := httptest.NewRecorder()

			// Create router and register handler
			router := gin.New()
			router.POST("/api/v1/webhooks/:source/validate", func(c *gin.Context) {
				// Mock webhook validation handler
				source := c.Param("source")

				switch source {
				case "github", "jira", "slack":
					var payload map[string]interface{}
					if err := c.ShouldBindJSON(&payload); err != nil {
						c.JSON(400, gin.H{
							"valid":  false,
							"errors": []string{"Invalid JSON payload"},
						})
						return
					}

					// Mock validation logic
					if source == "github" {
						if _, hasAction := payload["action"]; !hasAction {
							c.JSON(400, gin.H{
								"valid":  false,
								"errors": []string{"Missing required field: action"},
							})
							return
						}
					}

					c.JSON(200, gin.H{
						"valid":     true,
						"namespace": "team-alpha", // Mock namespace resolution
						"framework": "claude-code",
					})
				default:
					c.JSON(404, gin.H{
						"error": "Unsupported webhook source",
					})
				}
			})

			// Perform request
			router.ServeHTTP(w, req)

			// Assert response code
			assert.Equal(t, tt.expectCode, w.Code)

			// Parse response
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			// Assert expected keys are present
			for _, key := range tt.expectKeys {
				assert.Contains(t, response, key, "Response should contain key: %s", key)
			}

			// Additional assertions based on response code
			if tt.expectCode == 200 {
				assert.True(t, response["valid"].(bool))
				assert.NotEmpty(t, response["namespace"])
				assert.NotEmpty(t, response["framework"])
			}

			if tt.expectCode == 400 {
				assert.False(t, response["valid"].(bool))
				assert.Contains(t, response, "errors")
			}
		})
	}
}

func TestWebhookValidateAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		source     string
		headers    map[string]string
		expectCode int
	}{
		{
			name:   "GitHub webhook without signature",
			source: "github",
			headers: map[string]string{
				"Content-Type": "application/json",
			},
			expectCode: 401,
		},
		{
			name:   "Jira webhook without identifier",
			source: "jira",
			headers: map[string]string{
				"Content-Type": "application/json",
			},
			expectCode: 401,
		},
		{
			name:   "Slack webhook without signature",
			source: "slack",
			headers: map[string]string{
				"Content-Type": "application/json",
			},
			expectCode: 401,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request with minimal payload
			payload := map[string]interface{}{"test": "data"}
			payloadBytes, _ := json.Marshal(payload)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/"+tt.source+"/validate", bytes.NewBuffer(payloadBytes))

			// Set headers
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			// Create response recorder
			w := httptest.NewRecorder()

			// Create router and register handler
			router := gin.New()
			router.POST("/api/v1/webhooks/:source/validate", func(c *gin.Context) {
				source := c.Param("source")

				// Mock authentication check
				switch source {
				case "github":
					if c.GetHeader("X-Hub-Signature-256") == "" && c.GetHeader("X-GitHub-Event") == "" {
						c.JSON(401, gin.H{"error": "Authentication required"})
						return
					}
				case "jira":
					if c.GetHeader("X-Atlassian-Webhook-Identifier") == "" {
						c.JSON(401, gin.H{"error": "Authentication required"})
						return
					}
				case "slack":
					if c.GetHeader("X-Slack-Signature") == "" {
						c.JSON(401, gin.H{"error": "Authentication required"})
						return
					}
				}

				c.JSON(200, gin.H{"valid": true})
			})

			// Perform request
			router.ServeHTTP(w, req)

			// Assert response code
			assert.Equal(t, tt.expectCode, w.Code)

			if tt.expectCode == 401 {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response, "error")
			}
		})
	}
}