package webhooks

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// PolicyValidator validates NamespacePolicy CRD operations
type PolicyValidator struct {
	decoder *admission.Decoder
}

// NewPolicyValidator creates a new policy validation webhook
func NewPolicyValidator() *PolicyValidator {
	return &PolicyValidator{}
}

// InjectDecoder injects the admission decoder
func (v *PolicyValidator) InjectDecoder(d *admission.Decoder) error {
	v.decoder = d
	return nil
}

// Handle validates NamespacePolicy CRD create and update operations
func (v *PolicyValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	policy := &unstructured.Unstructured{}

	err := (*v.decoder).Decode(req, policy)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	// Validate based on operation type
	switch req.Operation {
	case "CREATE", "UPDATE":
		return v.validatePolicy(ctx, policy)
	default:
		return admission.Allowed("")
	}
}

// validatePolicy validates namespace policy configuration
func (v *PolicyValidator) validatePolicy(ctx context.Context, policy *unstructured.Unstructured) admission.Response {
	spec, exists := policy.Object["spec"].(map[string]interface{})
	if !exists {
		return admission.Denied("spec field is required")
	}

	// Validate models configuration
	if err := v.validateModelsConfig(spec); err != nil {
		return admission.Denied(fmt.Sprintf("Invalid models configuration: %v", err))
	}

	// Validate tools configuration
	if err := v.validateToolsConfig(spec); err != nil {
		return admission.Denied(fmt.Sprintf("Invalid tools configuration: %v", err))
	}

	// Validate retention configuration
	if err := v.validateRetentionConfig(spec); err != nil {
		return admission.Denied(fmt.Sprintf("Invalid retention configuration: %v", err))
	}

	// Validate budget configuration
	if err := v.validateBudgetConfig(spec); err != nil {
		return admission.Denied(fmt.Sprintf("Invalid budget configuration: %v", err))
	}

	// Validate webhook authentication
	if err := v.validateWebhookAuth(spec); err != nil {
		return admission.Denied(fmt.Sprintf("Invalid webhook authentication: %v", err))
	}

	// Validate notification configuration
	if err := v.validateNotifications(spec); err != nil {
		return admission.Denied(fmt.Sprintf("Invalid notification configuration: %v", err))
	}

	return admission.Allowed("")
}

// validateModelsConfig validates the models configuration section
func (v *PolicyValidator) validateModelsConfig(spec map[string]interface{}) error {
	models, exists := spec["models"].(map[string]interface{})
	if !exists {
		return nil // models configuration is optional
	}

	// Validate allowed and blocked lists don't overlap
	allowed := getStringSlice(models, "allowed")
	blocked := getStringSlice(models, "blocked")

	for _, allowedModel := range allowed {
		for _, blockedModel := range blocked {
			if allowedModel == blockedModel {
				return fmt.Errorf("model '%s' cannot be both allowed and blocked", allowedModel)
			}
		}
	}

	// Validate budget format
	if budget, exists := models["budget"].(map[string]interface{}); exists {
		if monthly, exists := budget["monthly"].(string); exists {
			if !isValidBudgetFormat(monthly) {
				return fmt.Errorf("invalid budget format '%s', expected format: '100.00'", monthly)
			}
		}

		if currency, exists := budget["currency"].(string); exists {
			if currency != "USD" {
				return fmt.Errorf("unsupported currency '%s', only USD is supported", currency)
			}
		}

		if resetDay, exists := budget["resetDay"]; exists {
			day, ok := toInt(resetDay)
			if !ok || day < 1 || day > 28 {
				return fmt.Errorf("resetDay must be between 1 and 28, got %v", resetDay)
			}
		}
	}

	return nil
}

// validateToolsConfig validates the tools configuration section
func (v *PolicyValidator) validateToolsConfig(spec map[string]interface{}) error {
	tools, exists := spec["tools"].(map[string]interface{})
	if !exists {
		return nil // tools configuration is optional
	}

	// Validate allowed and blocked lists don't overlap
	allowed := getStringSlice(tools, "allowed")
	blocked := getStringSlice(tools, "blocked")

	for _, allowedTool := range allowed {
		for _, blockedTool := range blocked {
			if allowedTool == blockedTool {
				return fmt.Errorf("tool '%s' cannot be both allowed and blocked", allowedTool)
			}
		}
	}

	// Validate restrictions
	if restrictions, exists := tools["restrictions"].(map[string]interface{}); exists {
		// All restriction fields should be boolean
		for key, value := range restrictions {
			if _, ok := value.(bool); !ok {
				return fmt.Errorf("restriction '%s' must be a boolean value", key)
			}
		}
	}

	return nil
}

// validateRetentionConfig validates retention configuration
func (v *PolicyValidator) validateRetentionConfig(spec map[string]interface{}) error {
	retention, exists := spec["retention"].(map[string]interface{})
	if !exists {
		return nil // retention configuration is optional
	}

	// Validate retention period formats
	periodFields := []string{"sessions", "artifacts", "auditLogs"}
	for _, field := range periodFields {
		if period, exists := retention[field].(string); exists {
			if !isValidRetentionPeriod(period) {
				return fmt.Errorf("invalid retention period '%s' for %s, expected format: '30d', '1y', etc.", period, field)
			}
		}
	}

	return nil
}

