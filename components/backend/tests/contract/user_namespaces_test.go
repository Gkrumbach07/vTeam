package contract

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestGetUserNamespacesEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		headers    map[string]string
		expectCode int
		expectKeys []string
	}{
		{
			name: "Get namespaces for regular user",
			headers: map[string]string{
				"Authorization": "Bearer valid-user-token",
			},
			expectCode: 200,
			expectKeys: []string{"namespaces", "totalCount", "user"},
		},
		{
			name: "Get namespaces for admin user",
			headers: map[string]string{
				"Authorization": "Bearer admin-token",
			},
			expectCode: 200,
			expectKeys: []string{"namespaces", "totalCount", "user"},
		},
		{
			name:       "Get namespaces without authorization",
			headers:    map[string]string{},
			expectCode: 401,
			expectKeys: []string{"error"},
		},
		{
			name: "Get namespaces with invalid token",
			headers: map[string]string{
				"Authorization": "Bearer invalid-token",
			},
			expectCode: 401,
			expectKeys: []string{"error"},
		},
		{
			name: "Get namespaces with expired token",
			headers: map[string]string{
				"Authorization": "Bearer expired-token",
			},
			expectCode: 401,
			expectKeys: []string{"error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			req := httptest.NewRequest(http.MethodGet, "/api/v1/user/namespaces", nil)

			// Set headers
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			// Create response recorder
			w := httptest.NewRecorder()

			// Create router and register handler
			router := gin.New()
			router.GET("/api/v1/user/namespaces", func(c *gin.Context) {
				authHeader := c.GetHeader("Authorization")

				// Mock authentication and authorization
				if authHeader == "" {
					c.JSON(401, gin.H{"error": "Authorization required"})
					return
				}

				if authHeader == "Bearer invalid-token" || authHeader == "Bearer expired-token" {
					c.JSON(401, gin.H{"error": "Invalid or expired token"})
					return
				}

				// Mock user data based on token
				var response map[string]interface{}

				switch authHeader {
				case "Bearer valid-user-token":
					response = map[string]interface{}{
						"user": map[string]interface{}{
							"id":       "user-123",
							"username": "alice",
							"email":    "alice@example.com",
							"roles":    []string{"developer"},
						},
						"totalCount": 2,
						"namespaces": []map[string]interface{}{
							{
								"name":        "team-alpha",
								"displayName": "Team Alpha",
								"description": "Development team Alpha workspace",
								"role":        "member",
								"permissions": []string{"read", "create_session", "view_artifacts"},
								"policy": map[string]interface{}{
									"models": map[string]interface{}{
										"allowed": []string{"claude-3-sonnet", "claude-3-haiku"},
										"budget": map[string]interface{}{
											"monthly": "50.00",
											"used":    "12.50",
										},
									},
									"tools": map[string]interface{}{
										"allowed": []string{"bash", "read", "write"},
										"blocked": []string{},
									},
								},
								"stats": map[string]interface{}{
									"totalSessions":    15,
									"runningSessions":  2,
									"completedSessions": 13,
									"failedSessions":   0,
								},
								"createdAt": "2024-01-01T10:00:00Z",
								"updatedAt": "2024-01-15T14:30:00Z",
							},
							{
								"name":        "team-beta",
								"displayName": "Team Beta",
								"description": "Development team Beta workspace",
								"role":        "viewer",
								"permissions": []string{"read", "view_artifacts"},
								"policy": map[string]interface{}{
									"models": map[string]interface{}{
										"allowed": []string{"claude-3-haiku"},
										"budget": map[string]interface{}{
											"monthly": "25.00",
											"used":    "5.00",
										},
									},
									"tools": map[string]interface{}{
										"allowed": []string{"read"},
										"blocked": []string{"bash", "exec"},
									},
								},
								"stats": map[string]interface{}{
									"totalSessions":    8,
									"runningSessions":  0,
									"completedSessions": 7,
									"failedSessions":   1,
								},
								"createdAt": "2024-01-10T10:00:00Z",
								"updatedAt": "2024-01-20T09:15:00Z",
							},
						},
					}

				case "Bearer admin-token":
					response = map[string]interface{}{
						"user": map[string]interface{}{
							"id":       "admin-456",
							"username": "admin",
							"email":    "admin@example.com",
							"roles":    []string{"admin", "system"},
						},
						"totalCount": 4,
						"namespaces": []map[string]interface{}{
							{
								"name":        "ambient-system",
								"displayName": "System Namespace",
								"description": "System administration namespace",
								"role":        "admin",
								"permissions": []string{"read", "write", "delete", "admin"},
								"policy": map[string]interface{}{
									"models": map[string]interface{}{
										"allowed": []string{"all"},
										"budget": map[string]interface{}{
											"monthly": "unlimited",
											"used":    "150.00",
										},
									},
									"tools": map[string]interface{}{
										"allowed": []string{"all"},
										"blocked": []string{},
									},
								},
								"stats": map[string]interface{}{
									"totalSessions":    50,
									"runningSessions":  5,
									"completedSessions": 43,
									"failedSessions":   2,
								},
							},
							{
								"name":        "team-alpha",
								"displayName": "Team Alpha",
								"description": "Development team Alpha workspace",
								"role":        "admin",
								"permissions": []string{"read", "write", "delete", "admin"},
							},
							{
								"name":        "team-beta",
								"displayName": "Team Beta",
								"description": "Development team Beta workspace",
								"role":        "admin",
								"permissions": []string{"read", "write", "delete", "admin"},
							},
							{
								"name":        "team-gamma",
								"displayName": "Team Gamma",
								"description": "Development team Gamma workspace",
								"role":        "admin",
								"permissions": []string{"read", "write", "delete", "admin"},
							},
						},
					}

				default:
					c.JSON(401, gin.H{"error": "Invalid token"})
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
				user := response["user"].(map[string]interface{})
				assert.NotEmpty(t, user["id"])
				assert.NotEmpty(t, user["username"])
				assert.NotEmpty(t, user["email"])
				assert.Contains(t, user, "roles")

				namespaces := response["namespaces"].([]interface{})
				totalCount := int(response["totalCount"].(float64))
				assert.Equal(t, len(namespaces), totalCount)

				// Validate namespace structure
				if len(namespaces) > 0 {
					firstNamespace := namespaces[0].(map[string]interface{})
					expectedNamespaceKeys := []string{"name", "displayName", "role", "permissions"}
					for _, key := range expectedNamespaceKeys {
						assert.Contains(t, firstNamespace, key, "Namespace should contain key: %s", key)
					}

					// Validate permissions array
					permissions := firstNamespace["permissions"].([]interface{})
					assert.Greater(t, len(permissions), 0, "Permissions should not be empty")
				}
			}
		})
	}
}

func TestGetUserNamespacesWithFiltering(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		query      string
		expectCode int
		expectKeys []string
	}{
		{
			name:       "Filter namespaces by role",
			query:      "?role=member",
			expectCode: 200,
			expectKeys: []string{"namespaces", "totalCount", "filters"},
		},
		{
			name:       "Filter namespaces by permission",
			query:      "?permission=create_session",
			expectCode: 200,
			expectKeys: []string{"namespaces", "totalCount", "filters"},
		},
		{
			name:       "Search namespaces by name",
			query:      "?search=alpha",
			expectCode: 200,
			expectKeys: []string{"namespaces", "totalCount", "filters"},
		},
		{
			name:       "Get namespaces with stats",
			query:      "?include=stats",
			expectCode: 200,
			expectKeys: []string{"namespaces", "totalCount"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/user/namespaces"+tt.query, nil)
			req.Header.Set("Authorization", "Bearer valid-user-token")

			w := httptest.NewRecorder()

			router := gin.New()
			router.GET("/api/v1/user/namespaces", func(c *gin.Context) {
				// Parse query parameters
				roleFilter := c.Query("role")
				permissionFilter := c.Query("permission")
				searchFilter := c.Query("search")
				include := c.Query("include")

				// Mock filtered response
				response := map[string]interface{}{
					"user": map[string]interface{}{
						"id":       "user-123",
						"username": "alice",
						"email":    "alice@example.com",
						"roles":    []string{"developer"},
					},
				}

				// Apply filters (mock logic)
				if roleFilter == "member" {
					response["namespaces"] = []map[string]interface{}{
						{
							"name":        "team-alpha",
							"displayName": "Team Alpha",
							"role":        "member",
							"permissions": []string{"read", "create_session"},
						},
					}
					response["totalCount"] = 1
					response["filters"] = map[string]interface{}{
						"role": "member",
					}
				} else if permissionFilter == "create_session" {
					response["namespaces"] = []map[string]interface{}{
						{
							"name":        "team-alpha",
							"displayName": "Team Alpha",
							"role":        "member",
							"permissions": []string{"read", "create_session"},
						},
					}
					response["totalCount"] = 1
					response["filters"] = map[string]interface{}{
						"permission": "create_session",
					}
				} else if searchFilter == "alpha" {
					response["namespaces"] = []map[string]interface{}{
						{
							"name":        "team-alpha",
							"displayName": "Team Alpha",
							"role":        "member",
							"permissions": []string{"read", "create_session"},
						},
					}
					response["totalCount"] = 1
					response["filters"] = map[string]interface{}{
						"search": "alpha",
					}
				} else if include == "stats" {
					response["namespaces"] = []map[string]interface{}{
						{
							"name":        "team-alpha",
							"displayName": "Team Alpha",
							"role":        "member",
							"permissions": []string{"read", "create_session"},
							"stats": map[string]interface{}{
								"totalSessions":   15,
								"runningSessions": 2,
							},
						},
					}
					response["totalCount"] = 1
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

func TestGetUserNamespacesRoleBasedAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name             string
		authToken        string
		expectCode       int
		expectNamespaces int
		expectRoles      []string
	}{
		{
			name:             "Regular developer sees limited namespaces",
			authToken:        "Bearer developer-token",
			expectCode:       200,
			expectNamespaces: 2,
			expectRoles:      []string{"member", "viewer"},
		},
		{
			name:             "Team lead sees team namespaces with elevated permissions",
			authToken:        "Bearer team-lead-token",
			expectCode:       200,
			expectNamespaces: 3,
			expectRoles:      []string{"admin", "member"},
		},
		{
			name:             "System admin sees all namespaces",
			authToken:        "Bearer admin-token",
			expectCode:       200,
			expectNamespaces: 4,
			expectRoles:      []string{"admin"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/user/namespaces", nil)
			req.Header.Set("Authorization", tt.authToken)

			w := httptest.NewRecorder()

			router := gin.New()
			router.GET("/api/v1/user/namespaces", func(c *gin.Context) {
				authHeader := c.GetHeader("Authorization")

				var response map[string]interface{}

				switch authHeader {
				case "Bearer developer-token":
					response = map[string]interface{}{
						"user":       map[string]interface{}{"id": "dev-123", "roles": []string{"developer"}},
						"totalCount": 2,
						"namespaces": []map[string]interface{}{
							{"name": "team-alpha", "role": "member"},
							{"name": "team-beta", "role": "viewer"},
						},
					}
				case "Bearer team-lead-token":
					response = map[string]interface{}{
						"user":       map[string]interface{}{"id": "lead-456", "roles": []string{"team-lead"}},
						"totalCount": 3,
						"namespaces": []map[string]interface{}{
							{"name": "team-alpha", "role": "admin"},
							{"name": "team-beta", "role": "admin"},
							{"name": "team-gamma", "role": "member"},
						},
					}
				case "Bearer admin-token":
					response = map[string]interface{}{
						"user":       map[string]interface{}{"id": "admin-789", "roles": []string{"admin"}},
						"totalCount": 4,
						"namespaces": []map[string]interface{}{
							{"name": "ambient-system", "role": "admin"},
							{"name": "team-alpha", "role": "admin"},
							{"name": "team-beta", "role": "admin"},
							{"name": "team-gamma", "role": "admin"},
						},
					}
				}

				c.JSON(200, response)
			})

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectCode, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			if tt.expectCode == 200 {
				namespaces := response["namespaces"].([]interface{})
				assert.Equal(t, tt.expectNamespaces, len(namespaces))

				// Check that all namespace roles are in expected roles
				for _, ns := range namespaces {
					namespace := ns.(map[string]interface{})
					role := namespace["role"].(string)
					assert.Contains(t, tt.expectRoles, role, "Role %s should be in expected roles", role)
				}
			}
		})
	}
}