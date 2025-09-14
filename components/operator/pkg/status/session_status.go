package status

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// SessionPhase represents the current phase of a Session
type SessionPhase string

const (
	SessionPhasePending   SessionPhase = "Pending"
	SessionPhaseRunning   SessionPhase = "Running"
	SessionPhaseCompleted SessionPhase = "Completed"
	SessionPhaseFailed    SessionPhase = "Failed"
)

// ConditionType represents the type of a Session condition
type ConditionType string

const (
	ConditionPolicyValidated ConditionType = "PolicyValidated"
	ConditionWorkloadCreated ConditionType = "WorkloadCreated"
	ConditionWorkloadRunning ConditionType = "WorkloadRunning"
	ConditionArtifactsStored ConditionType = "ArtifactsStored"
)

// ConditionStatus represents the status of a condition
type ConditionStatus string

const (
	ConditionTrue    ConditionStatus = "True"
	ConditionFalse   ConditionStatus = "False"
	ConditionUnknown ConditionStatus = "Unknown"
)

// Condition represents a session condition
type Condition struct {
	Type               ConditionType   `json:"type"`
	Status             ConditionStatus `json:"status"`
	LastTransitionTime string          `json:"lastTransitionTime"`
	Reason             string          `json:"reason,omitempty"`
	Message            string          `json:"message,omitempty"`
}

