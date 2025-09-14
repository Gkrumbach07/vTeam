package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// AdmissionRequest represents a Kubernetes admission request
type AdmissionRequest struct {
	UID       string          `json:"uid"`
	Kind      GroupVersionKind `json:"kind"`
	Resource  GroupVersionResource `json:"resource"`
	Name      string          `json:"name"`
	Namespace string          `json:"namespace"`
	Operation string          `json:"operation"`
	Object    json.RawMessage `json:"object"`
	OldObject json.RawMessage `json:"oldObject"`
}

// AdmissionResponse represents a Kubernetes admission response
type AdmissionResponse struct {
	UID     string `json:"uid"`
	Allowed bool   `json:"allowed"`
	Result  *Status `json:"result,omitempty"`
}

// AdmissionReview contains the request and response
type AdmissionReview struct {
	APIVersion string             `json:"apiVersion"`
	Kind       string             `json:"kind"`
	Request    *AdmissionRequest  `json:"request,omitempty"`
	Response   *AdmissionResponse `json:"response,omitempty"`
}

// GroupVersionKind represents a Kubernetes object type
type GroupVersionKind struct {
	Group   string `json:"group"`
	Version string `json:"version"`
	Kind    string `json:"kind"`
}

// GroupVersionResource represents a Kubernetes resource type
type GroupVersionResource struct {
	Group    string `json:"group"`
	Version  string `json:"version"`
	Resource string `json:"resource"`
}

// Status represents the result of an operation
type Status struct {
	Code    int32  `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/validate-session", handleSessionValidation)
	mux.HandleFunc("/validate-policy", handlePolicyValidation)
	mux.HandleFunc("/healthz", handleHealth)

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Printf("Starting webhook server on port %s", port)

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

func handleSessionValidation(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received session validation request")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v", err)
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}

	var admissionReview AdmissionReview
	if err := json.Unmarshal(body, &admissionReview); err != nil {
		log.Printf("Error unmarshaling admission review: %v", err)
		http.Error(w, "Error unmarshaling admission review", http.StatusBadRequest)
		return
	}

	response := validateSession(admissionReview.Request)

	admissionReview.Response = response
	admissionReview.APIVersion = "admission.k8s.io/v1"
	admissionReview.Kind = "AdmissionReview"

	responseBytes, err := json.Marshal(admissionReview)
	if err != nil {
		log.Printf("Error marshaling response: %v", err)
		http.Error(w, "Error marshaling response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(responseBytes)

	log.Printf("Session validation completed")
}

func handlePolicyValidation(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received policy validation request")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v", err)
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}

	var admissionReview AdmissionReview
	if err := json.Unmarshal(body, &admissionReview); err != nil {
		log.Printf("Error unmarshaling admission review: %v", err)
		http.Error(w, "Error unmarshaling admission review", http.StatusBadRequest)
		return
	}

	response := validatePolicy(admissionReview.Request)

	admissionReview.Response = response
	admissionReview.APIVersion = "admission.k8s.io/v1"
	admissionReview.Kind = "AdmissionReview"

	responseBytes, err := json.Marshal(admissionReview)
	if err != nil {
		log.Printf("Error marshaling response: %v", err)
		http.Error(w, "Error marshaling response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(responseBytes)

	log.Printf("Policy validation completed")
}

func validateSession(req *AdmissionRequest) *AdmissionResponse {
	log.Printf("Validating Session: %s/%s", req.Namespace, req.Name)

	// Parse the session object
	var sessionObj map[string]interface{}
	if err := json.Unmarshal(req.Object, &sessionObj); err != nil {
		log.Printf("Error parsing session object: %v", err)
		return &AdmissionResponse{
			UID:     req.UID,
			Allowed: false,
			Result: &Status{
				Code:    http.StatusBadRequest,
				Message: fmt.Sprintf("Error parsing session object: %v", err),
			},
		}
	}

	// Extract framework type from spec.framework.type
	spec, ok := sessionObj["spec"].(map[string]interface{})
	if !ok {
		return &AdmissionResponse{
			UID:     req.UID,
			Allowed: false,
			Result: &Status{
				Code:    http.StatusBadRequest,
				Message: "Session spec not found",
			},
		}
	}

	framework, ok := spec["framework"].(map[string]interface{})
	if !ok {
		return &AdmissionResponse{
			UID:     req.UID,
			Allowed: false,
			Result: &Status{
				Code:    http.StatusBadRequest,
				Message: "Framework configuration not found",
			},
		}
	}

	frameworkType, ok := framework["type"].(string)
	if !ok {
		return &AdmissionResponse{
			UID:     req.UID,
			Allowed: false,
			Result: &Status{
				Code:    http.StatusBadRequest,
				Message: "Framework type not specified",
			},
		}
	}

	// Validate framework type
	validFrameworks := map[string]bool{
		"claude-code": true,
		"generic":     true,
	}

	if !validFrameworks[frameworkType] {
		log.Printf("Invalid framework type: %s", frameworkType)
		return &AdmissionResponse{
			UID:     req.UID,
			Allowed: false,
			Result: &Status{
				Code:    http.StatusBadRequest,
				Message: fmt.Sprintf("Unsupported framework type: %s. Supported types: claude-code, generic", frameworkType),
			},
		}
	}

	log.Printf("Session validation passed for framework: %s", frameworkType)
	return &AdmissionResponse{
		UID:     req.UID,
		Allowed: true,
		Result: &Status{
			Code:    http.StatusOK,
			Message: "Session validation passed",
		},
	}
}

func validatePolicy(req *AdmissionRequest) *AdmissionResponse {
	log.Printf("Validating NamespacePolicy: %s/%s", req.Namespace, req.Name)

	// Parse the policy object
	var policyObj map[string]interface{}
	if err := json.Unmarshal(req.Object, &policyObj); err != nil {
		log.Printf("Error parsing policy object: %v", err)
		return &AdmissionResponse{
			UID:     req.UID,
			Allowed: false,
			Result: &Status{
				Code:    http.StatusBadRequest,
				Message: fmt.Sprintf("Error parsing policy object: %v", err),
			},
		}
	}

	// Basic validation - policy objects are generally allowed
	// More sophisticated validation can be added here

	log.Printf("NamespacePolicy validation passed")
	return &AdmissionResponse{
		UID:     req.UID,
		Allowed: true,
		Result: &Status{
			Code:    http.StatusOK,
			Message: "NamespacePolicy validation passed",
		},
	}
}