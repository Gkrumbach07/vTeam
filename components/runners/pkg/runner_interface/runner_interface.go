package runner_interface

import (
	"context"
	"io"
	"time"
)

// RunnerInterface defines the contract that all agentic runners must implement
type RunnerInterface interface {
	// Initialize prepares the runner with configuration
	Initialize(ctx context.Context, config RunnerConfig) error

	// Execute runs the agentic session and returns the result
	Execute(ctx context.Context, request ExecutionRequest) (*ExecutionResult, error)

	// GetStatus returns the current status of the runner
	GetStatus() RunnerStatus

	// Stop gracefully stops the running session
	Stop(ctx context.Context) error

	// Cleanup releases any resources held by the runner
	Cleanup() error

	// GetLogs returns the logs from the runner
	GetLogs(ctx context.Context) (io.Reader, error)

	// GetArtifacts returns generated artifacts
	GetArtifacts(ctx context.Context) ([]Artifact, error)
}

// RunnerConfig contains the configuration for a runner instance
type RunnerConfig struct {
	// Runner identification
	RunnerID     string            `json:"runnerId"`
	SessionName  string            `json:"sessionName"`
	Namespace    string            `json:"namespace"`

	// Framework configuration
	FrameworkType    FrameworkType     `json:"frameworkType"`
	FrameworkVersion string            `json:"frameworkVersion"`
	FrameworkConfig  map[string]any    `json:"frameworkConfig"`

	// Execution environment
	Environment  map[string]string `json:"environment"`
	Resources    ResourceLimits    `json:"resources"`
	Timeout      time.Duration     `json:"timeout"`

	// Policy constraints
	Policy       PolicyConstraints `json:"policy"`

	// Backend integration
	BackendAPIURL string            `json:"backendApiUrl"`
	AuthToken     string            `json:"authToken"`

	// Storage configuration
	ArtifactStorage StorageConfig    `json:"artifactStorage"`
}

// ExecutionRequest contains the request for executing an agentic session
type ExecutionRequest struct {
	// Trigger information
	TriggerSource string         `json:"triggerSource"`
	TriggerEvent  string         `json:"triggerEvent"`
	TriggerPayload map[string]any `json:"triggerPayload"`

	// Execution parameters
	Parameters    map[string]any `json:"parameters"`
	Instructions  string         `json:"instructions,omitempty"`

	// Context and metadata
	RequestID     string         `json:"requestId"`
	StartTime     time.Time      `json:"startTime"`
}

// ExecutionResult contains the result of an agentic session execution
type ExecutionResult struct {
	// Execution status
	Status       ExecutionStatus `json:"status"`
	Message      string          `json:"message,omitempty"`

	// Timing information
	StartTime    time.Time       `json:"startTime"`
	EndTime      time.Time       `json:"endTime"`
	Duration     time.Duration   `json:"duration"`

	// Output and artifacts
	Output       string          `json:"output,omitempty"`
	Artifacts    []Artifact      `json:"artifacts"`

	// Resource usage
	ResourceUsage ResourceUsage   `json:"resourceUsage"`

	// Error information
	Error        string          `json:"error,omitempty"`
	ErrorCode    string          `json:"errorCode,omitempty"`

	// Metadata
	Metadata     map[string]any  `json:"metadata,omitempty"`
}

// FrameworkType represents the type of agentic framework
type FrameworkType string

const (
	FrameworkClaudeCode FrameworkType = "claude-code"
	FrameworkGeneric    FrameworkType = "generic"
	FrameworkCustom     FrameworkType = "custom"
)

// ExecutionStatus represents the status of execution
type ExecutionStatus string

const (
	StatusPending   ExecutionStatus = "pending"
	StatusRunning   ExecutionStatus = "running"
	StatusCompleted ExecutionStatus = "completed"
	StatusFailed    ExecutionStatus = "failed"
	StatusStopped   ExecutionStatus = "stopped"
	StatusTimeout   ExecutionStatus = "timeout"
)

// RunnerStatus represents the current status of a runner
type RunnerStatus struct {
	Status      ExecutionStatus `json:"status"`
	Message     string          `json:"message,omitempty"`
	Progress    float64         `json:"progress"` // 0.0 to 1.0
	StartTime   *time.Time      `json:"startTime,omitempty"`
	LastUpdate  time.Time       `json:"lastUpdate"`
}

// ResourceLimits defines resource constraints for execution
type ResourceLimits struct {
	CPUCores    float64 `json:"cpuCores"`
	MemoryMB    int64   `json:"memoryMb"`
	DiskMB      int64   `json:"diskMb,omitempty"`
	NetworkMbps int64   `json:"networkMbps,omitempty"`
}

// ResourceUsage tracks actual resource consumption
type ResourceUsage struct {
	CPUTime       time.Duration `json:"cpuTime"`
	MemoryPeakMB  int64         `json:"memoryPeakMb"`
	DiskUsedMB    int64         `json:"diskUsedMb"`
	NetworkBytesTX int64        `json:"networkBytesTx"`
	NetworkBytesRX int64        `json:"networkBytesRx"`
	APICallsCount  int64        `json:"apiCallsCount"`
	APICallsCost   float64      `json:"apiCallsCost"`
}