// HistoryEvent represents an event in the session history
type HistoryEvent struct {
	Timestamp string                 `json:"timestamp"`
	Event     string                 `json:"event"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// SessionStatusUpdater manages the status updates for Session CRDs
type SessionStatusUpdater struct {
	client client.Client
}

// NewSessionStatusUpdater creates a new SessionStatusUpdater
func NewSessionStatusUpdater(client client.Client) *SessionStatusUpdater {
	return &SessionStatusUpdater{
		client: client,
	}
}

// UpdatePhase updates the phase of a Session
func (s *SessionStatusUpdater) UpdatePhase(ctx context.Context, session *unstructured.Unstructured, phase SessionPhase, reason, message string) error {
	logger := log.FromContext(ctx)

	// Update the phase
	if err := unstructured.SetNestedField(session.Object, string(phase), "status", "phase"); err != nil {
		return fmt.Errorf("failed to set phase: %w", err)
	}

	// Add history event
	event := HistoryEvent{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Event:     fmt.Sprintf("PhaseChanged:%s", phase),
		Data: map[string]interface{}{
			"phase":   string(phase),
			"reason":  reason,
			"message": message,
		},
	}

	if err := s.addHistoryEvent(session, event); err != nil {
		return fmt.Errorf("failed to add history event: %w", err)
	}

	// Update the Session in Kubernetes
	if err := s.client.Status().Update(ctx, session); err != nil {
		logger.Error(err, "Failed to update Session status", "name", session.GetName(), "namespace", session.GetNamespace())
		return fmt.Errorf("failed to update Session status: %w", err)
	}

	logger.Info("Updated Session phase", "name", session.GetName(), "namespace", session.GetNamespace(), "phase", phase)
	return nil
}

// UpdateCondition updates or adds a condition to a Session
func (s *SessionStatusUpdater) UpdateCondition(ctx context.Context, session *unstructured.Unstructured, conditionType ConditionType, status ConditionStatus, reason, message string) error {
	logger := log.FromContext(ctx)

	condition := Condition{
		Type:               conditionType,
		Status:             status,
		LastTransitionTime: time.Now().UTC().Format(time.RFC3339),
		Reason:             reason,
		Message:            message,
	}

	// Get existing conditions
	conditions, found, err := unstructured.NestedSlice(session.Object, "status", "conditions")
	if err != nil {
		return fmt.Errorf("failed to get conditions: %w", err)
	}

	if !found {
		conditions = []interface{}{}
	}

	// Find and update existing condition or add new one
	updated := false
	for i, cond := range conditions {
		condMap, ok := cond.(map[string]interface{})
		if !ok {
			continue
		}

		if condType, exists := condMap["type"]; exists && condType == string(conditionType) {
			// Update existing condition
			condMap["status"] = string(status)
			condMap["lastTransitionTime"] = condition.LastTransitionTime
			condMap["reason"] = reason
			condMap["message"] = message
			conditions[i] = condMap
			updated = true
			break
		}
	}

	if !updated {
		// Add new condition
		conditionMap := map[string]interface{}{
			"type":               string(condition.Type),
			"status":             string(condition.Status),
			"lastTransitionTime": condition.LastTransitionTime,
			"reason":             condition.Reason,
			"message":            condition.Message,
		}
		conditions = append(conditions, conditionMap)
	}

	// Set the updated conditions
	if err := unstructured.SetNestedSlice(session.Object, conditions, "status", "conditions"); err != nil {
		return fmt.Errorf("failed to set conditions: %w", err)
	}

	// Add history event
	event := HistoryEvent{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Event:     fmt.Sprintf("ConditionChanged:%s", conditionType),
		Data: map[string]interface{}{
			"type":    string(conditionType),
			"status":  string(status),
			"reason":  reason,
			"message": message,
		},
	}

	if err := s.addHistoryEvent(session, event); err != nil {
		return fmt.Errorf("failed to add history event: %w", err)
	}

	// Update the Session in Kubernetes
	if err := s.client.Status().Update(ctx, session); err != nil {
		logger.Error(err, "Failed to update Session condition", "name", session.GetName(), "namespace", session.GetNamespace())
		return fmt.Errorf("failed to update Session condition: %w", err)
	}

	logger.Info("Updated Session condition", "name", session.GetName(), "namespace", session.GetNamespace(),
		"conditionType", conditionType, "status", status)
	return nil
}

// TransitionToRunning transitions a Session from Pending to Running
func (s *SessionStatusUpdater) TransitionToRunning(ctx context.Context, session *unstructured.Unstructured, workloadID string) error {
	// Update conditions
	if err := s.UpdateCondition(ctx, session, ConditionPolicyValidated, ConditionTrue, "PolicyValidated", "Session policy has been validated"); err != nil {
		return err
	}

	if err := s.UpdateCondition(ctx, session, ConditionWorkloadCreated, ConditionTrue, "WorkloadCreated", fmt.Sprintf("Workload created with ID: %s", workloadID)); err != nil {
		return err
	}

	// Update phase
	return s.UpdatePhase(ctx, session, SessionPhaseRunning, "WorkloadStarted", fmt.Sprintf("Workload %s started", workloadID))
}

// TransitionToCompleted transitions a Session to Completed
func (s *SessionStatusUpdater) TransitionToCompleted(ctx context.Context, session *unstructured.Unstructured, artifacts []string) error {
	// Update artifacts condition
	message := fmt.Sprintf("Session completed successfully. Artifacts: %v", artifacts)
	if err := s.UpdateCondition(ctx, session, ConditionArtifactsStored, ConditionTrue, "ArtifactsStored", message); err != nil {
		return err
	}

	// Update phase
	return s.UpdatePhase(ctx, session, SessionPhaseCompleted, "SessionCompleted", "Session completed successfully")
}

// TransitionToFailed transitions a Session to Failed
func (s *SessionStatusUpdater) TransitionToFailed(ctx context.Context, session *unstructured.Unstructured, reason, message string) error {
	// Update relevant condition as false
	var conditionType ConditionType
	switch reason {
	case "PolicyViolation":
		conditionType = ConditionPolicyValidated
	case "WorkloadFailed":
		conditionType = ConditionWorkloadRunning
	default:
		conditionType = ConditionWorkloadRunning
	}

	if err := s.UpdateCondition(ctx, session, conditionType, ConditionFalse, reason, message); err != nil {
		return err
	}

	// Update phase
	return s.UpdatePhase(ctx, session, SessionPhaseFailed, reason, message)
}

// addHistoryEvent adds an event to the session history (append-only)
func (s *SessionStatusUpdater) addHistoryEvent(session *unstructured.Unstructured, event HistoryEvent) error {
	// Get existing history
	history, found, err := unstructured.NestedSlice(session.Object, "status", "history")
	if err != nil {
		return fmt.Errorf("failed to get history: %w", err)
	}

	if !found {
		history = []interface{}{}
	}

	// Convert event to map for unstructured
	eventMap := map[string]interface{}{
		"timestamp": event.Timestamp,
		"event":     event.Event,
	}

	if event.Data != nil && len(event.Data) > 0 {
		eventMap["data"] = event.Data
	}

	// Append new event (history is append-only)
	history = append(history, eventMap)

	// Set the updated history
	return unstructured.SetNestedSlice(session.Object, history, "status", "history")
}

// GetCurrentPhase returns the current phase of a Session
func (s *SessionStatusUpdater) GetCurrentPhase(session *unstructured.Unstructured) (SessionPhase, error) {
	phase, found, err := unstructured.NestedString(session.Object, "status", "phase")
	if err != nil {
		return "", fmt.Errorf("failed to get phase: %w", err)
	}

	if !found {
		return SessionPhasePending, nil // Default to Pending if not set
	}

	return SessionPhase(phase), nil
}

// GetCondition returns a specific condition from a Session
func (s *SessionStatusUpdater) GetCondition(session *unstructured.Unstructured, conditionType ConditionType) (*Condition, error) {
	conditions, found, err := unstructured.NestedSlice(session.Object, "status", "conditions")
	if err != nil {
		return nil, fmt.Errorf("failed to get conditions: %w", err)
	}

	if !found {
		return nil, nil // No conditions found
	}

	for _, cond := range conditions {
		condMap, ok := cond.(map[string]interface{})
		if !ok {
			continue
		}

		if condType, exists := condMap["type"]; exists && condType == string(conditionType) {
			condition := &Condition{
				Type: ConditionType(condType.(string)),
			}

			if status, exists := condMap["status"]; exists {
				condition.Status = ConditionStatus(status.(string))
			}

			if lastTransition, exists := condMap["lastTransitionTime"]; exists {
				condition.LastTransitionTime = lastTransition.(string)
			}

			if reason, exists := condMap["reason"]; exists {
				condition.Reason = reason.(string)
			}

			if message, exists := condMap["message"]; exists {
				condition.Message = message.(string)
			}

			return condition, nil
		}
	}

	return nil, nil // Condition not found
}

// IsPhaseTransitionValid validates if a phase transition is allowed
func (s *SessionStatusUpdater) IsPhaseTransitionValid(from, to SessionPhase) bool {
	validTransitions := map[SessionPhase][]SessionPhase{
		SessionPhasePending:   {SessionPhaseRunning, SessionPhaseFailed},
		SessionPhaseRunning:   {SessionPhaseCompleted, SessionPhaseFailed},
		SessionPhaseCompleted: {}, // Terminal state
		SessionPhaseFailed:    {}, // Terminal state
	}

	allowedTransitions, exists := validTransitions[from]
	if !exists {
		return false
	}

	for _, allowed := range allowedTransitions {
		if allowed == to {
			return true
		}
	}

	return false
}