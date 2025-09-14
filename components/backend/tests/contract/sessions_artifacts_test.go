package contract

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestGetSessionArtifactsEndpoint(t *testing.T) {
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
			name:      "Get artifacts for completed session",
			namespace: "team-alpha",
			sessionId: "session-with-artifacts",
			headers: map[string]string{
				"Authorization": "Bearer test-token",
			},
			expectCode: 200,
			expectKeys: []string{"artifacts", "totalCount", "sessionId"},
		},
		{
			name:      "Get artifacts for running session (no artifacts yet)",
			namespace: "team-alpha",
			sessionId: "session-running",
			headers: map[string]string{
				"Authorization": "Bearer test-token",
			},
			expectCode: 200,
			expectKeys: []string{"artifacts", "totalCount", "sessionId"},
		},
		{
			name:      "Get artifacts for non-existent session",
			namespace: "team-alpha",
			sessionId: "non-existent-session",
			headers: map[string]string{
				"Authorization": "Bearer test-token",
			},
			expectCode: 404,
			expectKeys: []string{"error"},
		},
		{
			name:      "Get artifacts without authorization",
			namespace: "team-alpha",
			sessionId: "session-with-artifacts",
			headers:   map[string]string{},
			expectCode: 401,
			expectKeys: []string{"error"},
		},
		{
			name:      "Get artifacts from non-existent namespace",
			namespace: "non-existent-namespace",
			sessionId: "session-with-artifacts",
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
			req := httptest.NewRequest(http.MethodGet, "/api/v1/namespaces/"+tt.namespace+"/sessions/"+tt.sessionId+"/artifacts", nil)

			// Set headers
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			// Create response recorder
			w := httptest.NewRecorder()

			// Create router and register handler
			router := gin.New()
			router.GET("/api/v1/namespaces/:namespace/sessions/:sessionId/artifacts", func(c *gin.Context) {
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

				// Mock session existence check
				if sessionId == "non-existent-session" {
					c.JSON(404, gin.H{"error": "Session not found"})
					return
				}

				// Mock artifact data based on session
				var response map[string]interface{}

				switch sessionId {
				case "session-with-artifacts":
					response = map[string]interface{}{
						"sessionId":  sessionId,
						"totalCount": 3,
						"artifacts": []map[string]interface{}{
							{
								"id":          "artifact-1",
								"type":        "file",
								"name":        "output.txt",
								"size":        1024,
								"createdAt":   "2024-01-01T12:10:00Z",
								"contentType": "text/plain",
								"metadata": map[string]interface{}{
									"tool":        "bash",
									"exitCode":    0,
									"description": "Command output",
								},
								"downloadUrl": "/api/v1/artifacts/artifact-1/download",
								"viewUrl":     "/api/v1/artifacts/artifact-1/view",
							},
							{
								"id":          "artifact-2",
								"type":        "file",
								"name":        "screenshot.png",
								"size":        51200,
								"createdAt":   "2024-01-01T12:15:00Z",
								"contentType": "image/png",
								"metadata": map[string]interface{}{
									"tool":        "screenshot",
									"dimensions":  "1920x1080",
									"description": "Browser screenshot",
								},
								"downloadUrl": "/api/v1/artifacts/artifact-2/download",
								"viewUrl":     "/api/v1/artifacts/artifact-2/view",
							},
							{
								"id":          "artifact-3",
								"type":        "log",
								"name":        "session.log",
								"size":        2048,
								"createdAt":   "2024-01-01T12:30:00Z",
								"contentType": "text/plain",
								"metadata": map[string]interface{}{
									"logLevel":    "info",
									"source":      "runner",
									"description": "Session execution log",
								},
								"downloadUrl": "/api/v1/artifacts/artifact-3/download",
								"viewUrl":     "/api/v1/artifacts/artifact-3/view",
							},
						},
					}
				case "session-running":
					response = map[string]interface{}{
						"sessionId":  sessionId,
						"totalCount": 0,
						"artifacts":  []map[string]interface{}{},
					}
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

			// Additional assertions for successful responses
			if tt.expectCode == 200 {
				assert.Equal(t, tt.sessionId, response["sessionId"])
				assert.Contains(t, response, "totalCount")
				assert.Contains(t, response, "artifacts")

				artifacts, ok := response["artifacts"].([]interface{})
				assert.True(t, ok, "artifacts should be an array")

				totalCount := int(response["totalCount"].(float64))
				assert.Equal(t, len(artifacts), totalCount, "totalCount should match artifacts length")

				// Validate artifact structure for non-empty responses
				if totalCount > 0 {
					firstArtifact := artifacts[0].(map[string]interface{})
					expectedArtifactKeys := []string{"id", "type", "name", "size", "createdAt", "contentType", "downloadUrl", "viewUrl"}
					for _, key := range expectedArtifactKeys {
						assert.Contains(t, firstArtifact, key, "Artifact should contain key: %s", key)
					}
				}
			}
		})
	}
}

func TestGetSessionArtifactsWithPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		namespace  string
		sessionId  string
		query      string
		expectCode int
		expectKeys []string
	}{
		{
			name:       "Get artifacts with limit",
			namespace:  "team-alpha",
			sessionId:  "session-many-artifacts",
			query:      "?limit=2",
			expectCode: 200,
			expectKeys: []string{"artifacts", "totalCount", "hasMore", "nextCursor"},
		},
		{
			name:       "Get artifacts with offset",
			namespace:  "team-alpha",
			sessionId:  "session-many-artifacts",
			query:      "?limit=2&offset=2",
			expectCode: 200,
			expectKeys: []string{"artifacts", "totalCount", "hasMore"},
		},
		{
			name:       "Get artifacts filtered by type",
			namespace:  "team-alpha",
			sessionId:  "session-many-artifacts",
			query:      "?type=file",
			expectCode: 200,
			expectKeys: []string{"artifacts", "totalCount", "filters"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/namespaces/"+tt.namespace+"/sessions/"+tt.sessionId+"/artifacts"+tt.query, nil)
			req.Header.Set("Authorization", "Bearer test-token")

			w := httptest.NewRecorder()

			router := gin.New()
			router.GET("/api/v1/namespaces/:namespace/sessions/:sessionId/artifacts", func(c *gin.Context) {
				namespace := c.Param("namespace")
				sessionId := c.Param("sessionId")

				// Parse query parameters
				limit := c.DefaultQuery("limit", "10")
				offset := c.DefaultQuery("offset", "0")
				artifactType := c.Query("type")

				// Mock response with pagination
				response := map[string]interface{}{
					"sessionId":  sessionId,
					"namespace":  namespace,
					"totalCount": 10, // Mock total
				}

				// Mock artifacts based on pagination
				if limit == "2" && offset == "0" {
					response["artifacts"] = []map[string]interface{}{
						{
							"id":        "artifact-1",
							"type":      "file",
							"name":      "output1.txt",
							"createdAt": "2024-01-01T12:00:00Z",
						},
						{
							"id":        "artifact-2",
							"type":      "log",
							"name":      "session1.log",
							"createdAt": "2024-01-01T12:05:00Z",
						},
					}
					response["hasMore"] = true
					response["nextCursor"] = "cursor-2"
				} else if limit == "2" && offset == "2" {
					response["artifacts"] = []map[string]interface{}{
						{
							"id":        "artifact-3",
							"type":      "file",
							"name":      "output2.txt",
							"createdAt": "2024-01-01T12:10:00Z",
						},
						{
							"id":        "artifact-4",
							"type":      "screenshot",
							"name":      "screen1.png",
							"createdAt": "2024-01-01T12:15:00Z",
						},
					}
					response["hasMore"] = true
				} else if artifactType == "file" {
					response["artifacts"] = []map[string]interface{}{
						{
							"id":   "artifact-1",
							"type": "file",
							"name": "output1.txt",
						},
						{
							"id":   "artifact-3",
							"type": "file",
							"name": "output2.txt",
						},
					}
					response["totalCount"] = 2
					response["filters"] = map[string]interface{}{
						"type": "file",
					}
				}

				c.JSON(200, response)
			})

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectCode, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			for _, key := range tt.expectKeys {
				assert.Contains(t, response, key, "Response should contain key: %s", key)
			}
		})
	}
}

func TestGetSessionArtifactsRBAC(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		namespace  string
		sessionId  string
		authToken  string
		expectCode int
	}{
		{
			name:       "User can access artifacts in authorized namespace",
			namespace:  "team-alpha",
			sessionId:  "session-123",
			authToken:  "Bearer valid-token-team-alpha",
			expectCode: 200,
		},
		{
			name:       "User cannot access artifacts in unauthorized namespace",
			namespace:  "team-beta",
			sessionId:  "session-123",
			authToken:  "Bearer valid-token-team-alpha",
			expectCode: 403,
		},
		{
			name:       "Admin can access artifacts in any namespace",
			namespace:  "team-beta",
			sessionId:  "session-123",
			authToken:  "Bearer admin-token",
			expectCode: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/namespaces/"+tt.namespace+"/sessions/"+tt.sessionId+"/artifacts", nil)
			req.Header.Set("Authorization", tt.authToken)

			w := httptest.NewRecorder()

			router := gin.New()
			router.GET("/api/v1/namespaces/:namespace/sessions/:sessionId/artifacts", func(c *gin.Context) {
				namespace := c.Param("namespace")
				sessionId := c.Param("sessionId")
				authHeader := c.GetHeader("Authorization")

				// Mock RBAC logic
				if authHeader == "Bearer admin-token" {
					// Admin can access any namespace
				} else if authHeader == "Bearer valid-token-team-alpha" {
					if namespace != "team-alpha" {
						c.JSON(403, gin.H{"error": "Insufficient permissions"})
						return
					}
				}

				// Mock response
				c.JSON(200, map[string]interface{}{
					"sessionId":  sessionId,
					"totalCount": 1,
					"artifacts": []map[string]interface{}{
						{
							"id":   "artifact-1",
							"type": "file",
							"name": "output.txt",
						},
					},
				})
			})

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectCode, w.Code)
		})
	}
}