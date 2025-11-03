#!/bin/bash

set -e

echo "Deploying to Kubernetes..."

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
echo "Available services:"
echo "  - Payment API: http://<node-ip>:30080"
echo "  - Dashboard: http://<node-ip>:30000"
echo "  - Temporal UI: http://<node-ip>:30088"
echo ""
echo "To check status: kubectl get pods -n payment-system"
echo "To view logs: kubectl logs -f <pod-name> -n payment-system"
