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
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// SessionReconciler reconciles Session objects
type SessionReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	DynamicClient dynamic.Interface
}

// Session GVR
var sessionGVR = schema.GroupVersionResource{
	Group:    "ambient.ai",
	Version:  "v1alpha1",
	Resource: "sessions",
}

// NamespacePolicy GVR
var namespacePolicyGVR = schema.GroupVersionResource{
	Group:    "ambient.ai",
	Version:  "v1alpha1",
	Resource: "namespacepolicies",
}

// Session phases from data-model.md lines 79-84
const (
	PhasePending   = "Pending"
	PhaseRunning   = "Running"
	PhaseCompleted = "Completed"
	PhaseFailed    = "Failed"
)

// Reconcile handles Session CRD reconciliation
// Implements state transitions from data-model.md lines 79-84:
// 1. Pending → Running: Policy validated, workload created
// 2. Running → Completed: Workload finished successfully
// 3. Running → Failed: Workload failed or exceeded constraints
// 4. Any state → Failed: Policy violation detected
func (r *SessionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the Session instance
	session := &unstructured.Unstructured{}
	session.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "ambient.ai",
		Version: "v1alpha1",
		Kind:    "Session",
	})

	err := r.Get(ctx, req.NamespacedName, session)
	if err != nil {
		if errors.IsNotFound(err) {
			// Session deleted, nothing to do
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Get current phase
	phase := r.getPhase(session)
	log.Info("Reconciling session", "name", session.GetName(), "phase", phase)

	// Handle based on current phase
	switch phase {
	case PhasePending:
		return r.handlePending(ctx, session)
	case PhaseRunning:
		return r.handleRunning(ctx, session)
	case PhaseCompleted, PhaseFailed:
		// Terminal states, nothing to do
		return ctrl.Result{}, nil
	default:
		// Unknown phase, set to pending
		return r.updatePhase(ctx, session, PhasePending, "Unknown phase detected, resetting to Pending")
	}
}

// handlePending handles sessions in Pending phase
func (r *SessionReconciler) handlePending(ctx context.Context, session *unstructured.Unstructured) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Step 1: Validate against namespace policy
	if err := r.validatePolicy(ctx, session); err != nil {
		log.Error(err, "Policy validation failed")
		return r.updatePhase(ctx, session, PhaseFailed, fmt.Sprintf("Policy validation failed: %v", err))
	}

	// Step 2: Create workload (Job/Pod)
	if err := r.createWorkload(ctx, session); err != nil {
		log.Error(err, "Failed to create workload")
		return r.updatePhase(ctx, session, PhaseFailed, fmt.Sprintf("Workload creation failed: %v", err))
	}

	// Step 3: Update phase to Running
	return r.updatePhase(ctx, session, PhaseRunning, "Workload created successfully")
}

// handleRunning handles sessions in Running phase
func (r *SessionReconciler) handleRunning(ctx context.Context, session *unstructured.Unstructured) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Check workload status
	workloadStatus, err := r.checkWorkloadStatus(ctx, session)
	if err != nil {
		log.Error(err, "Failed to check workload status")
		// Requeue to check again
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	switch workloadStatus {
	case "Completed":
		// Workload finished successfully
		return r.updatePhase(ctx, session, PhaseCompleted, "Session completed successfully")
	case "Failed":
		// Workload failed
		return r.updatePhase(ctx, session, PhaseFailed, "Workload execution failed")
	case "Running":
		// Still running, check again later
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	default:
		// Unknown status, check again
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}
}

// validatePolicy validates session against namespace policy
func (r *SessionReconciler) validatePolicy(ctx context.Context, session *unstructured.Unstructured) error {
	// Get namespace policy
	policy, err := r.DynamicClient.Resource(namespacePolicyGVR).
		Namespace(session.GetNamespace()).
		Get(ctx, "policy", metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// No policy means no restrictions
			return nil
		}
		return fmt.Errorf("failed to get namespace policy: %w", err)
	}

	// Validate session spec against policy
	sessionSpec, _ := session.Object["spec"].(map[string]interface{})
	policySpec, _ := policy.Object["spec"].(map[string]interface{})

	// Check budget constraints
	if err := r.validateBudget(sessionSpec, policySpec); err != nil {
		return err
	}

	// Check model constraints
	if err := r.validateModels(sessionSpec, policySpec); err != nil {
		return err
	}

	// Check tool constraints
	if err := r.validateTools(sessionSpec, policySpec); err != nil {
		return err
	}

	return nil
}

// validateBudget checks if session is within budget limits
func (r *SessionReconciler) validateBudget(sessionSpec, policySpec map[string]interface{}) error {
	// Get policy budget
	if models, ok := policySpec["models"].(map[string]interface{}); ok {
		if budget, ok := models["budget"].(map[string]interface{}); ok {
			// In a real implementation, we would:
			// 1. Query current month's usage
			// 2. Compare against budget limit
			// 3. Reject if would exceed
			// For now, we just log
			log.Log.Info("Budget validation would check usage against", "budget", budget)
		}
	}
	return nil
}

