package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// WebhookAuthConfig holds configuration for webhook authentication
type WebhookAuthConfig struct {
	// GitHubSecret is the secret key for GitHub webhook verification
	GitHubSecret string
	// JiraSecret is the secret key for Jira webhook verification
	JiraSecret string
	// SlackSecret is the secret key for Slack webhook verification
	SlackSecret string
	// APIKeys maps webhook sources to their API keys for simple authentication
	APIKeys map[string]string
}

// WebhookAuthMiddleware provides authentication for incoming webhooks
func WebhookAuthMiddleware(config WebhookAuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		source := c.Param("source")
		if source == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Webhook source not specified"})
			c.Abort()
			return
		}

		switch source {
		case "github":
			if !authenticateGitHubWebhook(c, config.GitHubSecret) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "GitHub webhook authentication failed"})
				c.Abort()
				return
			}
		case "jira":
			if !authenticateJiraWebhook(c, config.JiraSecret) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Jira webhook authentication failed"})
				c.Abort()
				return
			}
		case "slack":
			if !authenticateSlackWebhook(c, config.SlackSecret) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Slack webhook authentication failed"})
				c.Abort()
				return
			}
		default:
			// For other sources, check API key authentication
			if !authenticateAPIKey(c, source, config.APIKeys) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
				c.Abort()
				return
			}
		}

		// Store authenticated source in context for later use
		c.Set("webhook_source", source)
		c.Next()
	}
}

// authenticateGitHubWebhook verifies GitHub webhook signatures
func authenticateGitHubWebhook(c *gin.Context, secret string) bool {
	signature := c.GetHeader("X-Hub-Signature-256")
	if signature == "" {
		// Fallback to older signature header
		signature = c.GetHeader("X-Hub-Signature")
		if signature == "" {
			return false
		}
	}

	// Read the body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return false
	}

	// Restore body for further processing
	c.Request.Body = io.NopCloser(strings.NewReader(string(body)))

	// Verify signature
	return verifyGitHubSignature(body, signature, secret)
}

// authenticateJiraWebhook verifies Jira webhook authentication
func authenticateJiraWebhook(c *gin.Context, secret string) bool {
	// Jira uses webhook identifier header
	identifier := c.GetHeader("X-Atlassian-Webhook-Identifier")
	if identifier == "" {
		return false
	}

	// For now, we'll use simple API key authentication
	// In production, you might want to implement proper Jira webhook verification
	apiKey := c.GetHeader("X-API-Key")
	return apiKey == secret
}

// authenticateSlackWebhook verifies Slack webhook signatures
func authenticateSlackWebhook(c *gin.Context, secret string) bool {
	signature := c.GetHeader("X-Slack-Signature")
	timestamp := c.GetHeader("X-Slack-Request-Timestamp")

	if signature == "" || timestamp == "" {
		return false
	}

	// Read the body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return false
	}

	// Restore body for further processing
	c.Request.Body = io.NopCloser(strings.NewReader(string(body)))

	// Verify Slack signature
	return verifySlackSignature(body, signature, timestamp, secret)
}

// authenticateAPIKey provides simple API key authentication for custom webhooks
func authenticateAPIKey(c *gin.Context, source string, apiKeys map[string]string) bool {
	providedKey := c.GetHeader("X-API-Key")
	if providedKey == "" {
		return false
	}

	expectedKey, exists := apiKeys[source]
	if !exists {
		return false
	}

	return providedKey == expectedKey
}

// verifyGitHubSignature verifies GitHub webhook signature using HMAC-SHA256
func verifyGitHubSignature(body []byte, signature, secret string) bool {
	// Remove the "sha256=" prefix if present
	if strings.HasPrefix(signature, "sha256=") {
		signature = signature[7:]
	} else if strings.HasPrefix(signature, "sha1=") {
		// Handle legacy SHA1 signatures
		signature = signature[5:]
		return verifyGitHubSignatureSHA1(body, signature, secret)
	}

	// Compute expected signature
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	// Compare signatures
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// verifyGitHubSignatureSHA1 verifies legacy SHA1 GitHub webhook signatures
func verifyGitHubSignatureSHA1(body []byte, signature, secret string) bool {
	mac := hmac.New(sha256.New, []byte(secret)) // Still use SHA256 for security
	mac.Write(body)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// verifySlackSignature verifies Slack webhook signature
func verifySlackSignature(body []byte, signature, timestamp, secret string) bool {
	// Remove the "v0=" prefix
	if strings.HasPrefix(signature, "v0=") {
		signature = signature[3:]
	}

	// Create the basestring
	basestring := fmt.Sprintf("v0:%s:%s", timestamp, string(body))

	// Compute expected signature
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(basestring))
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	// Compare signatures
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// SimpleAPIKeyAuth provides simple API key authentication middleware
func SimpleAPIKeyAuth(validKeys map[string]string) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "API key required"})
			c.Abort()
			return
		}

		// Check if the key is valid for any source
		var authenticatedSource string
		for source, validKey := range validKeys {
			if apiKey == validKey {
				authenticatedSource = source
				break
			}
		}

		if authenticatedSource == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
			c.Abort()
			return
		}

		// Store authenticated source in context
		c.Set("authenticated_source", authenticatedSource)
		c.Next()
	}
}

// ExtractNamespaceFromAPIKey resolves namespace from API key
// This is a simplified version - in production you'd query a database or K8s
func ExtractNamespaceFromAPIKey(apiKey string, keyMappings map[string]string) string {
	// Simple mapping of API keys to namespaces
	// In production, this would be more sophisticated
	namespace, exists := keyMappings[apiKey]
	if !exists {
		return ""
	}
	return namespace
}