package services

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ArtifactIndexer manages artifact storage and retrieval
type ArtifactIndexer struct {
	kubeClient      kubernetes.Interface
	storageBackend  StorageBackend
	indexNamespace  string
}

// StorageBackend interface for different storage implementations
type StorageBackend interface {
	Store(ctx context.Context, key string, data io.Reader, metadata ArtifactMetadata) error
	Retrieve(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	List(ctx context.Context, prefix string) ([]string, error)
	GetMetadata(ctx context.Context, key string) (*ArtifactMetadata, error)
}

// NewArtifactIndexer creates a new artifact indexer
func NewArtifactIndexer(kubeClient kubernetes.Interface, storageBackend StorageBackend, indexNamespace string) *ArtifactIndexer {
	return &ArtifactIndexer{
		kubeClient:      kubeClient,
		storageBackend:  storageBackend,
		indexNamespace:  indexNamespace,
	}
}

// Artifact represents a stored artifact
type Artifact struct {
	ID           string            `json:"id"`
	SessionID    string            `json:"sessionId"`
	Namespace    string            `json:"namespace"`
	Type         string            `json:"type"`
	Name         string            `json:"name"`
	Size         int64             `json:"size"`
	ContentType  string            `json:"contentType"`
	CreatedAt    time.Time         `json:"createdAt"`
	UpdatedAt    time.Time         `json:"updatedAt"`
	Metadata     ArtifactMetadata  `json:"metadata"`
	StorageKey   string            `json:"storageKey"`
	DownloadURL  string            `json:"downloadUrl"`
	ViewURL      string            `json:"viewUrl"`
	Tags         []string          `json:"tags,omitempty"`
	Labels       map[string]string `json:"labels,omitempty"`
}

// ArtifactMetadata contains metadata about an artifact
type ArtifactMetadata struct {
	Tool        string            `json:"tool,omitempty"`
	ExitCode    *int              `json:"exitCode,omitempty"`
	Duration    string            `json:"duration,omitempty"`
	Description string            `json:"description,omitempty"`
	Source      string            `json:"source,omitempty"`
	Format      string            `json:"format,omitempty"`
	Encoding    string            `json:"encoding,omitempty"`
	Checksum    string            `json:"checksum,omitempty"`
	Custom      map[string]string `json:"custom,omitempty"`
}

// ArtifactListOptions contains options for listing artifacts
type ArtifactListOptions struct {
	SessionID   string
	Type        string
	Tool        string
	Tags        []string
	Limit       int
	Offset      int
	SortBy      string
	SortOrder   string
	CreatedAfter *time.Time
	CreatedBefore *time.Time
}

// StoreArtifact stores a new artifact
func (ai *ArtifactIndexer) StoreArtifact(ctx context.Context, sessionID, namespace string, name string, contentType string, data io.Reader, metadata ArtifactMetadata) (*Artifact, error) {
	// Generate artifact ID
	artifactID := ai.generateArtifactID(sessionID, name)

	// Generate storage key
	storageKey := ai.generateStorageKey(namespace, sessionID, artifactID)

	// Determine artifact type from content type
	artifactType := ai.inferArtifactType(contentType, name)

	// Store the artifact data
	err := ai.storageBackend.Store(ctx, storageKey, data, metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to store artifact data: %v", err)
	}

	// Create artifact record
	artifact := &Artifact{
		ID:          artifactID,
		SessionID:   sessionID,
		Namespace:   namespace,
		Type:        artifactType,
		Name:        name,
		ContentType: contentType,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Metadata:    metadata,
		StorageKey:  storageKey,
		DownloadURL: ai.generateDownloadURL(artifactID),
		ViewURL:     ai.generateViewURL(artifactID),
	}

	// Index the artifact in Kubernetes ConfigMap for fast lookup
	err = ai.indexArtifact(ctx, artifact)
	if err != nil {
		// If indexing fails, clean up the stored data
		ai.storageBackend.Delete(ctx, storageKey)
		return nil, fmt.Errorf("failed to index artifact: %v", err)
	}

	return artifact, nil
}

// GetArtifact retrieves an artifact by ID
func (ai *ArtifactIndexer) GetArtifact(ctx context.Context, artifactID string) (*Artifact, error) {
	// Get artifact metadata from index
	artifact, err := ai.getArtifactFromIndex(ctx, artifactID)
	if err != nil {
		return nil, fmt.Errorf("artifact not found: %v", err)
	}

	return artifact, nil
}

// GetArtifactData retrieves the actual data of an artifact
func (ai *ArtifactIndexer) GetArtifactData(ctx context.Context, artifactID string) (io.ReadCloser, error) {
	// Get artifact metadata
	artifact, err := ai.GetArtifact(ctx, artifactID)
	if err != nil {
		return nil, err
	}

	// Retrieve data from storage
	return ai.storageBackend.Retrieve(ctx, artifact.StorageKey)
}

// ListSessionArtifacts lists artifacts for a specific session
func (ai *ArtifactIndexer) ListSessionArtifacts(ctx context.Context, sessionID, namespace string, options ArtifactListOptions) ([]*Artifact, int, error) {
	// Get all artifacts for the session from index
	artifacts, err := ai.getSessionArtifactsFromIndex(ctx, sessionID, namespace)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get session artifacts: %v", err)
	}

	totalCount := len(artifacts)

	// Apply filters
	filteredArtifacts := ai.applyFilters(artifacts, options)

	// Apply sorting
	sortedArtifacts := ai.applySorting(filteredArtifacts, options)

	// Apply pagination
	paginatedArtifacts := ai.applyPagination(sortedArtifacts, options)

	return paginatedArtifacts, totalCount, nil
}

