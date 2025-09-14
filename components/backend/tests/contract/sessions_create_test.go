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

func TestCreateSessionEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		namespace  string
		payload    interface{}
		headers    map[string]string
		expectCode int
		expectKeys []string
	}{
		{
			name:      "Create session with valid payload",
			namespace: "team-alpha",
			payload: map[string]interface{}{
				"trigger": map[string]interface{}{
					"source": "manual",
					"event":  "user_request",
					"payload": map[string]interface{}{
						"user": "test-user",
					},
				},
				"framework": map[string]interface{}{
					"type":    "claude-code",
					"version": "1.0",
					"config":  map[string]interface{}{},
				},
				"policy": map[string]interface{}{
					"modelConstraints": map[string]interface{}{
						"allowed": []string{"claude-3-sonnet"},
						"budget":  "10.00",
					},
					"toolConstraints": map[string]interface{}{
						"allowed": []string{"bash", "read"},
					},
					"approvalRequired": false,
				},
			},
			headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer test-token",
			},
			expectCode: 201,
			expectKeys: []string{"sessionId", "status", "createdAt", "namespace"},
		},
		{
			name:      "Create session from GitHub webhook",
			namespace: "team-beta",
			payload: map[string]interface{}{
				"trigger": map[string]interface{}{
					"source": "github",
					"event":  "pull_request_opened",
					"payload": map[string]interface{}{
						"action": "opened",
						"pull_request": map[string]interface{}{
							"id":    456,
							"title": "Feature implementation",
						},
						"repository": map[string]interface{}{
							"name": "test-repo",
						},
					},
				},
				"framework": map[string]interface{}{
					"type":    "claude-code",
					"version": "1.0",
				},
			},
			headers: map[string]string{
				"Content-Type": "application/json",
			},
			expectCode: 201,
			expectKeys: []string{"sessionId", "status", "createdAt", "namespace"},
		},
		{
			name:      "Create session with invalid framework",
			namespace: "team-alpha",
			payload: map[string]interface{}{
				"trigger": map[string]interface{}{
					"source": "manual",
					"event":  "user_request",
				},
				"framework": map[string]interface{}{
					"type":    "invalid-framework",
					"version": "1.0",
				},
			},
			headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer test-token",
			},
			expectCode: 400,
			expectKeys: []string{"error", "details"},
		},
		{
			name:      "Create session with missing required fields",
			namespace: "team-alpha",
			payload: map[string]interface{}{
				"framework": map[string]interface{}{
					"type": "claude-code",
				},
				// Missing trigger
			},
			headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer test-token",
			},
			expectCode: 400,
			expectKeys: []string{"error", "details"},
		},
		{
			name:      "Create session in non-existent namespace",
			namespace: "non-existent-namespace",
			payload: map[string]interface{}{
				"trigger": map[string]interface{}{
					"source": "manual",
					"event":  "user_request",
				},
				"framework": map[string]interface{}{
					"type":    "claude-code",
					"version": "1.0",
				},
			},
			headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer test-token",
			},
			expectCode: 404,
			expectKeys: []string{"error"},
		},
		{
			name:      "Create session without authorization",
			namespace: "team-alpha",
			payload: map[string]interface{}{
				"trigger": map[string]interface{}{
					"source": "manual",
					"event":  "user_request",
				},
				"framework": map[string]interface{}{
					"type": "claude-code",
				},
			},
			headers: map[string]string{
				"Content-Type": "application/json",
			},
			expectCode: 401,
			expectKeys: []string{"error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			payloadBytes, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/namespaces/"+tt.namespace+"/sessions", bytes.NewBuffer(payloadBytes))

			// Set headers
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			// Create response recorder
			w := httptest.NewRecorder()

			// Create router and register handler
			router := gin.New()
			router.POST("/api/v1/namespaces/:namespace/sessions", func(c *gin.Context) {
				namespace := c.Param("namespace")

				// Mock authentication check
				if c.GetHeader("Authorization") == "" && namespace != "team-beta" {
					c.JSON(401, gin.H{"error": "Authorization required"})
					return
				}

				// Mock namespace validation
				if namespace == "non-existent-namespace" {
					c.JSON(404, gin.H{"error": "Namespace not found"})
					return
				}

				var payload map[string]interface{}
				if err := c.ShouldBindJSON(&payload); err != nil {
					c.JSON(400, gin.H{
						"error":   "Invalid JSON payload",
						"details": err.Error(),
					})
					return
				}

				// Mock validation
				trigger, hasTrigger := payload["trigger"]
				framework, hasFramework := payload["framework"]

				if !hasTrigger {
					c.JSON(400, gin.H{
						"error":   "Missing required field",
						"details": "trigger is required",
					})
					return
				}

				if !hasFramework {
					c.JSON(400, gin.H{
						"error":   "Missing required field",
						"details": "framework is required",
					})
					return
				}

				// Validate framework type
				if frameworkMap, ok := framework.(map[string]interface{}); ok {
					if frameworkType, ok := frameworkMap["type"].(string); ok {
						if frameworkType != "claude-code" {
							c.JSON(400, gin.H{
								"error":   "Invalid framework type",
								"details": "Only claude-code framework is supported",
							})
							return
						}
					}
				}

				// Mock successful session creation
				sessionId := "session-" + generateMockId()
				c.JSON(201, gin.H{
					"sessionId":  sessionId,
					"status":     "Pending",
					"createdAt":  "2024-01-01T12:00:00Z",
					"namespace":  namespace,
					"framework":  framework,
					"trigger":    trigger,
				})
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
			if tt.expectCode == 201 {
				assert.NotEmpty(t, response["sessionId"])
				assert.Equal(t, "Pending", response["status"])
				assert.Equal(t, tt.namespace, response["namespace"])
				assert.NotEmpty(t, response["createdAt"])
			}

			if tt.expectCode >= 400 {
				assert.Contains(t, response, "error")
				assert.NotEmpty(t, response["error"])
			}
		})
	}
}

