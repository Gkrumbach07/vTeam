package contract

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestGetSessionEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		namespace  string
		sessionId  string
		headers    map[string]string
		expectCode int
		expectKeys []string
	}{
		{
			name:      "Get existing session",
			namespace: "team-alpha",
			sessionId: "session-123",
			headers: map[string]string{
				"Authorization": "Bearer test-token",
			},
			expectCode: 200,
			expectKeys: []string{"sessionId", "status", "createdAt", "namespace", "trigger", "framework"},
		},
		{
			name:      "Get session with completed status",
			namespace: "team-alpha",
			sessionId: "session-completed",
			headers: map[string]string{
				"Authorization": "Bearer test-token",
			},
			expectCode: 200,
			expectKeys: []string{"sessionId", "status", "createdAt", "completedAt", "result"},
		},
		{
			name:      "Get session with failed status",
			namespace: "team-alpha",
			sessionId: "session-failed",
			headers: map[string]string{
				"Authorization": "Bearer test-token",
			},
			expectCode: 200,
			expectKeys: []string{"sessionId", "status", "createdAt", "error", "failedAt"},
		},
		{
			name:      "Get non-existent session",
			namespace: "team-alpha",
			sessionId: "non-existent-session",
			headers: map[string]string{
				"Authorization": "Bearer test-token",
			},
			expectCode: 404,
			expectKeys: []string{"error"},
		},
		{
			name:      "Get session without authorization",
			namespace: "team-alpha",
			sessionId: "session-123",
			headers:   map[string]string{},
			expectCode: 401,
			expectKeys: []string{"error"},
		},
		{
			name:      "Get session from non-existent namespace",
			namespace: "non-existent-namespace",
			sessionId: "session-123",
			headers: map[string]string{
				"Authorization": "Bearer test-token",
			},
			expectCode: 404,
			expectKeys: []string{"error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			req := httptest.NewRequest(http.MethodGet, "/api/v1/namespaces/"+tt.namespace+"/sessions/"+tt.sessionId, nil)

			// Set headers
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			// Create response recorder
			w := httptest.NewRecorder()

			// Create router and register handler
			router := gin.New()
			router.GET("/api/v1/namespaces/:namespace/sessions/:sessionId", func(c *gin.Context) {
				namespace := c.Param("namespace")
				sessionId := c.Param("sessionId")

				// Mock authentication check
				if c.GetHeader("Authorization") == "" {
					c.JSON(401, gin.H{"error": "Authorization required"})
					return
				}

				// Mock namespace validation
				if namespace == "non-existent-namespace" {
					c.JSON(404, gin.H{"error": "Namespace not found"})
					return
				}

				// Mock session data
				var response map[string]interface{}

				switch sessionId {
				case "session-123":
					response = map[string]interface{}{
						"sessionId":  sessionId,
						"status":     "Running",
						"createdAt":  "2024-01-01T12:00:00Z",
						"namespace":  namespace,
						"trigger": map[string]interface{}{
							"source": "github",
							"event":  "pull_request_opened",
							"payload": map[string]interface{}{
								"action": "opened",
								"pull_request": map[string]interface{}{
									"id":    123,
									"title": "Test PR",
								},
							},
						},
						"framework": map[string]interface{}{
							"type":    "claude-code",
							"version": "1.0",
						},
						"policy": map[string]interface{}{
							"modelConstraints": map[string]interface{}{
								"allowed": []string{"claude-3-sonnet"},
								"budget":  "10.00",
							},
							"toolConstraints": map[string]interface{}{
								"allowed": []string{"bash", "read"},
							},
						},
					}
				case "session-completed":
					response = map[string]interface{}{
						"sessionId":   sessionId,
						"status":      "Completed",
						"createdAt":   "2024-01-01T12:00:00Z",
						"completedAt": "2024-01-01T12:30:00Z",
						"namespace":   namespace,
						"result": map[string]interface{}{
							"summary":     "Task completed successfully",
							"artifactIds": []string{"artifact-1", "artifact-2"},
							"exitCode":    0,
						},
						"trigger": map[string]interface{}{
							"source": "manual",
							"event":  "user_request",
						},
						"framework": map[string]interface{}{
							"type": "claude-code",
						},
					}
				case "session-failed":
					response = map[string]interface{}{
						"sessionId": sessionId,
						"status":    "Failed",
						"createdAt": "2024-01-01T12:00:00Z",
						"failedAt":  "2024-01-01T12:15:00Z",
						"namespace": namespace,
						"error": map[string]interface{}{
							"message": "Session execution failed",
							"code":    "EXECUTION_ERROR",
							"details": "Unable to execute task due to policy violation",
						},
						"trigger": map[string]interface{}{
							"source": "manual",
							"event":  "user_request",
						},
						"framework": map[string]interface{}{
							"type": "claude-code",
						},
					}
				case "non-existent-session":
					c.JSON(404, gin.H{"error": "Session not found"})
					return
				default:
					c.JSON(404, gin.H{"error": "Session not found"})
					return
				}

				c.JSON(200, response)
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

			// Additional assertions based on session type
			if tt.expectCode == 200 {
				assert.Equal(t, tt.sessionId, response["sessionId"])
				assert.Equal(t, tt.namespace, response["namespace"])
				assert.NotEmpty(t, response["status"])
				assert.NotEmpty(t, response["createdAt"])

				// Status-specific assertions
				status := response["status"].(string)
				switch status {
				case "Running":
					assert.Contains(t, response, "trigger")
					assert.Contains(t, response, "framework")
				case "Completed":
					assert.Contains(t, response, "completedAt")
					assert.Contains(t, response, "result")
				case "Failed":
					assert.Contains(t, response, "failedAt")
					assert.Contains(t, response, "error")
				}
			}
		})
	}
}

func TestGetSessionRBAC(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		namespace  string
		sessionId  string
		authToken  string
		expectCode int
	}{
		{
			name:       "User can access session in authorized namespace",
			namespace:  "team-alpha",
			sessionId:  "session-123",
			authToken:  "Bearer valid-token-team-alpha",
			expectCode: 200,
		},
		{
			name:       "User cannot access session in unauthorized namespace",
			namespace:  "team-beta",
			sessionId:  "session-123",
			authToken:  "Bearer valid-token-team-alpha",
			expectCode: 403,
		},
		{
			name:       "Admin can access session in any namespace",
			namespace:  "team-beta",
			sessionId:  "session-123",
			authToken:  "Bearer admin-token",
			expectCode: 200,
		},
		{
			name:       "Invalid token",
			namespace:  "team-alpha",
			sessionId:  "session-123",
			authToken:  "Bearer invalid-token",
			expectCode: 401,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/namespaces/"+tt.namespace+"/sessions/"+tt.sessionId, nil)
			req.Header.Set("Authorization", tt.authToken)

			w := httptest.NewRecorder()

			router := gin.New()
			router.GET("/api/v1/namespaces/:namespace/sessions/:sessionId", func(c *gin.Context) {
				namespace := c.Param("namespace")
				sessionId := c.Param("sessionId")
				authHeader := c.GetHeader("Authorization")

				// Mock RBAC logic
				if authHeader == "Bearer invalid-token" {
					c.JSON(401, gin.H{"error": "Invalid token"})
					return
				}

				if authHeader == "Bearer admin-token" {
					// Admin can access any namespace
				} else if authHeader == "Bearer valid-token-team-alpha" {
					// User can only access team-alpha
					if namespace != "team-alpha" {
						c.JSON(403, gin.H{"error": "Insufficient permissions"})
						return
					}
				}

				// Mock session response for authorized requests
				c.JSON(200, map[string]interface{}{
					"sessionId": sessionId,
					"status":    "Running",
					"namespace": namespace,
					"createdAt": "2024-01-01T12:00:00Z",
					"trigger": map[string]interface{}{
						"source": "manual",
					},
					"framework": map[string]interface{}{
						"type": "claude-code",
					},
				})
			})

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectCode, w.Code)

			if tt.expectCode == 200 {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.sessionId, response["sessionId"])
				assert.Equal(t, tt.namespace, response["namespace"])
			}
		})
	}
}