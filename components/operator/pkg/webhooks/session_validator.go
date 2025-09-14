package webhooks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// SessionValidator validates Session CRD operations
type SessionValidator struct {
	client   dynamic.Interface
	decoder  *admission.Decoder
}

// NamespacePolicy GVR for validation lookups
var namespacePolicyGVR = schema.GroupVersionResource{
	Group:    "ambient.ai",
	Version:  "v1alpha1",
	Resource: "namespacepolicies",
}

// Available runner framework types
var availableFrameworks = map[string]bool{
	"claude-code":     true,
	"custom-python":   true,
	"bash-runner":     true,
}

// NewSessionValidator creates a new session validation webhook
func NewSessionValidator(client dynamic.Interface) *SessionValidator {
	return &SessionValidator{
		client: client,
	}
}

// InjectDecoder injects the admission decoder
func (v *SessionValidator) InjectDecoder(d *admission.Decoder) error {
	v.decoder = d
	return nil
}

// Handle validates Session CRD create and update operations
func (v *SessionValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	session := &unstructured.Unstructured{}

	err := (*v.decoder).Decode(req, session)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	// Validate based on operation type
	switch req.Operation {
	case "CREATE":
		return v.validateCreate(ctx, session)
	case "UPDATE":
		return v.validateUpdate(ctx, session, req)
	default:
		return admission.Allowed("")
	}
}

// validateCreate validates session creation
func (v *SessionValidator) validateCreate(ctx context.Context, session *unstructured.Unstructured) admission.Response {
	// Validation Rule 1: spec.framework.type must match available runner types
	if err := v.validateFrameworkType(session); err != nil {
		return admission.Denied(fmt.Sprintf("Invalid framework type: %v", err))
	}

	// Validation Rule 2: spec.policy must comply with namespace policy constraints
	if err := v.validateAgainstNamespacePolicy(ctx, session); err != nil {
		return admission.Denied(fmt.Sprintf("Policy violation: %v", err))
	}

	// Validation Rule 3: metadata.namespace must be resolved server-side (enforced by API server)
	// This is handled by the webhook handler that sets the namespace

	return admission.Allowed("")
}

// validateUpdate validates session updates
func (v *SessionValidator) validateUpdate(ctx context.Context, newSession *unstructured.Unstructured, req admission.Request) admission.Response {
	// Get the old session for comparison
	oldSession := &unstructured.Unstructured{}
	if err := (*v.decoder).DecodeRaw(req.OldObject, oldSession); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	// Validation Rule 4: status.history is append-only, never modified or deleted
	if err := v.validateHistoryAppendOnly(oldSession, newSession); err != nil {
		return admission.Denied(fmt.Sprintf("History modification not allowed: %v", err))
	}

	// Allow other updates (spec changes are allowed for session configuration)
	return admission.Allowed("")
}

// validateFrameworkType ensures the framework type is supported
func (v *SessionValidator) validateFrameworkType(session *unstructured.Unstructured) error {
	spec, exists := session.Object["spec"].(map[string]interface{})
	if !exists {
		return fmt.Errorf("spec field is required")
	}

	framework, exists := spec["framework"].(map[string]interface{})
	if !exists {
		return fmt.Errorf("spec.framework field is required")
	}

	frameworkType, exists := framework["type"].(string)
	if !exists {
		return fmt.Errorf("spec.framework.type field is required")
	}

	if !availableFrameworks[frameworkType] {
		return fmt.Errorf("unsupported framework type '%s', available: %v",
			frameworkType, getAvailableFrameworksList())
	}

	return nil
}

// validateAgainstNamespacePolicy checks session against namespace policy constraints
func (v *SessionValidator) validateAgainstNamespacePolicy(ctx context.Context, session *unstructured.Unstructured) error {
	namespace := session.GetNamespace()
	if namespace == "" {
		return fmt.Errorf("namespace is required")
	}

	// Get namespace policy
	policy, err := v.client.Resource(namespacePolicyGVR).
		Namespace(namespace).
		Get(ctx, "policy", metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// No policy means no restrictions
			return nil
		}
		return fmt.Errorf("failed to get namespace policy: %v", err)
	}

	// Validate against policy constraints
	return v.validateSessionAgainstPolicy(session, policy)
}

// validateSessionAgainstPolicy validates session spec against policy
func (v *SessionValidator) validateSessionAgainstPolicy(session, policy *unstructured.Unstructured) error {
	sessionSpec, _ := session.Object["spec"].(map[string]interface{})
	policySpec, _ := policy.Object["spec"].(map[string]interface{})

	// Validate framework constraints
	if err := v.validateFrameworkConstraints(sessionSpec, policySpec); err != nil {
		return err
	}

	// Validate model constraints
	if err := v.validateModelConstraints(sessionSpec, policySpec); err != nil {
		return err
	}

	// Validate tool constraints
	if err := v.validateToolConstraints(sessionSpec, policySpec); err != nil {
		return err
	}

	return nil
}

