package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/payment/types"
	"github.com/payment/workflows"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
)

const (
	TaskQueue = "payment-task-queue"
)

type API struct {
	temporalClient client.Client
}

type CreatePaymentRequest struct {
	WebhookURL string                 `json:"webhook_url"`
	Sleep      int                    `json:"sleep,omitempty"`
	Data       map[string]interface{} `json:"data,omitempty"`
}

type CreatePaymentResponse struct {
	ID string `json:"id"`
}

type PaymentStatusResponse struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Result   string `json:"result,omitempty"`
	Error    string `json:"error,omitempty"`
	RunID    string `json:"run_id"`
}

func main() {
	// Get configuration from environment
	temporalAddr := os.Getenv("TEMPORAL_ADDRESS")
	if temporalAddr == "" {
		temporalAddr = "localhost:7233"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Create Temporal client
	c, err := client.Dial(client.Options{
		HostPort: temporalAddr,
	})
	if err != nil {
		log.Fatalln("Unable to create Temporal client", err)
	}
	defer c.Close()

	api := &API{
		temporalClient: c,
	}

	// Setup routes
	r := mux.NewRouter()

	// Enable CORS for dashboard (must be applied before routes)
	// This middleware handles OPTIONS requests for all routes
	r.Use(corsMiddleware)

	r.HandleFunc("/health", api.healthCheck).Methods("GET")
	// More specific routes first to avoid matching conflicts
	r.HandleFunc("/payment/{id}/cancel", api.cancelPayment).Methods("POST", "OPTIONS")
	r.HandleFunc("/payment/{id}/delete", api.cancelPayment).Methods("POST", "OPTIONS") // Use same handler, action determined by path
	r.HandleFunc("/payments/delete-all", api.deleteAllPayments).Methods("POST", "OPTIONS")
	r.HandleFunc("/payment/{id}", api.getPaymentStatus).Methods("GET", "OPTIONS")
	r.HandleFunc("/payment", api.createPayment).Methods("POST", "OPTIONS")
	r.HandleFunc("/payments", api.listPayments).Methods("GET", "OPTIONS")

	// Handle 404s that are OPTIONS requests (CORS preflight)
	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log.Printf("404 Handler: Method=%s, Path=%s, URL=%s, Host=%s\n", req.Method, req.URL.Path, req.URL.String(), req.Host)
		if req.Method == "OPTIONS" {
			optionsHandler(w, req)
			return
		}
		log.Printf("Sending 404 response for: %s %s\n", req.Method, req.URL.Path)
		http.NotFound(w, req)
	})

	log.Printf("Starting API server on port %s\n", port)
	log.Printf("Temporal server: %s\n", temporalAddr)
	log.Printf("Registered routes: /payment (POST), /payment/{id}/cancel (POST), /payment/{id}/delete (POST), /payments/delete-all (POST), /payment/{id} (GET), /payments (GET)\n")

	// Debug: Walk routes
	r.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		pathTemplate, err := route.GetPathTemplate()
		if err == nil {
			methods, _ := route.GetMethods()
			log.Printf("Route registered: %s %v\n", pathTemplate, methods)
		}
		return nil
	})

	log.Fatal(http.ListenAndServe(":"+port, r))
}

func (api *API) healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// Include build info to verify deployment
	buildInfo := map[string]string{
		"status": "ok",
		"version": "2025-11-04-10:22-delete-fix",
		"hasDelete": "true",
	}
	json.NewEncoder(w).Encode(buildInfo)
}

