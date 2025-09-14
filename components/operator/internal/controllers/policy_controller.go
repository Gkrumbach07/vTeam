package controllers

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// PolicyReconciler reconciles NamespacePolicy objects
type PolicyReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	DynamicClient dynamic.Interface
}

// Reconcile handles NamespacePolicy CRD reconciliation
// Implements policy enforcement and constraint management
func (r *PolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the NamespacePolicy instance
	policy := &unstructured.Unstructured{}
	policy.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "ambient.ai",
		Version: "v1alpha1",
		Kind:    "NamespacePolicy",
	})

	err := r.Get(ctx, req.NamespacedName, policy)
	if err != nil {
		if errors.IsNotFound(err) {
			// Policy deleted, clean up any related resources
			return r.handlePolicyDeletion(ctx, req.NamespacedName)
		}
		return ctrl.Result{}, err
	}

	log.Info("Reconciling namespace policy", "namespace", policy.GetNamespace())

	// Validate policy configuration
	if err := r.validatePolicy(ctx, policy); err != nil {
		log.Error(err, "Policy validation failed")
		return r.updatePolicyStatus(ctx, policy, "Invalid", err.Error())
	}

	// Update policy status with current usage
	if err := r.updateUsageMetrics(ctx, policy); err != nil {
		log.Error(err, "Failed to update usage metrics")
		// Don't fail reconciliation for metrics update failure
	}

	// Check for policy violations in existing sessions
	if err := r.checkPolicyViolations(ctx, policy); err != nil {
		log.Error(err, "Failed to check policy violations")
		return ctrl.Result{}, err
	}

	// Set up retention cleanup if configured
	if err := r.scheduleRetentionCleanup(ctx, policy); err != nil {
		log.Error(err, "Failed to schedule retention cleanup")
		// Don't fail reconciliation for cleanup scheduling
	}

	// Update status to Valid
	return r.updatePolicyStatus(ctx, policy, "Valid", "Policy is active and enforced")
}

// validatePolicy validates the policy configuration
func (r *PolicyReconciler) validatePolicy(ctx context.Context, policy *unstructured.Unstructured) error {
	spec, _ := policy.Object["spec"].(map[string]interface{})
	if spec == nil {
		return fmt.Errorf("policy spec is required")
	}

	// Validate models configuration
	if models, ok := spec["models"].(map[string]interface{}); ok {
		if err := r.validateModelsConfig(models); err != nil {
			return fmt.Errorf("invalid models configuration: %w", err)
		}
	}

	// Validate tools configuration
	if tools, ok := spec["tools"].(map[string]interface{}); ok {
		if err := r.validateToolsConfig(tools); err != nil {
			return fmt.Errorf("invalid tools configuration: %w", err)
		}
	}

	// Validate retention configuration
	if retention, ok := spec["retention"].(map[string]interface{}); ok {
		if err := r.validateRetentionConfig(retention); err != nil {
			return fmt.Errorf("invalid retention configuration: %w", err)
		}
	}

	return nil
}

// validateModelsConfig validates the models section of the policy
func (r *PolicyReconciler) validateModelsConfig(models map[string]interface{}) error {
	// Check for conflicting allowed/blocked lists
	allowed := getStringSlice(models, "allowed")
	blocked := getStringSlice(models, "blocked")

	for _, a := range allowed {
		for _, b := range blocked {
			if a == b {
				return fmt.Errorf("model '%s' cannot be both allowed and blocked", a)
			}
		}
	}

	// Validate budget if specified
	if budget, ok := models["budget"].(map[string]interface{}); ok {
		if monthly, ok := budget["monthly"].(string); ok {
			if monthly == "" {
				return fmt.Errorf("monthly budget cannot be empty")
			}
		}
	}

	return nil
}

// validateToolsConfig validates the tools section of the policy
func (r *PolicyReconciler) validateToolsConfig(tools map[string]interface{}) error {
	// Check for conflicting allowed/blocked lists
	allowed := getStringSlice(tools, "allowed")
	blocked := getStringSlice(tools, "blocked")

	for _, a := range allowed {
		for _, b := range blocked {
			if a == b {
				return fmt.Errorf("tool '%s' cannot be both allowed and blocked", a)
			}
		}
	}

	return nil
}

