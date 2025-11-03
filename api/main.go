package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
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
	r.HandleFunc("/health", api.healthCheck).Methods("GET")
	r.HandleFunc("/payment", api.createPayment).Methods("POST")
	r.HandleFunc("/payment/{id}", api.getPaymentStatus).Methods("GET")
	r.HandleFunc("/payment/{id}/cancel", api.cancelPayment).Methods("POST")
	r.HandleFunc("/payments", api.listPayments).Methods("GET")

	// Enable CORS for dashboard
	r.Use(corsMiddleware)

	log.Printf("Starting API server on port %s\n", port)
	log.Printf("Temporal server: %s\n", temporalAddr)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

func (api *API) healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
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

	workflowID := "payment-" + paymentID

	err := api.temporalClient.CancelWorkflow(context.Background(), workflowID, "")
	if err != nil {
		log.Printf("Failed to cancel workflow: %v\n", err)
		http.Error(w, "Failed to cancel payment", http.StatusInternalServerError)
		return
	}

	log.Printf("Payment workflow cancelled: %s\n", paymentID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "cancelled"})
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(executions)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
