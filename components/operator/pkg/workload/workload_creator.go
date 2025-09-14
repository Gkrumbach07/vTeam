package workload

import (
	"context"
	"fmt"
	"strings"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// FrameworkType represents the type of framework to run
type FrameworkType string

const (
	FrameworkClaudeCode FrameworkType = "claude-code"
	FrameworkGeneric    FrameworkType = "generic"
)

// WorkloadConfig represents the configuration for a workload
type WorkloadConfig struct {
	FrameworkType FrameworkType          `json:"frameworkType"`
	Image         string                 `json:"image"`
	Command       []string               `json:"command"`
	Args          []string               `json:"args"`
	Env           map[string]string      `json:"env"`
	Resources     corev1.ResourceList    `json:"resources"`
	Timeout       int32                  `json:"timeoutSeconds"`
	Config        map[string]interface{} `json:"config"`
}

// WorkloadCreator manages the creation of workloads for Sessions
type WorkloadCreator struct {
	client            client.Client
	defaultResources  corev1.ResourceList
	defaultTimeout    int32
	backendAPIURL     string
}

// NewWorkloadCreator creates a new WorkloadCreator
func NewWorkloadCreator(client client.Client, backendAPIURL string) *WorkloadCreator {
	return &WorkloadCreator{
		client: client,
		defaultResources: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("500m"),
			corev1.ResourceMemory: resource.MustParse("1Gi"),
		},
		defaultTimeout:    300, // 5 minutes
		backendAPIURL:     backendAPIURL,
	}
}

// CreateWorkloadForSession creates a Kubernetes Job for a Session
func (w *WorkloadCreator) CreateWorkloadForSession(ctx context.Context, session *unstructured.Unstructured) (*batchv1.Job, error) {
	logger := log.FromContext(ctx)

	sessionName := session.GetName()
	namespace := session.GetNamespace()

	logger.Info("Creating workload for session", "name", sessionName, "namespace", namespace)

	// Extract framework configuration from session
	workloadConfig, err := w.extractWorkloadConfig(session)
	if err != nil {
		return nil, fmt.Errorf("failed to extract workload config: %w", err)
	}

	// Create the Job
	job := w.buildJob(session, workloadConfig)

	// Create the Job in Kubernetes
	if err := w.client.Create(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to create Job: %w", err)
	}

	logger.Info("Created workload Job", "name", job.GetName(), "namespace", job.GetNamespace())
	return job, nil
}

// extractWorkloadConfig extracts workload configuration from a Session
func (w *WorkloadCreator) extractWorkloadConfig(session *unstructured.Unstructured) (*WorkloadConfig, error) {
	config := &WorkloadConfig{
		Resources: w.defaultResources,
		Timeout:   w.defaultTimeout,
		Env:       make(map[string]string),
	}

	// Extract framework type
	frameworkType, found, err := unstructured.NestedString(session.Object, "spec", "framework", "type")
	if err != nil {
		return nil, fmt.Errorf("failed to get framework type: %w", err)
	}
	if !found || frameworkType == "" {
		return nil, fmt.Errorf("framework type not specified")
	}

	config.FrameworkType = FrameworkType(frameworkType)

	// Extract framework version
	frameworkVersion, _, err := unstructured.NestedString(session.Object, "spec", "framework", "version")
	if err != nil {
		return nil, fmt.Errorf("failed to get framework version: %w", err)
	}
	if frameworkVersion == "" {
		frameworkVersion = "latest"
	}

	// Extract framework config
	frameworkConfig, found, err := unstructured.NestedMap(session.Object, "spec", "framework", "config")
	if err != nil {
		return nil, fmt.Errorf("failed to get framework config: %w", err)
	}
	if found {
		config.Config = frameworkConfig
	}

	// Configure based on framework type
	switch config.FrameworkType {
	case FrameworkClaudeCode:
		w.configureClaudeCodeWorkload(config, frameworkVersion)
	case FrameworkGeneric:
		w.configureGenericWorkload(config, frameworkVersion)
	default:
		return nil, fmt.Errorf("unsupported framework type: %s", config.FrameworkType)
	}

	// Extract trigger information for environment variables
	triggerSource, _, _ := unstructured.NestedString(session.Object, "spec", "trigger", "source")
	triggerEvent, _, _ := unstructured.NestedString(session.Object, "spec", "trigger", "event")

	// Set common environment variables
	config.Env["SESSION_NAME"] = session.GetName()
	config.Env["SESSION_NAMESPACE"] = session.GetNamespace()
	config.Env["BACKEND_API_URL"] = w.backendAPIURL
	config.Env["TRIGGER_SOURCE"] = triggerSource
	config.Env["TRIGGER_EVENT"] = triggerEvent

	// Add framework-specific config as environment variables
	for key, value := range config.Config {
		if strValue, ok := value.(string); ok {
			envKey := fmt.Sprintf("CONFIG_%s", strings.ToUpper(key))
			config.Env[envKey] = strValue
		}
	}

	return config, nil
}