// validateRetentionConfig validates the retention configuration
func (r *PolicyReconciler) validateRetentionConfig(retention map[string]interface{}) error {
	// Validate retention periods are valid durations
	fields := []string{"sessions", "artifacts", "auditLogs"}
	for _, field := range fields {
		if period, ok := retention[field].(string); ok {
			if _, err := parseRetentionPeriod(period); err != nil {
				return fmt.Errorf("invalid retention period for %s: %w", field, err)
			}
		}
	}

	return nil
}

// updateUsageMetrics updates the policy status with current usage metrics
func (r *PolicyReconciler) updateUsageMetrics(ctx context.Context, policy *unstructured.Unstructured) error {
	log := log.FromContext(ctx)
	namespace := policy.GetNamespace()

	// Count active sessions
	sessionList, err := r.DynamicClient.Resource(sessionGVR).
		Namespace(namespace).
		List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	activeSessions := 0
	totalSessions := len(sessionList.Items)

	for _, session := range sessionList.Items {
		status, _ := session.Object["status"].(map[string]interface{})
		phase, _ := status["phase"].(string)
		if phase == "Running" || phase == "Pending" {
			activeSessions++
		}
	}

	// Calculate budget usage (mock calculation for now)
	budgetUsed := 0.0
	percentUsed := 0.0

	spec, _ := policy.Object["spec"].(map[string]interface{})
	if models, ok := spec["models"].(map[string]interface{}); ok {
		if budget, ok := models["budget"].(map[string]interface{}); ok {
			if monthly, ok := budget["monthly"].(string); ok {
				// In a real implementation, we would calculate actual usage
				// For now, just mock some usage
				budgetUsed = float64(activeSessions) * 10.0 // $10 per active session
				var monthlyBudget float64
				fmt.Sscanf(monthly, "%f", &monthlyBudget)
				if monthlyBudget > 0 {
					percentUsed = (budgetUsed / monthlyBudget) * 100
				}
			}
		}
	}

	// Update status
	status, _ := policy.Object["status"].(map[string]interface{})
	if status == nil {
		status = make(map[string]interface{})
	}

	status["usage"] = map[string]interface{}{
		"budget": map[string]interface{}{
			"currentMonth": budgetUsed,
			"lastReset":    time.Now().Format(time.RFC3339),
			"percentUsed":  percentUsed,
		},
		"sessions": map[string]interface{}{
			"active": activeSessions,
			"total":  totalSessions,
		},
	}

	policy.Object["status"] = status

	if err := r.Status().Update(ctx, policy); err != nil {
		log.Error(err, "Failed to update policy usage metrics")
		return err
	}

	log.Info("Updated policy usage metrics",
		"activeSessions", activeSessions,
		"budgetUsed", budgetUsed,
		"percentUsed", percentUsed)

	return nil
}

// checkPolicyViolations checks for violations in existing sessions
func (r *PolicyReconciler) checkPolicyViolations(ctx context.Context, policy *unstructured.Unstructured) error {
	log := log.FromContext(ctx)
	namespace := policy.GetNamespace()

	// Get all sessions in namespace
	sessionList, err := r.DynamicClient.Resource(sessionGVR).
		Namespace(namespace).
		List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	violationCount := 0
	for _, session := range sessionList.Items {
		// Check if session violates current policy
		if violation := r.checkSessionViolation(session, policy); violation != nil {
			log.Info("Session policy violation detected",
				"session", session.GetName(),
				"violation", violation.Error())
			violationCount++

			// Mark session as failed if it violates policy
			r.markSessionViolation(ctx, &session, violation.Error())
		}
	}

	if violationCount > 0 {
		log.Info("Policy violations found", "count", violationCount)
		// Update policy status with violation count
		r.updatePolicyCondition(ctx, policy, "PolicyViolation",
			fmt.Sprintf("%d sessions violate current policy", violationCount))
	}

	return nil
}

