package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type WebhookHandler struct{}

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

func (wh *WebhookHandler) ServeMutate(w http.ResponseWriter, r *http.Request) {
	log.Println("Received mutation request")

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Failed to read request body: %v", err)
		http.Error(w, "Failed to read request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse admission review
	var admissionReview admissionv1.AdmissionReview
	if err := json.Unmarshal(body, &admissionReview); err != nil {
		log.Printf("Failed to parse admission review: %v", err)
		http.Error(w, "Failed to parse request", http.StatusBadRequest)
		return
	}

	// Process the request
	admissionResponse := wh.mutate(admissionReview.Request)

	// Construct response
	responseAdmissionReview := admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "admission.k8s.io/v1",
			Kind:       "AdmissionReview",
		},
		Response: admissionResponse,
	}
	responseAdmissionReview.Response.UID = admissionReview.Request.UID

	// Send response
	respBytes, err := json.Marshal(responseAdmissionReview)
	if err != nil {
		log.Printf("Failed to marshal response: %v", err)
		http.Error(w, "Failed to create response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(respBytes)
	log.Println("Mutation response sent")
}

func (wh *WebhookHandler) mutate(req *admissionv1.AdmissionRequest) *admissionv1.AdmissionResponse {
	// Default response
	response := &admissionv1.AdmissionResponse{
		Allowed: true,
	}

	// Only handle Pod creation
	if req.Kind.Kind != "Pod" {
		return response
	}

	// Parse the Pod object
	var pod corev1.Pod
	if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
		log.Printf("Failed to unmarshal pod: %v", err)
		return response
	}

	log.Printf("Processing pod: %s in namespace: %s", pod.Name, pod.Namespace)

	// Check if pod has app=frontend label
	if pod.Labels == nil || pod.Labels["app"] != "frontend" {
		log.Println("Pod does not have app=frontend label, skipping")
		return response
	}

	log.Println("Frontend pod detected, injecting sidecar...")

	// Create patch to inject sidecar
	patches := createSidecarPatch(&pod)
	if len(patches) == 0 {
		return response
	}

	patchBytes, err := json.Marshal(patches)
	if err != nil {
		log.Printf("Failed to marshal patches: %v", err)
		return response
	}

	response.Patch = patchBytes
	patchType := admissionv1.PatchTypeJSONPatch
	response.PatchType = &patchType

	log.Printf("Sidecar injected successfully")
	return response
}

func createSidecarPatch(pod *corev1.Pod) []patchOperation {
	var patches []patchOperation

	// Define the sidecar container
	sidecar := corev1.Container{
		Name:    "frontend-sidecar",
		Image:   "alpine:latest", // Replace with your sidecar image
		Command: []string{"sh", "-c", "tail -f /dev/null"},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("100m"),
				corev1.ResourceMemory: resource.MustParse("128Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("200m"),
				corev1.ResourceMemory: resource.MustParse("256Mi"),
			},
		},
		Env: []corev1.EnvVar{
			{
				Name:  "SIDECAR_MODE",
				Value: "frontend",
			},
			{
				Name: "NODE_NAME",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "spec.nodeName",
					},
				},
			},
			{
				Name: "POD_NAME",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.name",
					},
				},
			},
		},
	}

	// Check if containers array exists
	if len(pod.Spec.Containers) == 0 {
		// If no containers, add the sidecar as the first container
		patches = append(patches, patchOperation{
			Op:    "add",
			Path:  "/spec/containers",
			Value: []corev1.Container{sidecar},
		})
	} else {
		// Add sidecar to existing containers
		patches = append(patches, patchOperation{
			Op:    "add",
			Path:  "/spec/containers/-",
			Value: sidecar,
		})
	}

	// Add annotation to mark that sidecar was injected
	if pod.Annotations == nil {
		patches = append(patches, patchOperation{
			Op:   "add",
			Path: "/metadata/annotations",
			Value: map[string]string{
				"sidecar-injector": "injected",
			},
		})
	} else {
		patches = append(patches, patchOperation{
			Op:    "add",
			Path:  "/metadata/annotations/sidecar-injector",
			Value: "injected",
		})
	}

	return patches
}
