package services

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// NamespaceResolver provides functionality to resolve namespaces based on various criteria
type NamespaceResolver struct {
	kubeClient    kubernetes.Interface
	dynamicClient dynamic.Interface
}

// NewNamespaceResolver creates a new namespace resolver
func NewNamespaceResolver(kubeClient kubernetes.Interface, dynamicClient dynamic.Interface) *NamespaceResolver {
	return &NamespaceResolver{
		kubeClient:    kubeClient,
		dynamicClient: dynamicClient,
	}
}

// NamespaceInfo contains information about a namespace
type NamespaceInfo struct {
	Name        string            `json:"name"`
	DisplayName string            `json:"displayName"`
	Description string            `json:"description"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	Status      string            `json:"status"`
	CreatedAt   string            `json:"createdAt"`
	Policy      *NamespacePolicy  `json:"policy,omitempty"`
}

// NamespacePolicy represents the policy associated with a namespace
type NamespacePolicy struct {
	Models      ModelConstraints      `json:"models"`
	Tools       ToolConstraints       `json:"tools"`
	Retention   RetentionPolicy       `json:"retention"`
	WebhookAuth map[string]string     `json:"webhookAuth"`
	Budget      BudgetInfo           `json:"budget"`
}

// ModelConstraints defines allowed models and budget constraints
type ModelConstraints struct {
	Allowed []string   `json:"allowed"`
	Blocked []string   `json:"blocked"`
	Budget  BudgetInfo `json:"budget"`
}

// ToolConstraints defines allowed and blocked tools
type ToolConstraints struct {
	Allowed []string `json:"allowed"`
	Blocked []string `json:"blocked"`
}

// RetentionPolicy defines data retention policies
type RetentionPolicy struct {
	Sessions   string `json:"sessions"`
	Artifacts  string `json:"artifacts"`
	AuditLogs  string `json:"auditLogs"`
}

// BudgetInfo contains budget information
type BudgetInfo struct {
	Monthly  string `json:"monthly"`
	Used     string `json:"used"`
	Currency string `json:"currency"`
}

// ResolveNamespaceFromAPIKey resolves namespace from API key
func (nr *NamespaceResolver) ResolveNamespaceFromAPIKey(ctx context.Context, apiKey string) (string, error) {
	// Query all NamespacePolicy resources to find matching API key
	namespacePolicyGVR := schema.GroupVersionResource{
		Group:    "ambient.ai",
		Version:  "v1alpha1",
		Resource: "namespacepolicies",
	}

	// List all NamespacePolicy resources across all namespaces
	policies, err := nr.dynamicClient.Resource(namespacePolicyGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list namespace policies: %v", err)
	}

	for _, policy := range policies.Items {
		webhookAuth, found, err := unstructured.NestedMap(policy.Object, "spec", "webhookAuth", "apiKeys")
		if err != nil || !found {
			continue
		}

		// Check if any of the API keys match
		for source, key := range webhookAuth {
			if keyStr, ok := key.(string); ok && keyStr == apiKey {
				// Found matching API key, return the namespace
				return policy.GetNamespace(), nil
			}
			_ = source // Use source if needed for logging
		}
	}

	return "", fmt.Errorf("namespace not found for API key")
}

// ResolveNamespaceFromWebhook resolves namespace from webhook payload
func (nr *NamespaceResolver) ResolveNamespaceFromWebhook(ctx context.Context, source string, payload map[string]interface{}) (string, error) {
	switch source {
	case "github":
		return nr.resolveGitHubNamespace(ctx, payload)
	case "jira":
		return nr.resolveJiraNamespace(ctx, payload)
	case "slack":
		return nr.resolveSlackNamespace(ctx, payload)
	default:
		return "", fmt.Errorf("unsupported webhook source: %s", source)
	}
}

// resolveGitHubNamespace resolves namespace for GitHub webhooks
func (nr *NamespaceResolver) resolveGitHubNamespace(ctx context.Context, payload map[string]interface{}) (string, error) {
	// Extract repository information
	repo, found := payload["repository"].(map[string]interface{})
	if !found {
		return "", fmt.Errorf("repository information not found in GitHub webhook")
	}

	owner, found := repo["owner"].(map[string]interface{})
	if !found {
		return "", fmt.Errorf("repository owner not found in GitHub webhook")
	}

	ownerLogin, ok := owner["login"].(string)
	if !ok {
		return "", fmt.Errorf("repository owner login not found")
	}

	repoName, ok := repo["name"].(string)
	if !ok {
		return "", fmt.Errorf("repository name not found")
	}

	// Look for namespace mapping based on repository
	namespace := nr.mapRepositoryToNamespace(ownerLogin, repoName)
	if namespace != "" {
		return namespace, nil
	}

	// Default mapping: convert owner to namespace-friendly format
	namespace = strings.ToLower(strings.ReplaceAll(ownerLogin, "-", ""))
	if !strings.HasPrefix(namespace, "team-") {
		namespace = "team-" + namespace
	}

	// Verify namespace exists
	if nr.namespaceExists(ctx, namespace) {
		return namespace, nil
	}

	return "", fmt.Errorf("unable to resolve namespace for GitHub repository %s/%s", ownerLogin, repoName)
}

// resolveJiraNamespace resolves namespace for Jira webhooks
func (nr *NamespaceResolver) resolveJiraNamespace(ctx context.Context, payload map[string]interface{}) (string, error) {
	// Extract project information
	issue, found := payload["issue"].(map[string]interface{})
	if !found {
		return "", fmt.Errorf("issue information not found in Jira webhook")
	}

	fields, found := issue["fields"].(map[string]interface{})
	if !found {
		return "", fmt.Errorf("issue fields not found in Jira webhook")
	}

	project, found := fields["project"].(map[string]interface{})
	if !found {
		return "", fmt.Errorf("project information not found in Jira webhook")
	}

	projectKey, ok := project["key"].(string)
	if !ok {
		return "", fmt.Errorf("project key not found")
	}

	// Map Jira project to namespace
	namespace := nr.mapJiraProjectToNamespace(projectKey)
	if namespace != "" && nr.namespaceExists(ctx, namespace) {
		return namespace, nil
	}

	return "", fmt.Errorf("unable to resolve namespace for Jira project %s", projectKey)
}

// resolveSlackNamespace resolves namespace for Slack webhooks
func (nr *NamespaceResolver) resolveSlackNamespace(ctx context.Context, payload map[string]interface{}) (string, error) {
	// Extract team and channel information
	teamID, ok := payload["team_id"].(string)
	if !ok {
		return "", fmt.Errorf("team ID not found in Slack webhook")
	}

	channel, found := payload["channel"].(map[string]interface{})
	if found {
		channelName, ok := channel["name"].(string)
		if ok {
			// Map Slack channel to namespace
			namespace := nr.mapSlackChannelToNamespace(teamID, channelName)
			if namespace != "" && nr.namespaceExists(ctx, namespace) {
				return namespace, nil
			}
		}
	}

	return "", fmt.Errorf("unable to resolve namespace for Slack team %s", teamID)
}

// GetNamespaceInfo retrieves detailed information about a namespace
func (nr *NamespaceResolver) GetNamespaceInfo(ctx context.Context, namespace string) (*NamespaceInfo, error) {
	// Get basic namespace information
	ns, err := nr.kubeClient.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get namespace %s: %v", namespace, err)
	}

	info := &NamespaceInfo{
		Name:        ns.Name,
		DisplayName: getDisplayName(ns),
		Description: getDescription(ns),
		Labels:      ns.Labels,
		Annotations: ns.Annotations,
		Status:      string(ns.Status.Phase),
		CreatedAt:   ns.CreationTimestamp.Format("2006-01-02T15:04:05Z"),
	}

	// Get associated policy
	policy, err := nr.getNamespacePolicy(ctx, namespace)
	if err == nil {
		info.Policy = policy
	}

	return info, nil
}

// ListUserNamespaces returns namespaces accessible to a user
func (nr *NamespaceResolver) ListUserNamespaces(ctx context.Context, username string, groups []string) ([]*NamespaceInfo, error) {
	// This is a simplified implementation
	// In production, you'd use Kubernetes RBAC to determine accessible namespaces

	var accessibleNamespaces []*NamespaceInfo

	// System admins can access all namespaces
	if nr.isSystemAdmin(groups) {
		return nr.getAllNamespaces(ctx)
	}

	// Regular users based on group membership
	for _, group := range groups {
		if strings.HasPrefix(group, "team-") {
			info, err := nr.GetNamespaceInfo(ctx, group)
			if err == nil {
				accessibleNamespaces = append(accessibleNamespaces, info)
			}
		}
	}

	return accessibleNamespaces, nil
}

// Helper methods

func (nr *NamespaceResolver) namespaceExists(ctx context.Context, namespace string) bool {
	_, err := nr.kubeClient.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	return err == nil
}

func (nr *NamespaceResolver) getNamespacePolicy(ctx context.Context, namespace string) (*NamespacePolicy, error) {
	namespacePolicyGVR := schema.GroupVersionResource{
		Group:    "ambient.ai",
		Version:  "v1alpha1",
		Resource: "namespacepolicies",
	}

	policy, err := nr.dynamicClient.Resource(namespacePolicyGVR).
		Namespace(namespace).
		Get(ctx, "policy", metav1.GetOptions{})

	if err != nil {
		return nil, err
	}

	return nr.parseNamespacePolicy(policy)
}

func (nr *NamespaceResolver) parseNamespacePolicy(policy *unstructured.Unstructured) (*NamespacePolicy, error) {
	spec, found, err := unstructured.NestedMap(policy.Object, "spec")
	if err != nil || !found {
		return nil, fmt.Errorf("policy spec not found")
	}

	result := &NamespacePolicy{}

	// Parse models
	if models, found, _ := unstructured.NestedMap(spec, "models"); found {
		if allowed, found, _ := unstructured.NestedStringSlice(models, "allowed"); found {
			result.Models.Allowed = allowed
		}
		if budget, found, _ := unstructured.NestedMap(models, "budget"); found {
			if monthly, ok, _ := unstructured.NestedString(budget, "monthly"); ok {
				result.Models.Budget.Monthly = monthly
			}
			if currency, ok, _ := unstructured.NestedString(budget, "currency"); ok {
				result.Models.Budget.Currency = currency
			}
		}
	}

	// Parse tools
	if tools, found, _ := unstructured.NestedMap(spec, "tools"); found {
		if allowed, found, _ := unstructured.NestedStringSlice(tools, "allowed"); found {
			result.Tools.Allowed = allowed
		}
		if blocked, found, _ := unstructured.NestedStringSlice(tools, "blocked"); found {
			result.Tools.Blocked = blocked
		}
	}

	return result, nil
}

func (nr *NamespaceResolver) getAllNamespaces(ctx context.Context) ([]*NamespaceInfo, error) {
	namespaces, err := nr.kubeClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var result []*NamespaceInfo
	for _, ns := range namespaces.Items {
		// Skip system namespaces for regular users
		if strings.HasPrefix(ns.Name, "kube-") || strings.HasPrefix(ns.Name, "openshift-") {
			continue
		}

		info, err := nr.GetNamespaceInfo(ctx, ns.Name)
		if err == nil {
			result = append(result, info)
		}
	}

	return result, nil
}

func (nr *NamespaceResolver) isSystemAdmin(groups []string) bool {
	for _, group := range groups {
		if group == "system:masters" || group == "admins" {
			return true
		}
	}
	return false
}

// Mapping functions - these would typically be configurable or stored in a database

func (nr *NamespaceResolver) mapRepositoryToNamespace(owner, repo string) string {
	// Example mappings
	mappings := map[string]string{
		"team-alpha/backend":   "team-alpha",
		"team-alpha/frontend":  "team-alpha",
		"team-beta/api":        "team-beta",
		"team-gamma/services":  "team-gamma",
	}

	key := fmt.Sprintf("%s/%s", owner, repo)
	return mappings[key]
}

func (nr *NamespaceResolver) mapJiraProjectToNamespace(projectKey string) string {
	mappings := map[string]string{
		"ALPHA": "team-alpha",
		"BETA":  "team-beta",
		"GAMMA": "team-gamma",
	}

	return mappings[projectKey]
}

func (nr *NamespaceResolver) mapSlackChannelToNamespace(teamID, channelName string) string {
	mappings := map[string]string{
		"dev-alpha":   "team-alpha",
		"dev-beta":    "team-beta",
		"dev-gamma":   "team-gamma",
		"engineering": "team-alpha", // Default
	}

	return mappings[channelName]
}

func getDisplayName(ns *corev1.Namespace) string {
	if displayName, exists := ns.Annotations["ambient.ai/display-name"]; exists {
		return displayName
	}
	return ns.Name
}

func getDescription(ns *corev1.Namespace) string {
	if description, exists := ns.Annotations["ambient.ai/description"]; exists {
		return description
	}
	return ""
}