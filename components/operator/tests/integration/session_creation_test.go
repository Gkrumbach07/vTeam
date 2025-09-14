package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

func TestSessionCreation(t *testing.T) {
	// Skip if not in integration test environment
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()

	// Initialize Kubernetes clients (this will need actual cluster)
	k8sClient, dynamicClient := setupKubernetesClients(t)

	// Create test namespace
	testNamespace := "test-session-creation"
	createTestNamespace(t, k8sClient, testNamespace)
	defer cleanupTestNamespace(t, k8sClient, testNamespace)

	// Define Session GVR (Group Version Resource)
	sessionGVR := schema.GroupVersionResource{
		Group:    "ambient.ai",
		Version:  "v1alpha1",
		Resource: "sessions",
	}

	tests := []struct {
		name           string
		sessionSpec    map[string]interface{}
		expectCreation bool
		expectStatus   string
	}{
		{
			name: "create_valid_session",
			sessionSpec: map[string]interface{}{
				"trigger": map[string]interface{}{
					"source": "github",
					"event":  "pull_request",
					"payload": map[string]interface{}{
						"action": "opened",
					},
				},
				"framework": map[string]interface{}{
					"type":    "claude-code",
					"version": "1.0",
					"config":  map[string]interface{}{},
				},
				"policy": map[string]interface{}{
					"modelConstraints": map[string]interface{}{
						"allowed": []string{"claude-3-sonnet"},
						"budget":  "100.00",
					},
					"toolConstraints": map[string]interface{}{
						"allowed": []string{"bash", "edit", "read"},
					},
					"approvalRequired": false,
				},
			},
			expectCreation: true,
			expectStatus:   "Pending",
		},
		{
			name: "create_session_invalid_framework",
			sessionSpec: map[string]interface{}{
				"framework": map[string]interface{}{
					"type": "nonexistent-framework",
				},
			},
			expectCreation: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create Session object
			sessionName := "test-session-" + time.Now().Format("20060102-150405")
			session := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "ambient.ai/v1alpha1",
					"kind":       "Session",
					"metadata": map[string]interface{}{
						"name":      sessionName,
						"namespace": testNamespace,
					},
					"spec": tt.sessionSpec,
				},
			}

			// Create the session
			createdSession, err := dynamicClient.Resource(sessionGVR).
				Namespace(testNamespace).
				Create(ctx, session, metav1.CreateOptions{})

			if tt.expectCreation {
				require.NoError(t, err, "Session creation should succeed")
				assert.Equal(t, sessionName, createdSession.GetName())

				// Wait for controller to process (timeout after 30 seconds)
				timeout := time.After(30 * time.Second)
				ticker := time.NewTicker(1 * time.Second)
				defer ticker.Stop()

				var finalSession *unstructured.Unstructured
				for {
					select {
					case <-timeout:
						t.Fatal("Timeout waiting for session to be processed")
					case <-ticker.C:
						finalSession, err = dynamicClient.Resource(sessionGVR).
							Namespace(testNamespace).
							Get(ctx, sessionName, metav1.GetOptions{})
						require.NoError(t, err)

						// Check if status is set
						status, found, err := unstructured.NestedMap(finalSession.Object, "status")
						require.NoError(t, err)
						if found && len(status) > 0 {
							phase, exists, err := unstructured.NestedString(status, "phase")
							require.NoError(t, err)
							if exists {
								assert.Equal(t, tt.expectStatus, phase)
								return // Test passed
							}
						}
					}
				}
			} else {
				assert.Error(t, err, "Session creation should fail for invalid spec")
			}
		})
	}
}

// Helper functions that will be implemented with actual Kubernetes integration
func setupKubernetesClients(t *testing.T) (kubernetes.Interface, dynamic.Interface) {
	// This will fail until we implement proper cluster connection
	t.Skip("Kubernetes clients not implemented yet")
	return nil, nil
}

func createTestNamespace(t *testing.T, client kubernetes.Interface, namespace string) {
	// Implementation will be added when we have working clients
	t.Logf("Would create namespace: %s", namespace)
}

func cleanupTestNamespace(t *testing.T, client kubernetes.Interface, namespace string) {
	// Implementation will be added when we have working clients
	t.Logf("Would cleanup namespace: %s", namespace)
}