// checkSessionViolation checks if a session violates the policy
func (r *PolicyReconciler) checkSessionViolation(session unstructured.Unstructured, policy *unstructured.Unstructured) error {
	sessionSpec, _ := session.Object["spec"].(map[string]interface{})
	policySpec, _ := policy.Object["spec"].(map[string]interface{})

	// Check model constraints
	if sessionPolicy, ok := sessionSpec["policy"].(map[string]interface{}); ok {
		if modelConstraints, ok := sessionPolicy["modelConstraints"].(map[string]interface{}); ok {
			if allowed, ok := modelConstraints["allowed"].([]interface{}); ok {
				if err := r.checkModelAllowed(allowed, policySpec); err != nil {
					return err
				}
			}
		}
	}

	// Check tool constraints
	if sessionPolicy, ok := sessionSpec["policy"].(map[string]interface{}); ok {
		if toolConstraints, ok := sessionPolicy["toolConstraints"].(map[string]interface{}); ok {
			if allowed, ok := toolConstraints["allowed"].([]interface{}); ok {
				if err := r.checkToolsAllowed(allowed, policySpec); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// checkModelAllowed checks if models are allowed by policy
func (r *PolicyReconciler) checkModelAllowed(requestedModels []interface{}, policySpec map[string]interface{}) error {
	if models, ok := policySpec["models"].(map[string]interface{}); ok {
		allowed := getInterfaceSlice(models, "allowed")
		blocked := getInterfaceSlice(models, "blocked")

		for _, requested := range requestedModels {
			requestedStr := requested.(string)

			// Check if blocked
			for _, blockedModel := range blocked {
				if requestedStr == blockedModel.(string) {
					return fmt.Errorf("model '%s' is blocked by policy", requestedStr)
				}
			}

			// Check if in allowed list (if specified)
			if len(allowed) > 0 {
				found := false
				for _, allowedModel := range allowed {
					if requestedStr == allowedModel.(string) {
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("model '%s' is not in allowed list", requestedStr)
				}
			}
		}
	}

	return nil
}

// checkToolsAllowed checks if tools are allowed by policy
func (r *PolicyReconciler) checkToolsAllowed(requestedTools []interface{}, policySpec map[string]interface{}) error {
	if tools, ok := policySpec["tools"].(map[string]interface{}); ok {
		allowed := getInterfaceSlice(tools, "allowed")
		blocked := getInterfaceSlice(tools, "blocked")

		for _, requested := range requestedTools {
			requestedStr := requested.(string)

			// Check if blocked
			for _, blockedTool := range blocked {
				if requestedStr == blockedTool.(string) {
					return fmt.Errorf("tool '%s' is blocked by policy", requestedStr)
				}
			}

			// Check if in allowed list (if specified)
			if len(allowed) > 0 {
				found := false
				for _, allowedTool := range allowed {
					if requestedStr == allowedTool.(string) {
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("tool '%s' is not in allowed list", requestedStr)
				}
			}
		}
	}

	return nil
}

// markSessionViolation marks a session as having a policy violation
func (r *PolicyReconciler) markSessionViolation(ctx context.Context, session *unstructured.Unstructured, violation string) error {
	status, _ := session.Object["status"].(map[string]interface{})
	if status == nil {
		status = make(map[string]interface{})
	}

	// Only mark as failed if not already in terminal state
	phase, _ := status["phase"].(string)
	if phase != "Completed" && phase != "Failed" {
		status["phase"] = "Failed"
		status["violationReason"] = violation

		// Add to history
		history, _ := status["history"].([]interface{})
		if history == nil {
			history = make([]interface{}, 0)
		}

		historyEntry := map[string]interface{}{
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"event":     "PolicyViolation",
			"data": map[string]interface{}{
				"reason": violation,
			},
		}
		history = append(history, historyEntry)
		status["history"] = history

		session.Object["status"] = status
		return r.Status().Update(ctx, session)
	}

	return nil
}

// scheduleRetentionCleanup schedules cleanup based on retention policy
func (r *PolicyReconciler) scheduleRetentionCleanup(ctx context.Context, policy *unstructured.Unstructured) error {
	log := log.FromContext(ctx)

	spec, _ := policy.Object["spec"].(map[string]interface{})
	retention, ok := spec["retention"].(map[string]interface{})
	if !ok {
		return nil // No retention policy
	}

	// Parse retention periods
	sessionRetention, _ := retention["sessions"].(string)
	artifactRetention, _ := retention["artifacts"].(string)

	if sessionRetention != "" {
		duration, err := parseRetentionPeriod(sessionRetention)
		if err == nil {
			// Schedule cleanup (in a real implementation, this would create a CronJob)
			log.Info("Would schedule session cleanup", "retention", sessionRetention, "duration", duration)
		}
	}

	if artifactRetention != "" {
		duration, err := parseRetentionPeriod(artifactRetention)
		if err == nil {
			// Schedule cleanup (in a real implementation, this would create a CronJob)
			log.Info("Would schedule artifact cleanup", "retention", artifactRetention, "duration", duration)
		}
	}

	return nil
}

// updatePolicyStatus updates the policy status
func (r *PolicyReconciler) updatePolicyStatus(ctx context.Context, policy *unstructured.Unstructured, condition, message string) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	status, _ := policy.Object["status"].(map[string]interface{})
	if status == nil {
		status = make(map[string]interface{})
	}

	// Update conditions
	conditions, _ := status["conditions"].([]interface{})
	if conditions == nil {
		conditions = make([]interface{}, 0)
	}

	conditionEntry := map[string]interface{}{
		"type":               "ConfigValid",
		"status":             condition == "Valid",
		"lastTransitionTime": time.Now().UTC().Format(time.RFC3339),
		"reason":             condition,
		"message":            message,
	}
	conditions = append(conditions, conditionEntry)
	status["conditions"] = conditions

	policy.Object["status"] = status

	if err := r.Status().Update(ctx, policy); err != nil {
		log.Error(err, "Failed to update policy status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// updatePolicyCondition adds a condition to the policy status
func (r *PolicyReconciler) updatePolicyCondition(ctx context.Context, policy *unstructured.Unstructured, conditionType, message string) error {
	status, _ := policy.Object["status"].(map[string]interface{})
	if status == nil {
		status = make(map[string]interface{})
	}

	conditions, _ := status["conditions"].([]interface{})
	if conditions == nil {
		conditions = make([]interface{}, 0)
	}

	condition := map[string]interface{}{
		"type":               conditionType,
		"status":             "True",
		"lastTransitionTime": time.Now().UTC().Format(time.RFC3339),
		"message":            message,
	}
	conditions = append(conditions, condition)
	status["conditions"] = conditions

	policy.Object["status"] = status
	return r.Status().Update(ctx, policy)
}

// handlePolicyDeletion handles cleanup when a policy is deleted
func (r *PolicyReconciler) handlePolicyDeletion(ctx context.Context, namespacedName client.ObjectKey) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Handling policy deletion", "namespace", namespacedName.Namespace)

	// In a real implementation, we might:
	// 1. Stop any scheduled cleanup jobs
	// 2. Remove webhook configurations
	// 3. Clean up any policy-specific resources

	return ctrl.Result{}, nil
}

// Helper functions

func getStringSlice(m map[string]interface{}, key string) []string {
	var result []string
	if list, ok := m[key].([]interface{}); ok {
		for _, item := range list {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
	}
	return result
}

func getInterfaceSlice(m map[string]interface{}, key string) []interface{} {
	if list, ok := m[key].([]interface{}); ok {
		return list
	}
	return []interface{}{}
}

func parseRetentionPeriod(period string) (time.Duration, error) {
	// Parse periods like "30d", "1y", "6m"
	if len(period) < 2 {
		return 0, fmt.Errorf("invalid period format")
	}

	valueStr := period[:len(period)-1]
	unit := period[len(period)-1:]

	var value int
	if _, err := fmt.Sscanf(valueStr, "%d", &value); err != nil {
		return 0, err
	}

	switch unit {
	case "d":
		return time.Duration(value) * 24 * time.Hour, nil
	case "w":
		return time.Duration(value) * 7 * 24 * time.Hour, nil
	case "m":
		return time.Duration(value) * 30 * 24 * time.Hour, nil
	case "y":
		return time.Duration(value) * 365 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unknown time unit: %s", unit)
	}
}

// SetupWithManager sets up the controller with the Manager
func (r *PolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&unstructured.Unstructured{}).
		Complete(r)
}

// +kubebuilder:rbac:groups=ambient.ai,resources=namespacepolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ambient.ai,resources=namespacepolicies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ambient.ai,resources=namespacepolicies/finalizers,verbs=update
// +kubebuilder:rbac:groups=ambient.ai,resources=sessions,verbs=get;list;watch;update;patch