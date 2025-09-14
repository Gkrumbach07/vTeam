package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

var dynamicClient dynamic.Interface

func main() {
	// Create Kubernetes clients
	config, err := rest.InClusterConfig()
	if err != nil {
		fmt.Printf("Failed to create in-cluster config: %v\n", err)
		os.Exit(1)
	}

	dynamicClient, err = dynamic.NewForConfig(config)
	if err != nil {
		fmt.Printf("Failed to create dynamic client: %v\n", err)
		os.Exit(1)
	}

	// Set up HTTP handlers
	http.HandleFunc("/validate-session", handleSessionValidation)
	http.HandleFunc("/validate-policy", handlePolicyValidation)
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	fmt.Println("Starting webhook server on port 8080")
	server := &http.Server{
		Addr: ":8080",
	}

	if err := server.ListenAndServe(); err != nil {
		fmt.Printf("Failed to start webhook server: %v\n", err)
		os.Exit(1)
	}
}

func handleSessionValidation(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read request body: %v", err), http.StatusBadRequest)
		return
	}

	var admissionReview admissionv1.AdmissionReview
	if err := json.Unmarshal(body, &admissionReview); err != nil {
		http.Error(w, fmt.Sprintf("Failed to unmarshal admission review: %v", err), http.StatusBadRequest)
		return
	}

	allowed := true
	message := ""

	// Validate Session
	if admissionReview.Request != nil {
		session := &unstructured.Unstructured{}
		if err := json.Unmarshal(admissionReview.Request.Object.Raw, &session.Object); err == nil {
			if validationErr := validateSession(r.Context(), session); validationErr != nil {
				allowed = false
				message = validationErr.Error()
			}
		}
	}

	response := &admissionv1.AdmissionResponse{
		UID:     admissionReview.Request.UID,
		Allowed: allowed,
		Result: &metav1.Status{
			Message: message,
		},
	}

	admissionReview.Response = response
	respBytes, _ := json.Marshal(admissionReview)
	w.Header().Set("Content-Type", "application/json")
	w.Write(respBytes)
}

func handlePolicyValidation(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read request body: %v", err), http.StatusBadRequest)
		return
	}

	var admissionReview admissionv1.AdmissionReview
	if err := json.Unmarshal(body, &admissionReview); err != nil {
		http.Error(w, fmt.Sprintf("Failed to unmarshal admission review: %v", err), http.StatusBadRequest)
		return
	}

	allowed := true
	message := ""

	// Validate NamespacePolicy
	if admissionReview.Request != nil {
		policy := &unstructured.Unstructured{}
		if err := json.Unmarshal(admissionReview.Request.Object.Raw, &policy.Object); err == nil {
			if validationErr := validatePolicy(r.Context(), policy); validationErr != nil {
				allowed = false
				message = validationErr.Error()
			}
		}
	}

	response := &admissionv1.AdmissionResponse{
		UID:     admissionReview.Request.UID,
		Allowed: allowed,
		Result: &metav1.Status{
			Message: message,
		},
	}

	admissionReview.Response = response
	respBytes, _ := json.Marshal(admissionReview)
	w.Header().Set("Content-Type", "application/json")
	w.Write(respBytes)
}

func validateSession(ctx context.Context, session *unstructured.Unstructured) error {
	// Basic validation - check if framework type is supported
	spec, exists := session.Object["spec"].(map[string]interface{})
	if !exists {
		return fmt.Errorf("spec field is required")
	}

	framework, exists := spec["framework"].(map[string]interface{})
	if !exists {
		return fmt.Errorf("spec.framework field is required")
	}

	frameworkType, exists := framework["type"].(string)
	if !exists {
		return fmt.Errorf("spec.framework.type field is required")
	}

	validFrameworks := map[string]bool{
		"claude-code":     true,
		"custom-python":   true,
		"bash-runner":     true,
	}

	if !validFrameworks[frameworkType] {
		return fmt.Errorf("unsupported framework type '%s', supported: claude-code, custom-python, bash-runner", frameworkType)
	}

	return nil
}

func validatePolicy(ctx context.Context, policy *unstructured.Unstructured) error {
	// Basic validation - check if policy spec exists
	spec, exists := policy.Object["spec"].(map[string]interface{})
	if !exists {
		return fmt.Errorf("spec field is required")
	}

	// Validate models configuration if present
	if models, ok := spec["models"].(map[string]interface{}); ok {
		if err := validateModelsConfig(models); err != nil {
			return fmt.Errorf("invalid models configuration: %v", err)
		}
	}

	// Validate tools configuration if present
	if tools, ok := spec["tools"].(map[string]interface{}); ok {
		if err := validateToolsConfig(tools); err != nil {
			return fmt.Errorf("invalid tools configuration: %v", err)
		}
	}

	return nil
}

func validateModelsConfig(models map[string]interface{}) error {
	// Check for conflicting allowed/blocked lists
	allowed := getStringSlice(models, "allowed")
	blocked := getStringSlice(models, "blocked")

	for _, allowedModel := range allowed {
		for _, blockedModel := range blocked {
			if allowedModel == blockedModel {
				return fmt.Errorf("model '%s' cannot be both allowed and blocked", allowedModel)
			}
		}
	}

	return nil
}

func validateToolsConfig(tools map[string]interface{}) error {
	// Check for conflicting allowed/blocked lists
	allowed := getStringSlice(tools, "allowed")
	blocked := getStringSlice(tools, "blocked")

	for _, allowedTool := range allowed {
		for _, blockedTool := range blocked {
			if allowedTool == blockedTool {
				return fmt.Errorf("tool '%s' cannot be both allowed and blocked", allowedTool)
			}
		}
	}

	return nil
}

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