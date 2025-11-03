package workflows

import (
	"time"

	"github.com/payment/activities"
	"github.com/payment/types"
	"go.temporal.io/sdk/workflow"
)

// PaymentWorkflow processes a payment request and sends a webhook after a delay
func PaymentWorkflow(ctx workflow.Context, req types.PaymentRequest) (*types.PaymentResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Payment workflow started", "paymentID", req.ID)

	// Set default sleep time if not provided
	sleepDuration := 5
	if req.Sleep > 0 {
		sleepDuration = req.Sleep
	}

	// Wait for the specified duration
	logger.Info("Sleeping before webhook", "seconds", sleepDuration)
	err := workflow.Sleep(ctx, time.Duration(sleepDuration)*time.Second)
	if err != nil {
		return &types.PaymentResult{
			ID:          req.ID,
			WebhookSent: false,
			Error:       err.Error(),
		}, err
	}

	// Send webhook
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var webhookResp string
	err = workflow.ExecuteActivity(ctx, activities.SendWebhook, req).Get(ctx, &webhookResp)
	if err != nil {
		logger.Error("Failed to send webhook", "error", err)
		return &types.PaymentResult{
			ID:          req.ID,
			WebhookSent: false,
			Error:       err.Error(),
		}, nil
	}

	logger.Info("Payment workflow completed successfully", "paymentID", req.ID)
	return &types.PaymentResult{
		ID:              req.ID,
		WebhookSent:     true,
		WebhookResponse: webhookResp,
	}, nil
}
