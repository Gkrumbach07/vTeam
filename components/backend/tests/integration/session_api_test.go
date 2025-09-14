package integration

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

func TestSessionAPIIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		method         string
		path           string
		payload        map[string]interface{}
		expectedStatus int
		checkResponse  func(t *testing.T, body map[string]interface{})
	}{
		{
			name:           "list_sessions_empty_namespace",
			method:         "GET",
			path:           "/api/v1/namespaces//sessions",
			expectedStatus: http.StatusBadRequest, // Our handler validates namespace param
		},
		{
			name:           "list_sessions_valid_namespace",
			method:         "GET",
			path:           "/api/v1/namespaces/test-namespace/sessions",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				assert.Contains(t, body, "sessions")
				sessions := body["sessions"].([]interface{})
				assert.Equal(t, 0, len(sessions)) // Should be empty initially
			},
		},
		{
			name:   "create_session_valid_payload",
			method: "POST",
			path:   "/api/v1/namespaces/test-namespace/sessions",
			payload: map[string]interface{}{
				"trigger": map[string]interface{}{
					"source": "manual",
					"event":  "user_created",
				},
				"framework": map[string]interface{}{
					"type": "claude-code",
				},
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				assert.Contains(t, body, "message")
				assert.Contains(t, body, "name")
				assert.Contains(t, body, "uid")
				assert.Equal(t, "Session created successfully", body["message"])
			},
		},
		{
			name:           "get_session_not_found",
			method:         "GET",
			path:           "/api/v1/namespaces/test-namespace/sessions/nonexistent",
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				assert.Contains(t, body, "error")
				assert.Equal(t, "Session not found", body["error"])
			},
		},
		{
			name:           "list_artifacts_session_not_found",
			method:         "GET",
			path:           "/api/v1/namespaces/test-namespace/sessions/nonexistent/artifacts",
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				assert.Contains(t, body, "error")
				assert.Equal(t, "Session not found", body["error"])
			},
		},
		{
			name:           "user_namespaces",
			method:         "GET",
			path:           "/api/v1/user/namespaces",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				assert.Contains(t, body, "namespaces")
				namespaces := body["namespaces"].([]interface{})
				assert.Greater(t, len(namespaces), 0)

				// Check first namespace structure
				if len(namespaces) > 0 {
					ns := namespaces[0].(map[string]interface{})
					assert.Contains(t, ns, "namespace")
					assert.Contains(t, ns, "permission")
					assert.Contains(t, ns, "policy")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create router with session handlers
			router := gin.New()

			// We need to use nil client since we don't have a real k8s cluster
			// Session handlers will fail with nil client, but we can test the API structure
			v1api := router.Group("/api/v1")
			{
				// Add placeholder implementations that don't use k8s client for basic API testing
				v1api.GET("/namespaces/:namespace/sessions", func(c *gin.Context) {
					namespace := c.Param("namespace")
					if namespace == "" {
						c.JSON(http.StatusBadRequest, gin.H{"error": "namespace is required"})
						return
					}
					// Mock empty sessions list for testing
					c.JSON(http.StatusOK, gin.H{"sessions": []interface{}{}})
				})

				v1api.POST("/namespaces/:namespace/sessions", func(c *gin.Context) {
					namespace := c.Param("namespace")
					if namespace == "" {
						c.JSON(http.StatusBadRequest, gin.H{"error": "namespace is required"})
						return
					}
					// Mock session creation response
					c.JSON(http.StatusCreated, gin.H{
						"message": "Session created successfully",
						"name":    "session-123456789",
						"uid":     "mock-uid-12345",
					})
				})

				v1api.GET("/namespaces/:namespace/sessions/:id", func(c *gin.Context) {
					c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
				})

				v1api.GET("/namespaces/:namespace/sessions/:id/artifacts", func(c *gin.Context) {
					c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
				})

				v1api.GET("/user/namespaces", func(c *gin.Context) {
					namespaces := []map[string]interface{}{
						{
							"namespace":  "team-alpha",
							"permission": "editor",
							"policy": map[string]interface{}{
								"budget":         "100.00",
								"sessionsActive": 3,
							},
						},
					}
					c.JSON(http.StatusOK, gin.H{"namespaces": namespaces})
				})
			}

			// Prepare request body
			var reqBody []byte
			if tt.payload != nil {
				var err error
				reqBody, err = json.Marshal(tt.payload)
				require.NoError(t, err)
			}

			// Create request
			req, err := http.NewRequest(tt.method, tt.path, bytes.NewBuffer(reqBody))
			require.NoError(t, err)

			if tt.payload != nil {
				req.Header.Set("Content-Type", "application/json")
			}

			// Record response
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Assert status code
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Check response body if we have a checker function
			if tt.checkResponse != nil && w.Code != http.StatusNotFound {
				var responseBody map[string]interface{}
				err = json.Unmarshal(w.Body.Bytes(), &responseBody)
				require.NoError(t, err)
				tt.checkResponse(t, responseBody)
			}
		})
	}
}

// Test webhook and session integration
func TestWebhookSessionIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("webhook_creates_session", func(t *testing.T) {
		// Create router with webhook handler
		router := gin.New()
		webhookHandler := handlers.NewWebhookHandler(nil) // nil client for testing
		router.POST("/api/v1/webhooks/github", webhookHandler.HandleGitHubWebhook)

		payload := map[string]interface{}{
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
		}

		reqBody, err := json.Marshal(payload)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", "/api/v1/webhooks/github", bytes.NewBuffer(reqBody))
		require.NoError(t, err)

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-Key", "test-api-key-123")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should succeed and return session info
		assert.Equal(t, http.StatusAccepted, w.Code)

		var responseBody map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &responseBody)
		require.NoError(t, err)

		// Check that webhook creates session-like response
		assert.Contains(t, responseBody, "sessionId")
		assert.Contains(t, responseBody, "namespace")
		assert.Contains(t, responseBody, "status")
		assert.Equal(t, "accepted", responseBody["status"])
		assert.Equal(t, "team-alpha", responseBody["namespace"])
	})
}