package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// SessionManager manages session lifecycle operations
type SessionManager struct {
	kubeClient    kubernetes.Interface
	dynamicClient dynamic.Interface
	namespace     string
}

// NewSessionManager creates a new session manager
func NewSessionManager(kubeClient kubernetes.Interface, dynamicClient dynamic.Interface, namespace string) *SessionManager {
	return &SessionManager{
		kubeClient:    kubeClient,
		dynamicClient: dynamicClient,
		namespace:     namespace,
	}
}

// SessionSpec represents the specification of a session
type SessionSpec struct {
	Trigger   TriggerInfo   `json:"trigger"`
	Framework FrameworkInfo `json:"framework"`
	Policy    PolicyInfo    `json:"policy,omitempty"`
}

// SessionStatus represents the status of a session
type SessionStatus struct {
	Phase       string            `json:"phase"`
	Message     string            `json:"message,omitempty"`
	StartTime   *time.Time        `json:"startTime,omitempty"`
	EndTime     *time.Time        `json:"endTime,omitempty"`
	Result      *SessionResult    `json:"result,omitempty"`
	Error       *SessionError     `json:"error,omitempty"`
	History     []StatusTransition `json:"history,omitempty"`
	Workload    *WorkloadInfo     `json:"workload,omitempty"`
}

// Session represents a complete session object
type Session struct {
	ID          string            `json:"sessionId"`
	Namespace   string            `json:"namespace"`
	Spec        SessionSpec       `json:"spec"`
	Status      SessionStatus     `json:"status"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// TriggerInfo contains information about what triggered the session
type TriggerInfo struct {
	Source  string                 `json:"source"`
	Event   string                 `json:"event"`
	Payload map[string]interface{} `json:"payload"`
}

// FrameworkInfo contains information about the execution framework
type FrameworkInfo struct {
	Type    string                 `json:"type"`
	Version string                 `json:"version"`
	Config  map[string]interface{} `json:"config,omitempty"`
}

// PolicyInfo contains policy constraints for the session
type PolicyInfo struct {
	ModelConstraints struct {
		Allowed []string `json:"allowed"`
		Budget  string   `json:"budget"`
	} `json:"modelConstraints"`
	ToolConstraints struct {
		Allowed []string `json:"allowed"`
		Blocked []string `json:"blocked,omitempty"`
	} `json:"toolConstraints"`
	ApprovalRequired bool `json:"approvalRequired,omitempty"`
}

// SessionResult contains the result of a completed session
type SessionResult struct {
	Summary     string   `json:"summary"`
	ArtifactIDs []string `json:"artifactIds,omitempty"`
	ExitCode    int      `json:"exitCode"`
	Duration    string   `json:"duration"`
}

// SessionError contains error information for failed sessions
type SessionError struct {
	Message string `json:"message"`
	Code    string `json:"code"`
	Details string `json:"details,omitempty"`
}

// StatusTransition represents a status change in the session
type StatusTransition struct {
	From      string    `json:"from"`
	To        string    `json:"to"`
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message,omitempty"`
}

// WorkloadInfo contains information about the running workload
type WorkloadInfo struct {
	PodName   string `json:"podName,omitempty"`
	JobName   string `json:"jobName,omitempty"`
	NodeName  string `json:"nodeName,omitempty"`
	StartTime string `json:"startTime,omitempty"`
}

// SessionListOptions contains options for listing sessions
type SessionListOptions struct {
	Status      string
	Framework   string
	Limit       int
	Offset      int
	SortBy      string
	SortOrder   string
	Labels      map[string]string
}