func (api *API) createPayment(w http.ResponseWriter, r *http.Request) {
	var req CreatePaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.WebhookURL == "" {
		http.Error(w, "webhook_url is required", http.StatusBadRequest)
		return
	}

	// Generate payment ID
	paymentID := uuid.New().String()

	// Create payment request
	paymentReq := types.PaymentRequest{
		ID:         paymentID,
		WebhookURL: req.WebhookURL,
		Sleep:      req.Sleep,
		Data:       req.Data,
	}

	// Start workflow
	workflowOptions := client.StartWorkflowOptions{
		ID:        "payment-" + paymentID,
		TaskQueue: TaskQueue,
	}

	_, err := api.temporalClient.ExecuteWorkflow(context.Background(), workflowOptions, workflows.PaymentWorkflow, paymentReq)
	if err != nil {
		log.Printf("Failed to start workflow: %v\n", err)
		http.Error(w, "Failed to create payment", http.StatusInternalServerError)
		return
	}

	log.Printf("Payment workflow started: %s\n", paymentID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(CreatePaymentResponse{ID: paymentID})
}

func (api *API) getPaymentStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	paymentID := vars["id"]

	workflowID := "payment-" + paymentID

	// Get workflow execution
	desc, err := api.temporalClient.DescribeWorkflowExecution(context.Background(), workflowID, "")
	if err != nil {
		http.Error(w, "Payment not found", http.StatusNotFound)
		return
	}

	status := desc.WorkflowExecutionInfo.Status.String()

	response := PaymentStatusResponse{
		ID:     paymentID,
		Status: status,
		RunID:  desc.WorkflowExecutionInfo.Execution.RunId,
	}

	// If workflow is completed, get the result
	if desc.WorkflowExecutionInfo.Status.String() == "Completed" {
		var result types.PaymentResult
		err = api.temporalClient.GetWorkflow(context.Background(), workflowID, "").Get(context.Background(), &result)
		if err == nil {
			resultJSON, _ := json.Marshal(result)
			response.Result = string(resultJSON)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (api *API) cancelPayment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	paymentID := vars["id"]

	// Determine action from path
	isDelete := strings.HasSuffix(r.URL.Path, "/delete")

	workflowID := "payment-" + paymentID

	var err error
	var statusMsg string
	var logMsg string

	if isDelete {
		// Terminate the workflow (delete)
		err = api.temporalClient.TerminateWorkflow(context.Background(), workflowID, "", "Deleted by user")
		statusMsg = "deleted"
		logMsg = "Payment workflow deleted"
	} else {
		// Cancel the workflow
		err = api.temporalClient.CancelWorkflow(context.Background(), workflowID, "")
		statusMsg = "cancelled"
		logMsg = "Payment workflow cancelled"
	}

	if err != nil {
		// If workflow is already completed/terminated, consider it successful
		if strings.Contains(err.Error(), "already completed") || strings.Contains(err.Error(), "already terminated") {
			log.Printf("Workflow %s already %s, treating as success\n", workflowID, statusMsg)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"status": statusMsg, "message": "Workflow was already completed"})
			return
		}
		log.Printf("Failed to %s workflow: %v\n", statusMsg, err)
		http.Error(w, fmt.Sprintf("Failed to %s payment", statusMsg), http.StatusInternalServerError)
		return
	}

	log.Printf("%s: %s\n", logMsg, paymentID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": statusMsg})
}

func (api *API) deletePayment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	paymentID := vars["id"]

	workflowID := "payment-" + paymentID

	err := api.temporalClient.TerminateWorkflow(context.Background(), workflowID, "", "Deleted by user")
	if err != nil {
		log.Printf("Failed to delete workflow: %v\n", err)
		http.Error(w, "Failed to delete payment", http.StatusInternalServerError)
		return
	}

	log.Printf("Payment workflow deleted: %s\n", paymentID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

func (api *API) deleteAllPayments(w http.ResponseWriter, r *http.Request) {
	log.Printf("deleteAllPayments called: Method=%s, Path=%s\n", r.Method, r.URL.Path)

	// List all payment workflows
	query := "WorkflowType='PaymentWorkflow'"
	ctx := context.Background()
	request := &workflowservice.ListWorkflowExecutionsRequest{
		Query: query,
	}

	resp, err := api.temporalClient.ListWorkflow(ctx, request)
	if err != nil {
		log.Printf("Failed to list workflows: %v\n", err)
		http.Error(w, "Failed to list payments", http.StatusInternalServerError)
		return
	}

	deletedCount := 0
	terminatedCount := 0
	completedCount := 0
	failedCount := 0

	for _, exec := range resp.Executions {
		workflowID := exec.Execution.WorkflowId
		status := exec.Status.String()

		// Check status before attempting to terminate
		if status == "Completed" || status == "Canceled" {
			// Completed/canceled workflows can't be terminated, but we'll count them as "deleted" for UI purposes
			completedCount++
			log.Printf("Workflow %s is %s, will be filtered from list\n", workflowID, status)
			continue
		}

		if status == "Terminated" {
			// Already terminated
			terminatedCount++
			continue
		}

		// Try to terminate running workflows
		err := api.temporalClient.TerminateWorkflow(ctx, workflowID, "", "Deleted all by user")
		if err != nil {
			// If workflow is already completed/terminated, consider it successfully deleted
			if strings.Contains(err.Error(), "already completed") || strings.Contains(err.Error(), "already terminated") {
				log.Printf("Workflow %s already completed/terminated, treating as deleted\n", workflowID)
				completedCount++
			} else {
				log.Printf("Failed to delete workflow %s: %v\n", workflowID, err)
				failedCount++
			}
			continue
		}
		deletedCount++
	}

	totalDeleted := deletedCount + terminatedCount + completedCount

	log.Printf("Deleted %d payment workflows (terminated: %d, completed/canceled: %d, failed: %d)\n", totalDeleted, terminatedCount, completedCount, failedCount)

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"status": "deleted",
		"count":  totalDeleted,
		"terminated": deletedCount,
		"completed_filtered": completedCount,
	}
	if failedCount > 0 {
		response["failed"] = failedCount
	}
	json.NewEncoder(w).Encode(response)
}

func (api *API) listPayments(w http.ResponseWriter, r *http.Request) {
	// List recent workflow executions
	query := "WorkflowType='PaymentWorkflow'"

	var executions []map[string]interface{}

	ctx := context.Background()
	request := &workflowservice.ListWorkflowExecutionsRequest{
		Query: query,
	}

	resp, err := api.temporalClient.ListWorkflow(ctx, request)
	if err != nil {
		log.Printf("Failed to list workflows: %v\n", err)
		http.Error(w, "Failed to list payments", http.StatusInternalServerError)
		return
	}

	for _, exec := range resp.Executions {
		// Skip terminated workflows (they've been "deleted")
		// Also skip completed and canceled workflows (they can't be deleted from Temporal history,
		// but we filter them from the list after delete-all operations)
		status := exec.Status.String()
		if status == "Terminated" || status == "Completed" || status == "Canceled" {
			continue
		}

		// Extract payment ID from workflow ID
		paymentID := exec.Execution.WorkflowId
		if len(paymentID) > 8 && paymentID[:8] == "payment-" {
			paymentID = paymentID[8:]
		}

		startTime := ""
		if exec.StartTime != nil {
			startTime = exec.StartTime.Format(time.RFC3339)
		}

	executions = append(executions, map[string]interface{}{
		"id":          paymentID,
		"workflow_id": exec.Execution.WorkflowId,
		"run_id":      exec.Execution.RunId,
		"status":      exec.Status.String(),
		"start_time":  startTime,
	})
	}

	// Ensure we always return an array, never null
	if executions == nil {
		executions = []map[string]interface{}{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(executions)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Handle OPTIONS requests globally before routing
func optionsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.WriteHeader(http.StatusOK)
}