// DeleteArtifact deletes an artifact
func (ai *ArtifactIndexer) DeleteArtifact(ctx context.Context, artifactID string) error {
	// Get artifact metadata
	artifact, err := ai.GetArtifact(ctx, artifactID)
	if err != nil {
		return err
	}

	// Delete from storage
	err = ai.storageBackend.Delete(ctx, artifact.StorageKey)
	if err != nil {
		return fmt.Errorf("failed to delete artifact data: %v", err)
	}

	// Remove from index
	err = ai.removeFromIndex(ctx, artifactID)
	if err != nil {
		return fmt.Errorf("failed to remove from index: %v", err)
	}

	return nil
}

// CleanupSessionArtifacts removes all artifacts for a session
func (ai *ArtifactIndexer) CleanupSessionArtifacts(ctx context.Context, sessionID, namespace string) error {
	// Get all artifacts for the session
	artifacts, _, err := ai.ListSessionArtifacts(ctx, sessionID, namespace, ArtifactListOptions{})
	if err != nil {
		return err
	}

	// Delete each artifact
	for _, artifact := range artifacts {
		err := ai.DeleteArtifact(ctx, artifact.ID)
		if err != nil {
			// Log error but continue with cleanup
			fmt.Printf("Failed to delete artifact %s: %v\n", artifact.ID, err)
		}
	}

	return nil
}

// GetArtifactStats returns statistics about artifacts
func (ai *ArtifactIndexer) GetArtifactStats(ctx context.Context, namespace string) (*ArtifactStats, error) {
	// This would typically query an aggregated view
	// For now, we'll implement a basic version

	stats := &ArtifactStats{
		TotalArtifacts: 0,
		TotalSize:      0,
		TypeCounts:     make(map[string]int),
		ToolCounts:     make(map[string]int),
	}

	// Get all artifacts in the namespace (simplified implementation)
	// In production, this would be more efficient with aggregated indexes
	configMapName := ai.getIndexConfigMapName(namespace)
	configMap, err := ai.kubeClient.CoreV1().ConfigMaps(ai.indexNamespace).Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		return stats, nil // Return empty stats if no artifacts exist
	}

	for artifactID := range configMap.Data {
		stats.TotalArtifacts++
		// Parse artifact data to get type and size information
		// Implementation depends on how you serialize artifact metadata
		_ = artifactID // Use artifactID if needed for detailed stats
	}

	return stats, nil
}

// Helper methods

func (ai *ArtifactIndexer) generateArtifactID(sessionID, name string) string {
	timestamp := time.Now().Unix()
	return fmt.Sprintf("artifact-%s-%d", sessionID, timestamp)
}

func (ai *ArtifactIndexer) generateStorageKey(namespace, sessionID, artifactID string) string {
	return fmt.Sprintf("%s/%s/%s", namespace, sessionID, artifactID)
}

func (ai *ArtifactIndexer) generateDownloadURL(artifactID string) string {
	return fmt.Sprintf("/api/v1/artifacts/%s/download", artifactID)
}

func (ai *ArtifactIndexer) generateViewURL(artifactID string) string {
	return fmt.Sprintf("/api/v1/artifacts/%s/view", artifactID)
}

func (ai *ArtifactIndexer) inferArtifactType(contentType, name string) string {
	switch {
	case strings.HasPrefix(contentType, "image/"):
		return "image"
	case strings.HasPrefix(contentType, "video/"):
		return "video"
	case strings.HasPrefix(contentType, "audio/"):
		return "audio"
	case strings.HasPrefix(contentType, "text/"):
		return "text"
	case contentType == "application/json":
		return "json"
	case contentType == "application/pdf":
		return "pdf"
	case strings.HasSuffix(name, ".log"):
		return "log"
	case strings.HasSuffix(name, ".zip") || strings.HasSuffix(name, ".tar.gz"):
		return "archive"
	default:
		return "file"
	}
}