func TestCreateSessionRBAC(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		namespace  string
		authToken  string
		expectCode int
		expectMsg  string
	}{
		{
			name:       "Valid user creates session in authorized namespace",
			namespace:  "team-alpha",
			authToken:  "Bearer valid-token-team-alpha",
			expectCode: 201,
		},
		{
			name:       "User tries to create session in unauthorized namespace",
			namespace:  "team-beta",
			authToken:  "Bearer valid-token-team-alpha",
			expectCode: 403,
			expectMsg:  "Insufficient permissions",
		},
		{
			name:       "Invalid token",
			namespace:  "team-alpha",
			authToken:  "Bearer invalid-token",
			expectCode: 401,
			expectMsg:  "Invalid token",
		},
		{
			name:       "Admin can create session in any namespace",
			namespace:  "team-beta",
			authToken:  "Bearer admin-token",
			expectCode: 201,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create valid session payload
			payload := map[string]interface{}{
				"trigger": map[string]interface{}{
					"source": "manual",
					"event":  "user_request",
				},
				"framework": map[string]interface{}{
					"type":    "claude-code",
					"version": "1.0",
				},
			}

			payloadBytes, _ := json.Marshal(payload)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/namespaces/"+tt.namespace+"/sessions", bytes.NewBuffer(payloadBytes))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", tt.authToken)

			w := httptest.NewRecorder()

			router := gin.New()
			router.POST("/api/v1/namespaces/:namespace/sessions", func(c *gin.Context) {
				namespace := c.Param("namespace")
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
				} else {
					c.JSON(401, gin.H{"error": "Authorization required"})
					return
				}

				// Mock successful session creation
				sessionId := "session-" + generateMockId()
				c.JSON(201, gin.H{
					"sessionId": sessionId,
					"status":    "Pending",
					"namespace": namespace,
				})
			})

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectCode, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			if tt.expectCode >= 400 && tt.expectMsg != "" {
				assert.Contains(t, response["error"], tt.expectMsg)
			}
		})
	}
}

// Helper function to generate mock IDs
func generateMockId() string {
	return "abc123def456"
}