#!/bin/bash

set -e

# Handle kubectl context
if [ -z "$1" ]; then
  echo "No context specified. Available contexts:"
  echo ""
  kubectl config get-contexts
  echo ""
  echo "Usage: $0 <context-name>"
  echo "Example: $0 gke_my-project_us-central1_cluster-name"
  exit 1
fi

CONTEXT="$1"
echo "Switching to context: $CONTEXT"
kubectl config use-context "$CONTEXT"

# Verify context switch
CURRENT_CONTEXT=$(kubectl config current-context)
if [ "$CURRENT_CONTEXT" != "$CONTEXT" ]; then
  echo "Error: Failed to switch to context $CONTEXT"
  echo "Current context is: $CURRENT_CONTEXT"
  exit 1
fi

echo "Deploying to Kubernetes in context: $CURRENT_CONTEXT"
echo ""

# Create namespace
kubectl apply -f k8s/namespace.yaml

# Deploy PostgreSQL
echo "Deploying PostgreSQL..."
kubectl apply -f k8s/postgresql.yaml

# Wait for PostgreSQL to be ready
echo "Waiting for PostgreSQL to be ready..."
kubectl wait --for=condition=ready pod -l app=postgresql -n payment-system --timeout=300s

# Deploy Temporal
echo "Deploying Temporal server..."
kubectl apply -f k8s/temporal.yaml

# Wait for Temporal to be ready
echo "Waiting for Temporal to be ready..."
sleep 10
kubectl wait --for=condition=ready pod -l app=temporal -n payment-system --timeout=300s

# Deploy Temporal UI
echo "Deploying Temporal UI..."
kubectl apply -f k8s/temporal-ui.yaml

# Deploy Payment services
echo "Deploying Payment Worker..."
kubectl apply -f k8s/payment-worker.yaml

echo "Deploying Payment API..."
kubectl apply -f k8s/payment-api.yaml

echo "Deploying Payment Dashboard..."
kubectl apply -f k8s/payment-dashboard.yaml

echo ""
echo "Deployment complete!"
echo ""
echo "Waiting for LoadBalancer services to get external IPs..."
echo "Run the following command to get external IPs:"
echo "  kubectl get svc -n payment-system"
echo ""
echo "Available services:"
echo "  - Payment API: LoadBalancer (check external IP with: kubectl get svc payment-api -n payment-system)"
echo "  - Dashboard: LoadBalancer (check external IP with: kubectl get svc payment-dashboard -n payment-system)"
echo "  - Temporal UI: NodePort on port 30088"
echo ""
echo "To check status: kubectl get pods -n payment-system"
echo "To view logs: kubectl logs -f <pod-name> -n payment-system"
echo "To get service URLs: kubectl get svc -n payment-system"