func (ai *ArtifactIndexer) indexArtifact(ctx context.Context, artifact *Artifact) error {
	// Store artifact metadata in a ConfigMap for indexing
	configMapName := ai.getIndexConfigMapName(artifact.Namespace)

	configMap, err := ai.kubeClient.CoreV1().ConfigMaps(ai.indexNamespace).Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		// Create new ConfigMap if it doesn't exist
		configMap = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      configMapName,
				Namespace: ai.indexNamespace,
				Labels: map[string]string{
					"ambient.ai/component": "artifact-index",
					"ambient.ai/namespace": artifact.Namespace,
				},
			},
			Data: make(map[string]string),
		}
	}

	// Serialize artifact metadata (simplified - use JSON)
	artifactJSON := fmt.Sprintf(`{
		"id": "%s",
		"sessionId": "%s",
		"namespace": "%s",
		"type": "%s",
		"name": "%s",
		"contentType": "%s",
		"createdAt": "%s",
		"storageKey": "%s"
	}`, artifact.ID, artifact.SessionID, artifact.Namespace, artifact.Type,
		artifact.Name, artifact.ContentType, artifact.CreatedAt.Format(time.RFC3339),
		artifact.StorageKey)

	configMap.Data[artifact.ID] = artifactJSON

	// Create or update ConfigMap
	if configMap.CreationTimestamp.IsZero() {
		_, err = ai.kubeClient.CoreV1().ConfigMaps(ai.indexNamespace).Create(ctx, configMap, metav1.CreateOptions{})
	} else {
		_, err = ai.kubeClient.CoreV1().ConfigMaps(ai.indexNamespace).Update(ctx, configMap, metav1.UpdateOptions{})
	}

	return err
}

func (ai *ArtifactIndexer) getArtifactFromIndex(ctx context.Context, artifactID string) (*Artifact, error) {
	// Search through ConfigMaps to find the artifact
	// This is inefficient for large numbers of artifacts, but works for demo purposes

	configMaps, err := ai.kubeClient.CoreV1().ConfigMaps(ai.indexNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: "ambient.ai/component=artifact-index",
	})
	if err != nil {
		return nil, err
	}

	for _, cm := range configMaps.Items {
		if artifactData, exists := cm.Data[artifactID]; exists {
			// Parse artifact data (simplified JSON parsing)
			artifact := &Artifact{}
			// In production, use proper JSON unmarshaling
			// For now, return a basic artifact structure
			artifact.ID = artifactID
			artifact.StorageKey = artifactData // This would be properly parsed
			return artifact, nil
		}
	}

	return nil, fmt.Errorf("artifact not found")
}

func (ai *ArtifactIndexer) getSessionArtifactsFromIndex(ctx context.Context, sessionID, namespace string) ([]*Artifact, error) {
	configMapName := ai.getIndexConfigMapName(namespace)

	configMap, err := ai.kubeClient.CoreV1().ConfigMaps(ai.indexNamespace).Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		return []*Artifact{}, nil // No artifacts found
	}

	var artifacts []*Artifact
	for artifactID := range configMap.Data {
		// Parse artifact data to check if it belongs to the session
		// Simplified implementation
		artifact := &Artifact{
			ID:        artifactID,
			SessionID: sessionID, // Would be parsed from actual data
		}
		artifacts = append(artifacts, artifact)
	}

	return artifacts, nil
}

func (ai *ArtifactIndexer) removeFromIndex(ctx context.Context, artifactID string) error {
	// Find and remove from the appropriate ConfigMap
	configMaps, err := ai.kubeClient.CoreV1().ConfigMaps(ai.indexNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: "ambient.ai/component=artifact-index",
	})
	if err != nil {
		return err
	}

	for _, cm := range configMaps.Items {
		if _, exists := cm.Data[artifactID]; exists {
			delete(cm.Data, artifactID)
			_, err = ai.kubeClient.CoreV1().ConfigMaps(ai.indexNamespace).Update(ctx, &cm, metav1.UpdateOptions{})
			return err
		}
	}

	return nil // Artifact not found in index
}

func (ai *ArtifactIndexer) getIndexConfigMapName(namespace string) string {
	return fmt.Sprintf("artifact-index-%s", namespace)
}

func (ai *ArtifactIndexer) applyFilters(artifacts []*Artifact, options ArtifactListOptions) []*Artifact {
	var filtered []*Artifact

	for _, artifact := range artifacts {
		if options.Type != "" && artifact.Type != options.Type {
			continue
		}
		if options.Tool != "" && artifact.Metadata.Tool != options.Tool {
			continue
		}
		if options.CreatedAfter != nil && artifact.CreatedAt.Before(*options.CreatedAfter) {
			continue
		}
		if options.CreatedBefore != nil && artifact.CreatedAt.After(*options.CreatedBefore) {
			continue
		}

		filtered = append(filtered, artifact)
	}

	return filtered
}

func (ai *ArtifactIndexer) applySorting(artifacts []*Artifact, options ArtifactListOptions) []*Artifact {
	// Implement sorting logic based on options.SortBy and options.SortOrder
	// For now, return as-is
	return artifacts
}

func (ai *ArtifactIndexer) applyPagination(artifacts []*Artifact, options ArtifactListOptions) []*Artifact {
	if options.Limit <= 0 {
		return artifacts
	}

	start := options.Offset
	if start >= len(artifacts) {
		return []*Artifact{}
	}

	end := start + options.Limit
	if end > len(artifacts) {
		end = len(artifacts)
	}

	return artifacts[start:end]
}

// ArtifactStats represents statistics about artifacts
type ArtifactStats struct {
	TotalArtifacts int               `json:"totalArtifacts"`
	TotalSize      int64             `json:"totalSize"`
	TypeCounts     map[string]int    `json:"typeCounts"`
	ToolCounts     map[string]int    `json:"toolCounts"`
}