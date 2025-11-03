#!/bin/bash

# Example script to test the payment API

API_URL="http://localhost:8080"

echo "Creating a test payment..."

# Create a payment with custom data
RESPONSE=$(curl -s -X POST ${API_URL}/payment \
  -H "Content-Type: application/json" \
  -d '{
    "webhook_url": "https://webhook.site/unique-id",
    "sleep": 10,
    "data": {
      "amount": 100.50,
      "currency": "USD",
      "customer_id": "cust_12345",
      "order_id": "ord_67890"
    }
  }')

echo "Response: $RESPONSE"

# Extract payment ID from response
PAYMENT_ID=$(echo $RESPONSE | grep -o '"id":"[^"]*' | cut -d'"' -f4)

if [ -z "$PAYMENT_ID" ]; then
  echo "Failed to create payment"
  exit 1
fi

echo "Payment created with ID: $PAYMENT_ID"
echo ""
echo "Checking payment status..."

# Check status
curl -s ${API_URL}/payment/${PAYMENT_ID} | jq .

echo ""
echo "You can view the payment in the dashboard at: http://localhost:3000"
echo "Or view the workflow in Temporal UI at: http://localhost:8088"
