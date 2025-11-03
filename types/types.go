package types

type PaymentRequest struct {
	ID         string                 `json:"id"`
	WebhookURL string                 `json:"webhook_url"`
	Sleep      int                    `json:"sleep"` // in seconds
	Data       map[string]interface{} `json:"data"`
}

type PaymentResult struct {
	ID              string `json:"id"`
	WebhookSent     bool   `json:"webhook_sent"`
	WebhookResponse string `json:"webhook_response,omitempty"`
	Error           string `json:"error,omitempty"`
}