// PolicyConstraints defines policy-based execution constraints
type PolicyConstraints struct {
	// Model constraints
	AllowedModels    []string `json:"allowedModels"`
	BudgetLimit      float64  `json:"budgetLimit"`

	// Tool constraints
	AllowedTools     []string `json:"allowedTools"`
	BlockedTools     []string `json:"blockedTools"`

	// Network constraints
	AllowedDomains   []string `json:"allowedDomains,omitempty"`
	BlockedDomains   []string `json:"blockedDomains,omitempty"`
	NetworkAccess    bool     `json:"networkAccess"`

	// File system constraints
	ReadOnlyPaths    []string `json:"readOnlyPaths,omitempty"`
	WritablePaths    []string `json:"writablePaths,omitempty"`

	// Security constraints
	AllowPrivileged  bool     `json:"allowPrivileged"`
	ApprovalRequired bool     `json:"approvalRequired"`
}

// StorageConfig defines artifact storage configuration
type StorageConfig struct {
	Type       StorageType       `json:"type"`
	Location   string            `json:"location"`
	Credentials map[string]string `json:"credentials,omitempty"`
}

// StorageType represents the type of storage backend
type StorageType string

const (
	StorageS3       StorageType = "s3"
	StoragePVC      StorageType = "pvc"
	StorageExternal StorageType = "external"
	StorageLocal    StorageType = "local"
)

// Artifact represents a generated artifact from execution
type Artifact struct {
	Name        string            `json:"name"`
	Type        ArtifactType      `json:"type"`
	Size        int64             `json:"size"`
	Path        string            `json:"path"`
	URL         string            `json:"url,omitempty"`
	Checksum    string            `json:"checksum,omitempty"`
	Metadata    map[string]any    `json:"metadata,omitempty"`
	CreatedAt   time.Time         `json:"createdAt"`
}

// ArtifactType represents the type of artifact
type ArtifactType string

const (
	ArtifactText       ArtifactType = "text"
	ArtifactImage      ArtifactType = "image"
	ArtifactDocument   ArtifactType = "document"
	ArtifactCode       ArtifactType = "code"
	ArtifactData       ArtifactType = "data"
	ArtifactScreenshot ArtifactType = "screenshot"
	ArtifactReport     ArtifactType = "report"
	ArtifactLog        ArtifactType = "log"
)

// ProgressCallback is called to report execution progress
type ProgressCallback func(progress float64, message string)

// LogCallback is called to stream logs during execution
type LogCallback func(level LogLevel, message string, timestamp time.Time)

// LogLevel represents the severity level of a log message
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

// RunnerFactory creates runner instances based on framework type
type RunnerFactory interface {
	// CreateRunner creates a new runner instance for the specified framework
	CreateRunner(frameworkType FrameworkType) (RunnerInterface, error)

	// GetSupportedFrameworks returns the list of supported framework types
	GetSupportedFrameworks() []FrameworkType

	// ValidateConfig validates a runner configuration
	ValidateConfig(config RunnerConfig) error
}

// BaseRunner provides common functionality for runner implementations
type BaseRunner struct {
	config           RunnerConfig
	status           RunnerStatus
	progressCallback ProgressCallback
	logCallback      LogCallback
	artifacts        []Artifact
	resourceUsage    ResourceUsage
}

// NewBaseRunner creates a new base runner instance
func NewBaseRunner() *BaseRunner {
	return &BaseRunner{
		status: RunnerStatus{
			Status:     StatusPending,
			LastUpdate: time.Now(),
		},
		artifacts: make([]Artifact, 0),
	}
}

// SetProgressCallback sets the progress callback function
func (br *BaseRunner) SetProgressCallback(callback ProgressCallback) {
	br.progressCallback = callback
}

// SetLogCallback sets the log callback function
func (br *BaseRunner) SetLogCallback(callback LogCallback) {
	br.logCallback = callback
}

// UpdateStatus updates the runner status
func (br *BaseRunner) UpdateStatus(status ExecutionStatus, message string) {
	br.status.Status = status
	br.status.Message = message
	br.status.LastUpdate = time.Now()
}

// UpdateProgress updates the execution progress
func (br *BaseRunner) UpdateProgress(progress float64, message string) {
	br.status.Progress = progress
	br.status.Message = message
	br.status.LastUpdate = time.Now()

	if br.progressCallback != nil {
		br.progressCallback(progress, message)
	}
}

// LogMessage logs a message with the specified level
func (br *BaseRunner) LogMessage(level LogLevel, message string) {
	if br.logCallback != nil {
		br.logCallback(level, message, time.Now())
	}
}

// AddArtifact adds an artifact to the collection
func (br *BaseRunner) AddArtifact(artifact Artifact) {
	br.artifacts = append(br.artifacts, artifact)
}

// GetStatus returns the current status
func (br *BaseRunner) GetStatus() RunnerStatus {
	return br.status
}

// GetConfig returns the runner configuration
func (br *BaseRunner) GetConfig() RunnerConfig {
	return br.config
}

// SetConfig sets the runner configuration
func (br *BaseRunner) SetConfig(config RunnerConfig) {
	br.config = config
}

// IsValidFrameworkType checks if a framework type is valid
func IsValidFrameworkType(frameworkType FrameworkType) bool {
	switch frameworkType {
	case FrameworkClaudeCode, FrameworkGeneric, FrameworkCustom:
		return true
	default:
		return false
	}
}

// GetFrameworkTypes returns all valid framework types
func GetFrameworkTypes() []FrameworkType {
	return []FrameworkType{
		FrameworkClaudeCode,
		FrameworkGeneric,
		FrameworkCustom,
	}
}