// CreateSession creates a new session in Kubernetes
func (sm *SessionManager) CreateSession(ctx context.Context, spec SessionSpec, namespace string) (*Session, error) {
	sessionGVR := schema.GroupVersionResource{
		Group:    "ambient.ai",
		Version:  "v1alpha1",
		Resource: "sessions",
	}

	// Generate unique session ID
	sessionID := sm.generateSessionID()

	// Create session object
	sessionObj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "ambient.ai/v1alpha1",
			"kind":       "Session",
			"metadata": map[string]interface{}{
				"name":      sessionID,
				"namespace": namespace,
				"labels": map[string]interface{}{
					"ambient.ai/framework": spec.Framework.Type,
					"ambient.ai/source":    spec.Trigger.Source,
				},
			},
			"spec": map[string]interface{}{
				"trigger":   spec.Trigger,
				"framework": spec.Framework,
			},
		},
	}

	// Add policy if specified
	if spec.Policy.ModelConstraints.Allowed != nil || spec.Policy.ToolConstraints.Allowed != nil {
		specMap := sessionObj.Object["spec"].(map[string]interface{})
		specMap["policy"] = spec.Policy
	}

	// Create the session in Kubernetes
	createdSession, err := sm.dynamicClient.Resource(sessionGVR).
		Namespace(namespace).
		Create(ctx, sessionObj, metav1.CreateOptions{})

	if err != nil {
		return nil, fmt.Errorf("failed to create session: %v", err)
	}

	return sm.convertToSession(createdSession)
}

// GetSession retrieves a session by ID
func (sm *SessionManager) GetSession(ctx context.Context, sessionID, namespace string) (*Session, error) {
	sessionGVR := schema.GroupVersionResource{
		Group:    "ambient.ai",
		Version:  "v1alpha1",
		Resource: "sessions",
	}

	sessionObj, err := sm.dynamicClient.Resource(sessionGVR).
		Namespace(namespace).
		Get(ctx, sessionID, metav1.GetOptions{})

	if err != nil {
		return nil, fmt.Errorf("failed to get session %s: %v", sessionID, err)
	}

	return sm.convertToSession(sessionObj)
}

// ListSessions lists sessions in a namespace with optional filtering
func (sm *SessionManager) ListSessions(ctx context.Context, namespace string, options SessionListOptions) ([]*Session, error) {
	sessionGVR := schema.GroupVersionResource{
		Group:    "ambient.ai",
		Version:  "v1alpha1",
		Resource: "sessions",
	}

	listOptions := metav1.ListOptions{}

	// Add label selector if specified
	if options.Labels != nil && len(options.Labels) > 0 {
		selector := ""
		for key, value := range options.Labels {
			if selector != "" {
				selector += ","
			}
			selector += fmt.Sprintf("%s=%s", key, value)
		}
		listOptions.LabelSelector = selector
	}

	sessionList, err := sm.dynamicClient.Resource(sessionGVR).
		Namespace(namespace).
		List(ctx, listOptions)

	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %v", err)
	}

	var sessions []*Session
	for _, item := range sessionList.Items {
		session, err := sm.convertToSession(&item)
		if err != nil {
			continue // Skip invalid sessions
		}

		// Apply filters
		if sm.sessionMatchesFilters(session, options) {
			sessions = append(sessions, session)
		}
	}

	// Apply pagination
	sessions = sm.applyPagination(sessions, options)

	// Apply sorting
	sessions = sm.applySorting(sessions, options)

	return sessions, nil
}

// UpdateSessionStatus updates the status of a session
func (sm *SessionManager) UpdateSessionStatus(ctx context.Context, sessionID, namespace string, status SessionStatus) error {
	sessionGVR := schema.GroupVersionResource{
		Group:    "ambient.ai",
		Version:  "v1alpha1",
		Resource: "sessions",
	}

	// Get current session
	sessionObj, err := sm.dynamicClient.Resource(sessionGVR).
		Namespace(namespace).
		Get(ctx, sessionID, metav1.GetOptions{})

	if err != nil {
		return fmt.Errorf("failed to get session for update: %v", err)
	}

	// Update status
	statusMap := map[string]interface{}{
		"phase":   status.Phase,
		"message": status.Message,
	}

	if status.StartTime != nil {
		statusMap["startTime"] = status.StartTime.Format(time.RFC3339)
	}

	if status.EndTime != nil {
		statusMap["endTime"] = status.EndTime.Format(time.RFC3339)
	}

	if status.Result != nil {
		statusMap["result"] = status.Result
	}

	if status.Error != nil {
		statusMap["error"] = status.Error
	}

	if status.Workload != nil {
		statusMap["workload"] = status.Workload
	}

	// Add history entry
	currentStatus, _, _ := unstructured.NestedMap(sessionObj.Object, "status")
	currentPhase, _, _ := unstructured.NestedString(currentStatus, "phase")

	if currentPhase != status.Phase {
		history := status.History
		if history == nil {
			history = []StatusTransition{}
		}

		transition := StatusTransition{
			From:      currentPhase,
			To:        status.Phase,
			Timestamp: time.Now(),
			Message:   status.Message,
		}

		history = append(history, transition)
		statusMap["history"] = history
	}

	// Update the session object
	unstructured.SetNestedMap(sessionObj.Object, statusMap, "status")

	// Update in Kubernetes
	_, err = sm.dynamicClient.Resource(sessionGVR).
		Namespace(namespace).
		UpdateStatus(ctx, sessionObj, metav1.UpdateOptions{})

	return err
}