// configureClaudeCodeWorkload configures a Claude Code workload
func (w *WorkloadCreator) configureClaudeCodeWorkload(config *WorkloadConfig, version string) {
	config.Image = fmt.Sprintf("localhost/claude-code-runner:%s", version)
	config.Command = []string{"python3", "/app/main.py"}

	// Claude Code specific resources
	config.Resources = corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse("1000m"),
		corev1.ResourceMemory: resource.MustParse("2Gi"),
	}
	config.Timeout = 300 // 5 minutes

	// Add Claude Code specific environment variables
	config.Env["FRAMEWORK_TYPE"] = string(FrameworkClaudeCode)

	// Add model configuration if specified
	if model, exists := config.Config["model"]; exists {
		if modelStr, ok := model.(string); ok {
			config.Env["MODEL"] = modelStr
		}
	}
}

// configureGenericWorkload configures a generic framework workload
func (w *WorkloadCreator) configureGenericWorkload(config *WorkloadConfig, version string) {
	config.Image = fmt.Sprintf("localhost/generic-runner:%s", version)
	config.Command = []string{"/app/runner"}

	// Generic framework resources
	config.Resources = w.defaultResources
	config.Timeout = w.defaultTimeout

	// Add generic framework environment variables
	config.Env["FRAMEWORK_TYPE"] = string(FrameworkGeneric)
}

// buildJob creates a Kubernetes Job for the workload
func (w *WorkloadCreator) buildJob(session *unstructured.Unstructured, config *WorkloadConfig) *batchv1.Job {
	sessionName := session.GetName()
	namespace := session.GetNamespace()

	// Generate job name (session name + suffix for uniqueness)
	jobName := fmt.Sprintf("%s-workload", sessionName)

	// Convert environment variables to container env vars
	var envVars []corev1.EnvVar
	for key, value := range config.Env {
		envVars = append(envVars, corev1.EnvVar{
			Name:  key,
			Value: value,
		})
	}

	// Add ANTHROPIC_API_KEY from secret
	envVars = append(envVars, corev1.EnvVar{
		Name: "ANTHROPIC_API_KEY",
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "anthropic-api-key",
				},
				Key: "api-key",
			},
		},
	})

	// Build the Job spec
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: namespace,
			Labels: map[string]string{
				"app":          "session-workload",
				"session":      sessionName,
				"framework":    string(config.FrameworkType),
				"ambient.ai/session": sessionName,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: session.GetAPIVersion(),
					Kind:       session.GetKind(),
					Name:       session.GetName(),
					UID:        session.GetUID(),
					Controller: &[]bool{true}[0],
				},
			},
		},
		Spec: batchv1.JobSpec{
			TTLSecondsAfterFinished: &[]int32{3600}[0], // Clean up after 1 hour
			ActiveDeadlineSeconds:   &config.Timeout,
			BackoffLimit:           &[]int32{0}[0], // No retries
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":          "session-workload",
						"session":      sessionName,
						"framework":    string(config.FrameworkType),
						"ambient.ai/session": sessionName,
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: &[]bool{true}[0],
						RunAsUser:    &[]int64{1000}[0],
						FSGroup:      &[]int64{1000}[0],
					},
					Containers: []corev1.Container{
						{
							Name:  "session-runner",
							Image: config.Image,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Command: config.Command,
							Args:    config.Args,
							Env:     envVars,
							Resources: corev1.ResourceRequirements{
								Requests: config.Resources,
								Limits:   config.Resources,
							},
							SecurityContext: &corev1.SecurityContext{
								AllowPrivilegeEscalation: &[]bool{false}[0],
								ReadOnlyRootFilesystem:   &[]bool{false}[0], // Claude Code needs write access
								RunAsNonRoot:            &[]bool{true}[0],
								RunAsUser:               &[]int64{1000}[0],
								Capabilities: &corev1.Capabilities{
									Drop: []corev1.Capability{"ALL"},
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

// GetWorkloadForSession retrieves the workload Job for a Session
func (w *WorkloadCreator) GetWorkloadForSession(ctx context.Context, session *unstructured.Unstructured) (*batchv1.Job, error) {
	sessionName := session.GetName()
	namespace := session.GetNamespace()
	jobName := fmt.Sprintf("%s-workload", sessionName)

	job := &batchv1.Job{}
	if err := w.client.Get(ctx, client.ObjectKey{Name: jobName, Namespace: namespace}, job); err != nil {
		return nil, err
	}

	return job, nil
}

// DeleteWorkloadForSession deletes the workload Job for a Session
func (w *WorkloadCreator) DeleteWorkloadForSession(ctx context.Context, session *unstructured.Unstructured) error {
	logger := log.FromContext(ctx)

	job, err := w.GetWorkloadForSession(ctx, session)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			return nil // Job already deleted
		}
		return fmt.Errorf("failed to get workload Job: %w", err)
	}

	if err := w.client.Delete(ctx, job); err != nil {
		return fmt.Errorf("failed to delete workload Job: %w", err)
	}

	logger.Info("Deleted workload Job", "name", job.GetName(), "namespace", job.GetNamespace())
	return nil
}

// GetWorkloadStatus returns the status of a workload Job
func (w *WorkloadCreator) GetWorkloadStatus(ctx context.Context, session *unstructured.Unstructured) (WorkloadStatus, error) {
	job, err := w.GetWorkloadForSession(ctx, session)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			return WorkloadStatusNotFound, nil
		}
		return WorkloadStatusUnknown, fmt.Errorf("failed to get workload Job: %w", err)
	}

	// Check job conditions
	for _, condition := range job.Status.Conditions {
		switch condition.Type {
		case batchv1.JobComplete:
			if condition.Status == corev1.ConditionTrue {
				return WorkloadStatusCompleted, nil
			}
		case batchv1.JobFailed:
			if condition.Status == corev1.ConditionTrue {
				return WorkloadStatusFailed, nil
			}
		}
	}

	// Check if job has active pods
	if job.Status.Active > 0 {
		return WorkloadStatusRunning, nil
	}

	// Default to pending if no clear status
	return WorkloadStatusPending, nil
}

// WorkloadStatus represents the status of a workload
type WorkloadStatus string

const (
	WorkloadStatusPending   WorkloadStatus = "Pending"
	WorkloadStatusRunning   WorkloadStatus = "Running"
	WorkloadStatusCompleted WorkloadStatus = "Completed"
	WorkloadStatusFailed    WorkloadStatus = "Failed"
	WorkloadStatusNotFound  WorkloadStatus = "NotFound"
	WorkloadStatusUnknown   WorkloadStatus = "Unknown"
)

// IsValidFrameworkType checks if a framework type is supported
func IsValidFrameworkType(frameworkType string) bool {
	switch FrameworkType(frameworkType) {
	case FrameworkClaudeCode, FrameworkGeneric:
		return true
	default:
		return false
	}
}

// GetSupportedFrameworkTypes returns a list of supported framework types
func GetSupportedFrameworkTypes() []string {
	return []string{
		string(FrameworkClaudeCode),
		string(FrameworkGeneric),
	}
}