// validateModels checks if requested models are allowed
func (r *SessionReconciler) validateModels(sessionSpec, policySpec map[string]interface{}) error {
	sessionPolicy, ok := sessionSpec["policy"].(map[string]interface{})
	if !ok {
		return nil
	}

	modelConstraints, ok := sessionPolicy["modelConstraints"].(map[string]interface{})
	if !ok {
		return nil
	}

	requestedModels, ok := modelConstraints["allowed"].([]interface{})
	if !ok {
		return nil
	}

	// Get allowed models from policy
	if models, ok := policySpec["models"].(map[string]interface{}); ok {
		if allowed, ok := models["allowed"].([]interface{}); ok {
			for _, requested := range requestedModels {
				found := false
				for _, allowedModel := range allowed {
					if requested.(string) == allowedModel.(string) {
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("model %s is not allowed by namespace policy", requested)
				}
			}
		}
	}

	return nil
}

// validateTools checks if requested tools are allowed
func (r *SessionReconciler) validateTools(sessionSpec, policySpec map[string]interface{}) error {
	sessionPolicy, ok := sessionSpec["policy"].(map[string]interface{})
	if !ok {
		return nil
	}

	toolConstraints, ok := sessionPolicy["toolConstraints"].(map[string]interface{})
	if !ok {
		return nil
	}

	requestedTools, ok := toolConstraints["allowed"].([]interface{})
	if !ok {
		return nil
	}

	// Get allowed tools from policy
	if tools, ok := policySpec["tools"].(map[string]interface{}); ok {
		if allowed, ok := tools["allowed"].([]interface{}); ok {
			for _, requested := range requestedTools {
				found := false
				for _, allowedTool := range allowed {
					if requested.(string) == allowedTool.(string) {
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("tool %s is not allowed by namespace policy", requested)
				}
			}
		}
	}

	return nil
}

// createWorkload creates the runner workload for the session
func (r *SessionReconciler) createWorkload(ctx context.Context, session *unstructured.Unstructured) error {
	log := log.FromContext(ctx)

	// Get framework type from session spec
	spec, _ := session.Object["spec"].(map[string]interface{})
	framework, _ := spec["framework"].(map[string]interface{})
	frameworkType, _ := framework["type"].(string)

	// Create Job based on framework type
	job := r.buildJob(session, frameworkType)

	// Set owner reference
	if err := controllerutil.SetControllerReference(session, job, r.Scheme); err != nil {
		return fmt.Errorf("failed to set owner reference: %w", err)
	}

	// Create the job
	if err := r.Create(ctx, job); err != nil {
		if !errors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create job: %w", err)
		}
		log.Info("Job already exists", "name", job.GetName())
	}

	// Update session status with workload info
	r.updateWorkloadInfo(ctx, session, job.GetName())

	return nil
}

// buildJob creates a Job resource for the session
func (r *SessionReconciler) buildJob(session *unstructured.Unstructured, frameworkType string) *unstructured.Unstructured {
	jobName := fmt.Sprintf("%s-runner", session.GetName())

	// Select runner image based on framework type
	runnerImage := "ambient-platform/claude-code-runner:latest"
	switch frameworkType {
	case "claude-code":
		runnerImage = "ambient-platform/claude-code-runner:latest"
	case "custom-python":
		runnerImage = "ambient-platform/python-runner:latest"
	case "bash-runner":
		runnerImage = "ambient-platform/bash-runner:latest"
	}

	// Build Job specification
	job := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "batch/v1",
			"kind":       "Job",
			"metadata": map[string]interface{}{
				"name":      jobName,
				"namespace": session.GetNamespace(),
				"labels": map[string]interface{}{
					"ambient.ai/session-id":      session.GetName(),
					"ambient.ai/framework-type":  frameworkType,
					"app.kubernetes.io/component": "runner",
				},
			},
			"spec": map[string]interface{}{
				"backoffLimit": 3,
				"template": map[string]interface{}{
					"metadata": map[string]interface{}{
						"labels": map[string]interface{}{
							"ambient.ai/session-id":     session.GetName(),
							"ambient.ai/framework-type": frameworkType,
						},
					},
					"spec": map[string]interface{}{
						"restartPolicy": "OnFailure",
						"containers": []interface{}{
							map[string]interface{}{
								"name":  "runner",
								"image": runnerImage,
								"env": []interface{}{
									map[string]interface{}{
										"name":  "SESSION_NAME",
										"value": session.GetName(),
									},
									map[string]interface{}{
										"name":  "SESSION_NAMESPACE",
										"value": session.GetNamespace(),
									},
									map[string]interface{}{
										"name": "ANTHROPIC_API_KEY",
										"valueFrom": map[string]interface{}{
											"secretKeyRef": map[string]interface{}{
												"name": "runner-secrets",
												"key":  "anthropic-api-key",
											},
										},
									},
								},
								"resources": map[string]interface{}{
									"requests": map[string]interface{}{
										"memory": "512Mi",
										"cpu":    "500m",
									},
									"limits": map[string]interface{}{
										"memory": "2Gi",
										"cpu":    "2000m",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	return job
}

// checkWorkloadStatus checks the status of the session's workload
func (r *SessionReconciler) checkWorkloadStatus(ctx context.Context, session *unstructured.Unstructured) (string, error) {
	// Get workload info from session status
	status, _ := session.Object["status"].(map[string]interface{})
	workload, _ := status["workload"].(map[string]interface{})
	jobName, _ := workload["jobName"].(string)

	if jobName == "" {
		return "", fmt.Errorf("no job name in session status")
	}

	// Get the job
	job := &unstructured.Unstructured{}
	job.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "batch",
		Version: "v1",
		Kind:    "Job",
	})

	err := r.Get(ctx, client.ObjectKey{
		Namespace: session.GetNamespace(),
		Name:      jobName,
	}, job)
	if err != nil {
		return "", fmt.Errorf("failed to get job: %w", err)
	}

	// Check job status
	jobStatus, _ := job.Object["status"].(map[string]interface{})
	if jobStatus == nil {
		return "Running", nil
	}

	// Check for completion
	if succeeded, ok := jobStatus["succeeded"].(int64); ok && succeeded > 0 {
		return "Completed", nil
	}

	// Check for failure
	if failed, ok := jobStatus["failed"].(int64); ok && failed > 0 {
		return "Failed", nil
	}

	// Check if still active
	if active, ok := jobStatus["active"].(int64); ok && active > 0 {
		return "Running", nil
	}

	return "Unknown", nil
}

// updatePhase updates the session phase and adds to history
func (r *SessionReconciler) updatePhase(ctx context.Context, session *unstructured.Unstructured, phase, message string) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Get current status or create new one
	status, _ := session.Object["status"].(map[string]interface{})
	if status == nil {
		status = make(map[string]interface{})
	}

	// Update phase
	status["phase"] = phase

	// Add to history (append-only as per data-model.md line 77)
	history, _ := status["history"].([]interface{})
	if history == nil {
		history = make([]interface{}, 0)
	}

	historyEntry := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"event":     fmt.Sprintf("PhaseChanged%s", phase),
		"data": map[string]interface{}{
			"phase":   phase,
			"message": message,
		},
	}
	history = append(history, historyEntry)
	status["history"] = history

	// Update conditions
	conditions, _ := status["conditions"].([]interface{})
	if conditions == nil {
		conditions = make([]interface{}, 0)
	}

	// Add or update phase condition
	phaseCondition := map[string]interface{}{
		"type":               fmt.Sprintf("Phase%s", phase),
		"status":             "True",
		"lastTransitionTime": time.Now().UTC().Format(time.RFC3339),
		"reason":             phase,
		"message":            message,
	}
	conditions = append(conditions, phaseCondition)
	status["conditions"] = conditions

	// Update the session status
	session.Object["status"] = status

	if err := r.Status().Update(ctx, session); err != nil {
		log.Error(err, "Failed to update session status")
		return ctrl.Result{}, err
	}

	log.Info("Updated session phase", "phase", phase, "message", message)
	return ctrl.Result{}, nil
}

// updateWorkloadInfo updates the session status with workload information
func (r *SessionReconciler) updateWorkloadInfo(ctx context.Context, session *unstructured.Unstructured, jobName string) error {
	status, _ := session.Object["status"].(map[string]interface{})
	if status == nil {
		status = make(map[string]interface{})
	}

	status["workload"] = map[string]interface{}{
		"jobName":   jobName,
		"podName":   fmt.Sprintf("%s-pod", jobName), // Will be updated when pod is created
		"createdAt": time.Now().UTC().Format(time.RFC3339),
	}

	session.Object["status"] = status
	return r.Status().Update(ctx, session)
}

// getPhase gets the current phase from session status
func (r *SessionReconciler) getPhase(session *unstructured.Unstructured) string {
	status, _ := session.Object["status"].(map[string]interface{})
	if status == nil {
		return PhasePending
	}

	phase, _ := status["phase"].(string)
	if phase == "" {
		return PhasePending
	}

	return phase
}

// SetupWithManager sets up the controller with the Manager
func (r *SessionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&unstructured.Unstructured{}).
		Complete(r)
}

// +kubebuilder:rbac:groups=ambient.ai,resources=sessions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ambient.ai,resources=sessions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ambient.ai,resources=sessions/finalizers,verbs=update
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch