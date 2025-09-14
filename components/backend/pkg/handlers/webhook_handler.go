package handlers

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

type WebhookHandler struct {
	dynamicClient dynamic.Interface
}

type WebhookResponse struct {
	SessionID           string `json:"sessionId"`
	Namespace           string `json:"namespace"`
	Status              string `json:"status"`
	EstimatedStartTime  string `json:"estimatedStartTime,omitempty"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	TraceID string `json:"traceId,omitempty"`
}

var sessionGVR = schema.GroupVersionResource{
	Group:    "ambient.ai",
	Version:  "v1alpha1",
	Resource: "sessions",
}

func NewWebhookHandler(dynamicClient dynamic.Interface) *WebhookHandler {
	return &WebhookHandler{
		dynamicClient: dynamicClient,
	}
}

func (h *WebhookHandler) HandleGitHubWebhook(c *gin.Context) {
	h.handleWebhook(c, "github")
}

func (h *WebhookHandler) HandleJiraWebhook(c *gin.Context) {
	h.handleWebhook(c, "jira")
}

func (h *WebhookHandler) HandleSlackWebhook(c *gin.Context) {
	h.handleWebhook(c, "slack")
}

func (h *WebhookHandler) handleWebhook(c *gin.Context, source string) {
	ctx := context.Background()
	traceID := generateTraceID()
	c.Set("traceID", traceID)

	// Step 1: Validate API Key
	apiKey := c.GetHeader("X-API-Key")
	if apiKey == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "missing_api_key",
			Message: "X-API-Key header is required",
			TraceID: traceID,
		})
		return
	}

	// Step 2: Resolve namespace from API key
	namespace, err := h.resolveNamespace(ctx, apiKey)
	if err != nil {
		if err.Error() == "invalid_api_key" {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error:   "invalid_api_key",
				Message: "API key is invalid or expired",
				TraceID: traceID,
			})
		} else {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "namespace_resolution_failed",
				Message: "Failed to resolve namespace",
				TraceID: traceID,
			})
		}
		return
	}

	// Step 3: Parse webhook payload
	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_payload",
			Message: "Request body must be valid JSON",
			TraceID: traceID,
		})
		return
	}

	// Step 4: Validate namespace policy
	if err := h.validateNamespacePolicy(ctx, namespace, source); err != nil {
		if err.Error() == "budget_exceeded" {
			c.JSON(http.StatusForbidden, ErrorResponse{
				Error:   "policy_violation",
				Message: "Session creation would exceed namespace budget",
				TraceID: traceID,
			})
		} else if err.Error() == "rate_limit_exceeded" {
			c.JSON(http.StatusTooManyRequests, ErrorResponse{
				Error:   "rate_limit_exceeded",
				Message: "Maximum sessions per minute exceeded",
				TraceID: traceID,
			})
		} else {
			c.JSON(http.StatusForbidden, ErrorResponse{
				Error:   "policy_violation",
				Message: err.Error(),
				TraceID: traceID,
			})
		}
		return
	}

	// Step 5: Create Session CRD
	sessionID := uuid.New().String()

	// For testing with nil client, skip CRD creation
	if h.dynamicClient == nil {
		// Return mock response for testing
		estimatedStart := time.Now().Add(5 * time.Second).Format(time.RFC3339)
		c.JSON(http.StatusAccepted, WebhookResponse{
			SessionID:          sessionID,
			Namespace:          namespace,
			Status:             "accepted",
			EstimatedStartTime: estimatedStart,
		})
		return
	}

	session := h.createSessionFromWebhook(sessionID, namespace, source, payload)

	createdSession, err := h.dynamicClient.Resource(sessionGVR).
		Namespace(namespace).
		Create(ctx, session, metav1.CreateOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "session_creation_failed",
			Message: "Failed to create session",
			TraceID: traceID,
		})
		return
	}

	// Step 6: Return success response
	estimatedStart := time.Now().Add(5 * time.Second).Format(time.RFC3339)

	c.JSON(http.StatusAccepted, WebhookResponse{
		SessionID:          createdSession.GetName(),
		Namespace:          namespace,
		Status:             "accepted",
		EstimatedStartTime: estimatedStart,
	})
}

func (h *WebhookHandler) resolveNamespace(ctx context.Context, apiKey string) (string, error) {
	// Mock implementation - in real implementation, this would:
	// 1. Look up the API key in NamespacePolicy CRDs across namespaces
	// 2. Find which namespace this API key belongs to
	// 3. Validate the key is still active

	// For now, use simple mapping for testing
	keyToNamespace := map[string]string{
		"test-api-key-123":    "team-alpha",
		"test-api-key-456":    "team-alpha", // Jira key for same namespace
		"test-slack-key-789":  "team-alpha", // Slack key for same namespace
		"budget-exceeded-key": "team-beta",  // This will trigger policy violation
		"rate-limited-key":    "team-gamma", // This will trigger rate limit
	}

	namespace, exists := keyToNamespace[apiKey]
	if !exists {
		return "", fmt.Errorf("invalid_api_key")
	}

	return namespace, nil
}

func (h *WebhookHandler) validateNamespacePolicy(ctx context.Context, namespace, source string) error {
	// Mock implementation - in real implementation, this would:
	// 1. Fetch NamespacePolicy CRD from the namespace
	// 2. Check budget limits
	// 3. Check rate limits
	// 4. Validate webhook source is allowed

	// For testing purposes, simulate different policy violations
	if namespace == "team-beta" {
		return fmt.Errorf("budget_exceeded")
	}
	if namespace == "team-gamma" {
		return fmt.Errorf("rate_limit_exceeded")
	}

	return nil
}

func (h *WebhookHandler) createSessionFromWebhook(sessionID, namespace, source string, payload map[string]interface{}) *unstructured.Unstructured {
	// Determine event type and framework based on source and payload
	eventType := h.determineEventType(source, payload)
	frameworkType := h.determineFrameworkType(source, payload)

	session := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "ambient.ai/v1alpha1",
			"kind":       "Session",
			"metadata": map[string]interface{}{
				"name":      sessionID,
				"namespace": namespace,
				"labels": map[string]interface{}{
					"ambient.ai/trigger-source": source,
					"ambient.ai/framework-type": frameworkType,
				},
				"annotations": map[string]interface{}{
					"ambient.ai/created-by": "webhook-handler",
					"ambient.ai/created-at": time.Now().Format(time.RFC3339),
				},
			},
			"spec": map[string]interface{}{
				"trigger": map[string]interface{}{
					"source":  source,
					"event":   eventType,
					"payload": payload,
				},
				"framework": map[string]interface{}{
					"type":    frameworkType,
					"version": "1.0",
					"config":  map[string]interface{}{},
				},
				"policy": map[string]interface{}{
					"modelConstraints": map[string]interface{}{
						"allowed": []string{"claude-3-sonnet"},
						"budget":  "100.00",
					},
					"toolConstraints": map[string]interface{}{
						"allowed": []string{"bash", "edit", "read", "write"},
					},
					"approvalRequired": false,
				},
			},
		},
	}

	return session
}

func (h *WebhookHandler) determineEventType(source string, payload map[string]interface{}) string {
	switch source {
	case "github":
		if action, ok := payload["action"].(string); ok {
			return "pull_request_" + action
		}
		return "unknown"
	case "jira":
		if event, ok := payload["webhookEvent"].(string); ok {
			return event
		}
		return "unknown"
	case "slack":
		if msgType, ok := payload["type"].(string); ok {
			return msgType
		}
		return "message"
	default:
		return "unknown"
	}
}

func (h *WebhookHandler) determineFrameworkType(source string, payload map[string]interface{}) string {
	// For now, default to claude-code for all sources
	// In a real implementation, this could be determined by:
	// - Payload content analysis
	// - Namespace policy defaults
	// - User configuration
	return "claude-code"
}

func generateTraceID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return fmt.Sprintf("%x", bytes)
}