// validateFrameworkConstraints validates framework type against policy
func (v *SessionValidator) validateFrameworkConstraints(sessionSpec, policySpec map[string]interface{}) error {
	framework, exists := sessionSpec["framework"].(map[string]interface{})
	if !exists {
		return nil
	}

	frameworkType, _ := framework["type"].(string)

	// Check if framework type is allowed by policy (if policy specifies restrictions)
	if frameworks, exists := policySpec["frameworks"].(map[string]interface{}); exists {
		if blocked, exists := frameworks["blocked"].([]interface{}); exists {
			for _, blockedType := range blocked {
				if blockedType.(string) == frameworkType {
					return fmt.Errorf("framework type '%s' is blocked by namespace policy", frameworkType)
				}
			}
		}

		if allowed, exists := frameworks["allowed"].([]interface{}); exists {
			isAllowed := false
			for _, allowedType := range allowed {
				if allowedType.(string) == frameworkType {
					isAllowed = true
					break
				}
			}
			if !isAllowed {
				return fmt.Errorf("framework type '%s' is not in allowed list", frameworkType)
			}
		}
	}

	return nil
}

// validateModelConstraints validates model constraints against policy
func (v *SessionValidator) validateModelConstraints(sessionSpec, policySpec map[string]interface{}) error {
	sessionPolicy, exists := sessionSpec["policy"].(map[string]interface{})
	if !exists {
		return nil
	}

	modelConstraints, exists := sessionPolicy["modelConstraints"].(map[string]interface{})
	if !exists {
		return nil
	}

	requestedModels, exists := modelConstraints["allowed"].([]interface{})
	if !exists {
		return nil
	}

	// Get policy model constraints
	policyModels, exists := policySpec["models"].(map[string]interface{})
	if !exists {
		return nil // No model policy means no restrictions
	}

	allowedModels, exists := policyModels["allowed"].([]interface{})
	if exists {
		// Validate requested models are in allowed list
		for _, requestedModel := range requestedModels {
			found := false
			for _, allowedModel := range allowedModels {
				if requestedModel.(string) == allowedModel.(string) {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("model '%s' is not allowed by namespace policy", requestedModel.(string))
			}
		}
	}

	blockedModels, exists := policyModels["blocked"].([]interface{})
	if exists {
		// Validate requested models are not in blocked list
		for _, requestedModel := range requestedModels {
			for _, blockedModel := range blockedModels {
				if requestedModel.(string) == blockedModel.(string) {
					return fmt.Errorf("model '%s' is blocked by namespace policy", requestedModel.(string))
				}
			}
		}
	}

	return nil
}

// validateToolConstraints validates tool constraints against policy
func (v *SessionValidator) validateToolConstraints(sessionSpec, policySpec map[string]interface{}) error {
	sessionPolicy, exists := sessionSpec["policy"].(map[string]interface{})
	if !exists {
		return nil
	}

	toolConstraints, exists := sessionPolicy["toolConstraints"].(map[string]interface{})
	if !exists {
		return nil
	}

	requestedTools, exists := toolConstraints["allowed"].([]interface{})
	if !exists {
		return nil
	}

	// Get policy tool constraints
	policyTools, exists := policySpec["tools"].(map[string]interface{})
	if !exists {
		return nil // No tool policy means no restrictions
	}

	allowedTools, exists := policyTools["allowed"].([]interface{})
	if exists {
		// Validate requested tools are in allowed list
		for _, requestedTool := range requestedTools {
			found := false
			for _, allowedTool := range allowedTools {
				if requestedTool.(string) == allowedTool.(string) {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("tool '%s' is not allowed by namespace policy", requestedTool.(string))
			}
		}
	}

	blockedTools, exists := policyTools["blocked"].([]interface{})
	if exists {
		// Validate requested tools are not in blocked list
		for _, requestedTool := range requestedTools {
			for _, blockedTool := range blockedTools {
				if requestedTool.(string) == blockedTool.(string) {
					return fmt.Errorf("tool '%s' is blocked by namespace policy", requestedTool.(string))
				}
			}
		}
	}

	return nil
}

// validateHistoryAppendOnly ensures status.history is only appended to, never modified
func (v *SessionValidator) validateHistoryAppendOnly(oldSession, newSession *unstructured.Unstructured) error {
	oldStatus, _ := oldSession.Object["status"].(map[string]interface{})
	newStatus, _ := newSession.Object["status"].(map[string]interface{})

	if oldStatus == nil {
		return nil // No history to validate
	}

	oldHistory, exists := oldStatus["history"].([]interface{})
	if !exists {
		return nil
	}

	newHistory, exists := newStatus["history"].([]interface{})
	if !exists {
		return fmt.Errorf("history cannot be removed")
	}

	// History can only grow, never shrink
	if len(newHistory) < len(oldHistory) {
		return fmt.Errorf("history cannot be shortened (old: %d, new: %d)", len(oldHistory), len(newHistory))
	}

	// Existing history entries cannot be modified
	for i, oldEntry := range oldHistory {
		if i >= len(newHistory) {
			return fmt.Errorf("history entry %d was removed", i)
		}

		// Compare entries as JSON to detect any modifications
		oldJSON, _ := json.Marshal(oldEntry)
		newJSON, _ := json.Marshal(newHistory[i])

		if string(oldJSON) != string(newJSON) {
			return fmt.Errorf("history entry %d was modified", i)
		}
	}

	return nil
}

// getAvailableFrameworksList returns list of available frameworks for error messages
func getAvailableFrameworksList() []string {
	frameworks := make([]string, 0, len(availableFrameworks))
	for framework := range availableFrameworks {
		frameworks = append(frameworks, framework)
	}
	return frameworks
}

// +kubebuilder:webhook:path=/validate-ambient-ai-v1alpha1-session,mutating=false,failurePolicy=fail,sideEffects=None,groups=ambient.ai,resources=sessions,verbs=create;update,versions=v1alpha1,name=vsession.kb.io,admissionReviewVersions=v1

var _ admission.Handler = &SessionValidator{}