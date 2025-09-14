package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	authv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// RBACConfig holds configuration for RBAC middleware
type RBACConfig struct {
	KubernetesClient kubernetes.Interface
	SystemNamespace  string // Namespace for system operations
}

// Permission represents a required permission for an endpoint
type Permission struct {
	Resource   string // e.g., "sessions", "artifacts"
	Verb       string // e.g., "get", "create", "delete"
	APIGroup   string // e.g., "ambient.ai" or ""
	Namespace  string // Target namespace (can be extracted from URL)
}

// RBACMiddleware provides role-based access control using Kubernetes RBAC
func RBACMiddleware(config RBACConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract user information from token
		user, err := extractUserFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing authentication token"})
			c.Abort()
			return
		}

		// Store user info in context for later use
		c.Set("authenticated_user", user)
		c.Next()
	}
}

// RequirePermission creates a middleware that checks specific permissions
func RequirePermission(config RBACConfig, permission Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, exists := c.Get("authenticated_user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			c.Abort()
			return
		}

		userInfo := user.(*UserInfo)

		// Resolve namespace from URL parameters if needed
		targetNamespace := permission.Namespace
		if targetNamespace == "" {
			targetNamespace = c.Param("namespace")
		}
		if targetNamespace == "" {
			targetNamespace = c.Param("ns")
		}

		// Check if user has the required permission
		hasPermission, err := checkKubernetesPermission(
			config.KubernetesClient,
			userInfo,
			permission.Resource,
			permission.Verb,
			targetNamespace,
			permission.APIGroup,
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to verify permissions",
			})
			c.Abort()
			return
		}

		if !hasPermission {
			c.JSON(http.StatusForbidden, gin.H{
				"error":      "Insufficient permissions",
				"required":   fmt.Sprintf("%s:%s on %s in namespace %s", permission.Verb, permission.Resource, permission.APIGroup, targetNamespace),
				"user":       userInfo.Username,
				"namespace":  targetNamespace,
			})
			c.Abort()
			return
		}

		// Store resolved namespace for handlers to use
		c.Set("target_namespace", targetNamespace)
		c.Next()
	}
}

// UserInfo holds authenticated user information
type UserInfo struct {
	Username string
	Groups   []string
	UID      string
	Token    string
}

// extractUserFromToken extracts user information from the authorization token
func extractUserFromToken(c *gin.Context) (*UserInfo, error) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return nil, fmt.Errorf("authorization header missing")
	}

	if !strings.HasPrefix(authHeader, "Bearer ") {
		return nil, fmt.Errorf("invalid authorization header format")
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")

	// In a real implementation, you would:
	// 1. Validate the JWT token
	// 2. Extract user claims
	// 3. Verify token signature
	// 4. Check token expiration
	//
	// For now, we'll implement a simplified version
	userInfo, err := parseJWTToken(token)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %v", err)
	}

	return userInfo, nil
}

// parseJWTToken parses and validates a JWT token
// This is a simplified implementation - in production use a proper JWT library
func parseJWTToken(token string) (*UserInfo, error) {
	// Mock implementation for testing
	// In production, implement proper JWT parsing with libraries like:
	// - github.com/golang-jwt/jwt/v4
	// - github.com/lestrrat-go/jwx

	switch token {
	case "valid-user-token":
		return &UserInfo{
			Username: "alice",
			Groups:   []string{"developers", "team-alpha"},
			UID:      "user-123",
			Token:    token,
		}, nil
	case "admin-token":
		return &UserInfo{
			Username: "admin",
			Groups:   []string{"system:masters", "admins"},
			UID:      "admin-456",
			Token:    token,
		}, nil
	case "team-lead-token":
		return &UserInfo{
			Username: "team-lead",
			Groups:   []string{"team-leads", "team-alpha", "team-beta"},
			UID:      "lead-789",
			Token:    token,
		}, nil
	default:
		return nil, fmt.Errorf("invalid or expired token")
	}
}

// checkKubernetesPermission uses Kubernetes SubjectAccessReview to check permissions
func checkKubernetesPermission(
	client kubernetes.Interface,
	user *UserInfo,
	resource, verb, namespace, apiGroup string,
) (bool, error) {
	// Create a SubjectAccessReview request
	sar := &authv1.SubjectAccessReview{
		Spec: authv1.SubjectAccessReviewSpec{
			User:   user.Username,
			Groups: user.Groups,
			UID:    user.UID,
			ResourceAttributes: &authv1.ResourceAttributes{
				Namespace: namespace,
				Verb:      verb,
				Group:     apiGroup,
				Resource:  resource,
			},
		},
	}

	// Submit the request to Kubernetes API
	result, err := client.AuthorizationV1().SubjectAccessReviews().Create(
		context.TODO(),
		sar,
		metav1.CreateOptions{},
	)

	if err != nil {
		return false, fmt.Errorf("failed to check permissions: %v", err)
	}

	return result.Status.Allowed, nil
}

// RequireNamespaceAccess is a convenience function for namespace-level access
func RequireNamespaceAccess(config RBACConfig, verb string) gin.HandlerFunc {
	return RequirePermission(config, Permission{
		Resource: "namespaces",
		Verb:     verb,
		APIGroup: "",
	})
}

// RequireSessionAccess is a convenience function for session access
func RequireSessionAccess(config RBACConfig, verb string) gin.HandlerFunc {
	return RequirePermission(config, Permission{
		Resource: "sessions",
		Verb:     verb,
		APIGroup: "ambient.ai",
	})
}

// RequireArtifactAccess is a convenience function for artifact access
func RequireArtifactAccess(config RBACConfig, verb string) gin.HandlerFunc {
	return RequirePermission(config, Permission{
		Resource: "artifacts",
		Verb:     verb,
		APIGroup: "ambient.ai",
	})
}

// IsSystemAdmin checks if the user has system admin privileges
func IsSystemAdmin(user *UserInfo) bool {
	for _, group := range user.Groups {
		if group == "system:masters" || group == "admins" {
			return true
		}
	}
	return false
}

// GetUserNamespaces returns namespaces the user has access to
func GetUserNamespaces(config RBACConfig, user *UserInfo) ([]string, error) {
	// This would typically query Kubernetes to find all namespaces
	// the user has access to. For now, we'll return a mock list.

	if IsSystemAdmin(user) {
		// System admins can see all namespaces
		return []string{"ambient-system", "team-alpha", "team-beta", "team-gamma"}, nil
	}

	// Regular users see namespaces based on their groups
	var namespaces []string
	for _, group := range user.Groups {
		switch group {
		case "team-alpha":
			namespaces = append(namespaces, "team-alpha")
		case "team-beta":
			namespaces = append(namespaces, "team-beta")
		case "team-gamma":
			namespaces = append(namespaces, "team-gamma")
		}
	}

	return namespaces, nil
}

// CreateKubernetesClient creates a Kubernetes client from in-cluster config
func CreateKubernetesClient() (kubernetes.Interface, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create in-cluster config: %v", err)
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %v", err)
	}

	return client, nil
}

// ServiceAccountAuth middleware for service-to-service authentication
func ServiceAccountAuth(validServiceAccounts map[string]bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		saToken := c.GetHeader("Authorization")
		if saToken == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Service account token required"})
			c.Abort()
			return
		}

		// Extract and validate service account token
		// In production, validate the token against Kubernetes API
		if strings.HasPrefix(saToken, "Bearer sa-") {
			saName := strings.TrimPrefix(saToken, "Bearer sa-")
			if validServiceAccounts[saName] {
				c.Set("service_account", saName)
				c.Next()
				return
			}
		}

		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid service account token"})
		c.Abort()
	}
}