// validateBudgetConfig validates budget configuration in the spec
func (v *PolicyValidator) validateBudgetConfig(spec map[string]interface{}) error {
	// Budget can be in models section or as a separate section
	if models, exists := spec["models"].(map[string]interface{}); exists {
		if budget, exists := models["budget"].(map[string]interface{}); exists {
			if monthly, exists := budget["monthly"].(string); exists {
				value, err := strconv.ParseFloat(monthly, 64)
				if err != nil {
					return fmt.Errorf("invalid monthly budget value '%s': %v", monthly, err)
				}
				if value < 0 {
					return fmt.Errorf("monthly budget cannot be negative: %s", monthly)
				}
				if value > 1000000 {
					return fmt.Errorf("monthly budget exceeds maximum allowed (1000000): %s", monthly)
				}
			}
		}
	}

	return nil
}

// validateWebhookAuth validates webhook authentication configuration
func (v *PolicyValidator) validateWebhookAuth(spec map[string]interface{}) error {
	webhookAuth, exists := spec["webhookAuth"].(map[string]interface{})
	if !exists {
		return nil // webhook auth is optional
	}

	// Validate API keys (should be hashed)
	if apiKeys, exists := webhookAuth["apiKeys"].(map[string]interface{}); exists {
		for source, key := range apiKeys {
			keyStr, ok := key.(string)
			if !ok {
				return fmt.Errorf("API key for source '%s' must be a string", source)
			}
			if len(keyStr) < 10 {
				return fmt.Errorf("API key for source '%s' is too short (min 10 characters)", source)
			}
		}
	}

	// Validate rate limit configuration
	if rateLimit, exists := webhookAuth["rateLimit"].(map[string]interface{}); exists {
		if rpm, exists := rateLimit["requestsPerMinute"]; exists {
			rpmInt, ok := toInt(rpm)
			if !ok || rpmInt < 1 || rpmInt > 1000 {
				return fmt.Errorf("requestsPerMinute must be between 1 and 1000, got %v", rpm)
			}
		}

		if burst, exists := rateLimit["burstSize"]; exists {
			burstInt, ok := toInt(burst)
			if !ok || burstInt < 1 || burstInt > 100 {
				return fmt.Errorf("burstSize must be between 1 and 100, got %v", burst)
			}
		}
	}

	return nil
}

// validateNotifications validates notification configuration
func (v *PolicyValidator) validateNotifications(spec map[string]interface{}) error {
	notifications, exists := spec["notifications"].(map[string]interface{})
	if !exists {
		return nil // notifications are optional
	}

	// Validate webhooks
	if webhooks, exists := notifications["webhooks"].([]interface{}); exists {
		for i, webhook := range webhooks {
			webhookMap, ok := webhook.(map[string]interface{})
			if !ok {
				return fmt.Errorf("webhook %d is not a valid configuration", i)
			}

			// Validate URL
			url, exists := webhookMap["url"].(string)
			if !exists || url == "" {
				return fmt.Errorf("webhook %d requires a URL", i)
			}
			if !isValidURL(url) {
				return fmt.Errorf("webhook %d has invalid URL: %s", i, url)
			}

			// Validate events
			events, exists := webhookMap["events"].([]interface{})
			if !exists || len(events) == 0 {
				return fmt.Errorf("webhook %d requires at least one event", i)
			}

			validEvents := map[string]bool{
				"session.created":   true,
				"session.started":   true,
				"session.completed": true,
				"session.failed":    true,
				"session.approved":  true,
				"session.rejected":  true,
				"budget.warning":    true,
				"budget.exceeded":   true,
			}

			for _, event := range events {
				eventStr, ok := event.(string)
				if !ok {
					return fmt.Errorf("webhook %d event must be a string", i)
				}
				if !validEvents[eventStr] {
					return fmt.Errorf("webhook %d has invalid event '%s'", i, eventStr)
				}
			}
		}
	}

	// Validate email configuration
	if email, exists := notifications["email"].(map[string]interface{}); exists {
		if enabled, exists := email["enabled"].(bool); exists && enabled {
			recipients, exists := email["recipients"].([]interface{})
			if !exists || len(recipients) == 0 {
				return fmt.Errorf("email notifications enabled but no recipients specified")
			}

			for i, recipient := range recipients {
				recipientStr, ok := recipient.(string)
				if !ok || !isValidEmail(recipientStr) {
					return fmt.Errorf("invalid email recipient at index %d: %v", i, recipient)
				}
			}
		}
	}

	return nil
}

// Helper functions

func getStringSlice(m map[string]interface{}, key string) []string {
	var result []string
	if list, exists := m[key].([]interface{}); exists {
		for _, item := range list {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
	}
	return result
}

func isValidBudgetFormat(budget string) bool {
	// Budget should be in format "100.00"
	match, _ := regexp.MatchString(`^\d+\.\d{2}$`, budget)
	return match
}

func isValidRetentionPeriod(period string) bool {
	// Period should be in format like "30d", "1y", "6m", "2w"
	match, _ := regexp.MatchString(`^\d+[dwmy]$`, period)
	return match
}

func isValidURL(url string) bool {
	// Basic URL validation
	return strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")
}

func isValidEmail(email string) bool {
	// Basic email validation
	match, _ := regexp.MatchString(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`, email)
	return match
}

func toInt(value interface{}) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case string:
		i, err := strconv.Atoi(v)
		return i, err == nil
	default:
		return 0, false
	}
}

// +kubebuilder:webhook:path=/validate-ambient-ai-v1alpha1-namespacepolicy,mutating=false,failurePolicy=fail,sideEffects=None,groups=ambient.ai,resources=namespacepolicies,verbs=create;update,versions=v1alpha1,name=vnamespacepolicy.kb.io,admissionReviewVersions=v1

var _ admission.Handler = &PolicyValidator{}