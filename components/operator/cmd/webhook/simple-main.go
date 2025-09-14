package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func main() {
	// Simple HTTP server for webhook validation
	certFile := os.Getenv("TLS_CERT_FILE")
	keyFile := os.Getenv("TLS_KEY_FILE")
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

	if certFile != "" && keyFile != "" {
		// HTTPS mode
		log.Printf("Starting HTTPS server with cert=%s, key=%s", certFile, keyFile)
		if err := server.ListenAndServeTLS(certFile, keyFile); err != nil {
			log.Fatalf("Failed to start HTTPS server: %v", err)
		}
	} else {
		// HTTP mode for testing
		log.Printf("Starting HTTP server (no TLS)")
		if err := server.ListenAndServe(); err != nil {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
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

	var admissionReview admissionv1.AdmissionReview
	if err := json.Unmarshal(body, &admissionReview); err != nil {
		log.Printf("Error unmarshaling admission review: %v", err)
		http.Error(w, "Error unmarshaling admission review", http.StatusBadRequest)
		return
	}

	response := validateSession(admissionReview.Request)

	admissionReview.Response = response
	admissionReview.Response.UID = admissionReview.Request.UID

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

	var admissionReview admissionv1.AdmissionReview
	if err := json.Unmarshal(body, &admissionReview); err != nil {
		log.Printf("Error unmarshaling admission review: %v", err)
		http.Error(w, "Error unmarshaling admission review", http.StatusBadRequest)
		return
	}

	response := validatePolicy(admissionReview.Request)

	admissionReview.Response = response
	admissionReview.Response.UID = admissionReview.Request.UID

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

func validateSession(req *admissionv1.AdmissionRequest) *admissionv1.AdmissionResponse {
	log.Printf("Validating Session: %s/%s", req.Namespace, req.Name)

	// Parse the session object
	var sessionObj map[string]interface{}
	if err := json.Unmarshal(req.Object.Raw, &sessionObj); err != nil {
		log.Printf("Error parsing session object: %v", err)
		return &admissionv1.AdmissionResponse{
			Result: &metav1.Status{
				Code:    http.StatusBadRequest,
				Message: fmt.Sprintf("Error parsing session object: %v", err),
			},
			Allowed: false,
		}
	}

	// Extract framework type from spec.framework.type
	spec, ok := sessionObj["spec"].(map[string]interface{})
	if !ok {
		return &admissionv1.AdmissionResponse{
			Result: &metav1.Status{
				Code:    http.StatusBadRequest,
				Message: "Session spec not found",
			},
			Allowed: false,
		}
	}

	framework, ok := spec["framework"].(map[string]interface{})
	if !ok {
		return &admissionv1.AdmissionResponse{
			Result: &metav1.Status{
				Code:    http.StatusBadRequest,
				Message: "Framework configuration not found",
			},
			Allowed: false,
		}
	}

	frameworkType, ok := framework["type"].(string)
	if !ok {
		return &admissionv1.AdmissionResponse{
			Result: &metav1.Status{
				Code:    http.StatusBadRequest,
				Message: "Framework type not specified",
			},
			Allowed: false,
		}
	}

	// Validate framework type
	validFrameworks := map[string]bool{
		"claude-code": true,
		"generic":     true,
	}

	if !validFrameworks[frameworkType] {
		log.Printf("Invalid framework type: %s", frameworkType)
		return &admissionv1.AdmissionResponse{
			Result: &metav1.Status{
				Code:    http.StatusBadRequest,
				Message: fmt.Sprintf("Unsupported framework type: %s. Supported types: claude-code, generic", frameworkType),
			},
			Allowed: false,
		}
	}

	// Additional validations can be added here
	// For now, just validate framework type

	log.Printf("Session validation passed for framework: %s", frameworkType)
	return &admissionv1.AdmissionResponse{
		Result: &metav1.Status{
			Code:    http.StatusOK,
			Message: "Session validation passed",
		},
		Allowed: true,
	}
}

func validatePolicy(req *admissionv1.AdmissionRequest) *admissionv1.AdmissionResponse {
	log.Printf("Validating NamespacePolicy: %s/%s", req.Namespace, req.Name)

	// Parse the policy object
	var policyObj map[string]interface{}
	if err := json.Unmarshal(req.Object.Raw, &policyObj); err != nil {
		log.Printf("Error parsing policy object: %v", err)
		return &admissionv1.AdmissionResponse{
			Result: &metav1.Status{
				Code:    http.StatusBadRequest,
				Message: fmt.Sprintf("Error parsing policy object: %v", err),
			},
			Allowed: false,
		}
	}

	// Basic validation - policy objects are generally allowed
	// More sophisticated validation can be added here

	log.Printf("NamespacePolicy validation passed")
	return &admissionv1.AdmissionResponse{
		Result: &metav1.Status{
			Code:    http.StatusOK,
			Message: "NamespacePolicy validation passed",
		},
		Allowed: true,
	}
}