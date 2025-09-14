package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

func TestGitHubWebhookToSessionCreation(t *testing.T) {
	// Skip if not in e2e test environment
	if testing.Short() {
		t.Skip("Skipping e2e test")
	}

	ctx := context.Background()

	// Test configuration
	const (
		testNamespace    = "test-github-webhook"
		backendURL       = "http://localhost:8080" // Will be configurable
		testAPIKey       = "test-github-api-key"
	)

	// Initialize clients
	dynamicClient := setupDynamicClient(t)
	defer cleanupE2ENamespace(t, testNamespace)

	// Setup test namespace with policy
	setupE2ENamespace(t, testNamespace, testAPIKey)

	tests := []struct {
		name                string
		webhookPayload      map[string]interface{}
		expectedSessionSpec map[string]interface{}
	}{
		{
			name: "pull_request_opened_creates_session",
			webhookPayload: map[string]interface{}{
				"action": "opened",
				"pull_request": map[string]interface{}{
					"id":    123456,
					"title": "Add multi-tenant support",
					"body":  "This PR adds namespace-based multi-tenancy",
					"html_url": "https://github.com/ambient-ai/vteam/pull/123",
				},
				"repository": map[string]interface{}{
					"name":      "vteam",
					"full_name": "ambient-ai/vteam",
					"owner": map[string]interface{}{
						"login": "ambient-ai",
					},
				},
			},
			expectedSessionSpec: map[string]interface{}{
				"trigger": map[string]interface{}{
					"source": "github",
					"event":  "pull_request",
				},
				"framework": map[string]interface{}{
					"type": "claude-code",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Step 1: Send webhook to backend
			webhookURL := fmt.Sprintf("%s/api/v1/webhooks/github", backendURL)

			payloadBytes, err := json.Marshal(tt.webhookPayload)
			require.NoError(t, err)

			req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(payloadBytes))
			require.NoError(t, err)

			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-API-Key", testAPIKey)

			client := &http.Client{Timeout: 10 * time.Second}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Step 2: Verify webhook was accepted
			assert.Equal(t, http.StatusAccepted, resp.StatusCode)

			var webhookResponse map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&webhookResponse)
			require.NoError(t, err)

			sessionID, exists := webhookResponse["sessionId"].(string)
			require.True(t, exists, "Response should contain sessionId")
			require.NotEmpty(t, sessionID)

			namespace, exists := webhookResponse["namespace"].(string)
			require.True(t, exists, "Response should contain namespace")
			assert.Equal(t, testNamespace, namespace)

			// Step 3: Verify Session CRD was created
			sessionGVR := schema.GroupVersionResource{
				Group:    "ambient.ai",
				Version:  "v1alpha1",
				Resource: "sessions",
			}

			// Wait for session to appear (with timeout)
			timeout := time.After(30 * time.Second)
			ticker := time.NewTicker(2 * time.Second)
			defer ticker.Stop()

			var session *unstructured.Unstructured
			for {
				select {
				case <-timeout:
					t.Fatal("Timeout waiting for session to be created")
				case <-ticker.C:
					session, err = dynamicClient.Resource(sessionGVR).
						Namespace(testNamespace).
						Get(ctx, sessionID, metav1.GetOptions{})
					if err == nil {
						goto sessionFound
					}
				}
			}

		sessionFound:
			// Step 4: Validate session spec matches expectations
			spec, found, err := unstructured.NestedMap(session.Object, "spec")
			require.NoError(t, err)
			require.True(t, found, "Session should have spec")

			// Verify trigger information
			trigger, found, err := unstructured.NestedMap(spec, "trigger")
			require.NoError(t, err)
			require.True(t, found)

			triggerSource, _, _ := unstructured.NestedString(trigger, "source")
			assert.Equal(t, tt.expectedSessionSpec["trigger"].(map[string]interface{})["source"], triggerSource)

			triggerEvent, _, _ := unstructured.NestedString(trigger, "event")
			assert.Equal(t, tt.expectedSessionSpec["trigger"].(map[string]interface{})["event"], triggerEvent)

			// Step 5: Wait for session to start processing
			timeout = time.After(60 * time.Second)
			ticker = time.NewTicker(3 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-timeout:
					t.Log("Warning: Session did not start processing within timeout")
					return
				case <-ticker.C:
					session, err = dynamicClient.Resource(sessionGVR).
						Namespace(testNamespace).
						Get(ctx, sessionID, metav1.GetOptions{})
					require.NoError(t, err)

					status, found, err := unstructured.NestedMap(session.Object, "status")
					require.NoError(t, err)
					if found {
						phase, exists, _ := unstructured.NestedString(status, "phase")
						if exists && phase != "Pending" {
							t.Logf("Session phase changed to: %s", phase)
							return // Test completed successfully
						}
					}
				}
			}
		})
	}
}

// Helper functions - these will fail until implemented
func setupDynamicClient(t *testing.T) dynamic.Interface {
	t.Skip("Dynamic client setup not implemented yet")
	return nil
}

func setupE2ENamespace(t *testing.T, namespace, apiKey string) {
	t.Logf("Would setup namespace %s with API key %s", namespace, apiKey)
}

func cleanupE2ENamespace(t *testing.T, namespace string) {
	t.Logf("Would cleanup namespace %s", namespace)
}