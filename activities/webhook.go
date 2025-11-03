package activities

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/payment/types"
	"go.temporal.io/sdk/activity"
)

// SendWebhook sends an HTTP POST request to the webhook URL
func SendWebhook(ctx context.Context, req types.PaymentRequest) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Sending webhook", "url", req.WebhookURL, "paymentID", req.ID)

	// Prepare webhook payload
	payload := map[string]interface{}{
		"payment_id": req.ID,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
		"data":       req.Data,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", req.WebhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create webhook request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "Payment-Webhook/1.0")

	// Send request
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read webhook response: %w", err)
	}

	responseStr := fmt.Sprintf("Status: %d, Body: %s", resp.StatusCode, string(body))
	logger.Info("Webhook sent successfully", "status", resp.StatusCode)

	return responseStr, nil
}