// DeleteSession deletes a session
func (sm *SessionManager) DeleteSession(ctx context.Context, sessionID, namespace string) error {
	sessionGVR := schema.GroupVersionResource{
		Group:    "ambient.ai",
		Version:  "v1alpha1",
		Resource: "sessions",
	}

	return sm.dynamicClient.Resource(sessionGVR).
		Namespace(namespace).
		Delete(ctx, sessionID, metav1.DeleteOptions{})
}

// GetSessionsByStatus returns sessions filtered by status
func (sm *SessionManager) GetSessionsByStatus(ctx context.Context, namespace, status string) ([]*Session, error) {
	options := SessionListOptions{
		Status: status,
	}

	return sm.ListSessions(ctx, namespace, options)
}

// GetRunningSessionsCount returns the number of running sessions in a namespace
func (sm *SessionManager) GetRunningSessionsCount(ctx context.Context, namespace string) (int, error) {
	sessions, err := sm.GetSessionsByStatus(ctx, namespace, "Running")
	if err != nil {
		return 0, err
	}

	return len(sessions), nil
}

// Helper methods

func (sm *SessionManager) generateSessionID() string {
	// Generate a unique session ID
	// In production, use a proper UUID library
	timestamp := time.Now().Unix()
	return fmt.Sprintf("session-%d", timestamp)
}

func (sm *SessionManager) convertToSession(obj *unstructured.Unstructured) (*Session, error) {
	session := &Session{}

	// Basic metadata
	session.ID = obj.GetName()
	session.Namespace = obj.GetNamespace()
	session.CreatedAt = obj.GetCreationTimestamp().Time
	session.Labels = obj.GetLabels()
	session.Annotations = obj.GetAnnotations()

	// Parse spec
	spec, found, err := unstructured.NestedMap(obj.Object, "spec")
	if err != nil || !found {
		return nil, fmt.Errorf("session spec not found")
	}

	specBytes, err := json.Marshal(spec)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal spec: %v", err)
	}

	if err := json.Unmarshal(specBytes, &session.Spec); err != nil {
		return nil, fmt.Errorf("failed to unmarshal spec: %v", err)
	}

	// Parse status
	status, found, err := unstructured.NestedMap(obj.Object, "status")
	if err == nil && found {
		statusBytes, err := json.Marshal(status)
		if err == nil {
			json.Unmarshal(statusBytes, &session.Status)
		}
	}

	return session, nil
}

func (sm *SessionManager) sessionMatchesFilters(session *Session, options SessionListOptions) bool {
	// Filter by status
	if options.Status != "" && session.Status.Phase != options.Status {
		return false
	}

	// Filter by framework
	if options.Framework != "" && session.Spec.Framework.Type != options.Framework {
		return false
	}

	return true
}

func (sm *SessionManager) applyPagination(sessions []*Session, options SessionListOptions) []*Session {
	if options.Limit <= 0 {
		return sessions
	}

	start := options.Offset
	if start >= len(sessions) {
		return []*Session{}
	}

	end := start + options.Limit
	if end > len(sessions) {
		end = len(sessions)
	}

	return sessions[start:end]
}

func (sm *SessionManager) applySorting(sessions []*Session, options SessionListOptions) []*Session {
	// Implement sorting logic here
	// For now, return as-is (sorted by creation time by default)
	return sessions
}