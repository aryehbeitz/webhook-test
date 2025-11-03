#!/bin/bash

set -e

# Handle kubectl context
if [ -z "$1" ]; then
  echo "No context specified. Available contexts:"
  echo ""
  kubectl config get-contexts
  echo ""
  echo "Usage: $0 <context-name> <namespace>"
  echo "Example: $0 gke_my-project_us-central1_cluster-name meetup1"
  exit 1
fi

# Handle namespace
if [ -z "$2" ]; then
  echo "No namespace specified."
  echo ""
  echo "Usage: $0 <context-name> <namespace>"
  echo "Example: $0 gke_my-project_us-central1_cluster-name meetup1"
  exit 1
fi

CONTEXT="$1"
NAMESPACE="$2"

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

# Detect if GKE and set up GCR image paths
USE_GCR="false"
REGISTRY_PREFIX=""
if [[ "$CONTEXT" == gke_* ]]; then
  USE_GCR="true"
  PROJECT_ID="${GCP_PROJECT_ID:-}"
  if [ -z "$PROJECT_ID" ]; then
    PROJECT_ID=$(gcloud config get-value project 2>/dev/null || echo "")
  fi
  if [ -z "$PROJECT_ID" ]; then
    echo "Warning: GKE context detected but GCP_PROJECT_ID not set."
    echo "Images will use local names. Set GCP_PROJECT_ID to use registry images."
  else
    # Use Artifact Registry (modern) instead of GCR (deprecated)
    REGISTRY_LOCATION="${ARTIFACT_REGISTRY_LOCATION:-us-east1}"
    REGISTRY_REPO="${ARTIFACT_REGISTRY_REPO:-honeycomb}"
    REGISTRY_PREFIX="us-east1-docker.pkg.dev/$PROJECT_ID/$REGISTRY_REPO/"
    echo "Using Artifact Registry: $REGISTRY_PREFIX"
  fi
fi

# Create namespace
echo "Creating namespace: $NAMESPACE"
kubectl create namespace "$NAMESPACE" || true

# Deploy PostgreSQL
echo "Deploying PostgreSQL..."
SED_CMD="s/namespace: payment-system/namespace: $NAMESPACE/g"
if [ "$USE_GCR" = "true" ] && [ -n "$REGISTRY_PREFIX" ]; then
  # No image replacement needed for postgres (uses postgres:13 from Docker Hub)
  sed "$SED_CMD" k8s/postgresql.yaml | kubectl apply -f -
else
  sed "$SED_CMD" k8s/postgresql.yaml | kubectl apply -f -
fi

# Wait for PostgreSQL to be ready
echo "Waiting for PostgreSQL to be ready..."
kubectl wait --for=condition=ready pod -l app=postgresql -n "$NAMESPACE" --timeout=300s

# Deploy Temporal
echo "Deploying Temporal server..."
sed "s/namespace: payment-system/namespace: $NAMESPACE/g" k8s/temporal.yaml | kubectl apply -f -

# Wait for Temporal to be ready
echo "Waiting for Temporal to be ready..."
sleep 10
kubectl wait --for=condition=ready pod -l app=temporal -n "$NAMESPACE" --timeout=300s

# Deploy Temporal UI
echo "Deploying Temporal UI..."
sed "s/namespace: payment-system/namespace: $NAMESPACE/g" k8s/temporal-ui.yaml | kubectl apply -f -

# Deploy Payment services
echo "Deploying Payment Worker..."
SED_CMD="s/namespace: payment-system/namespace: $NAMESPACE/g"
if [ "$USE_GCR" = "true" ] && [ -n "$REGISTRY_PREFIX" ]; then
  SED_CMD="$SED_CMD; s|image: payment-worker:latest|image: ${REGISTRY_PREFIX}payment-worker:latest|g"
fi
sed "$SED_CMD" k8s/payment-worker.yaml | kubectl apply -f -

echo "Deploying Payment API..."
SED_CMD="s/namespace: payment-system/namespace: $NAMESPACE/g"
if [ "$USE_GCR" = "true" ] && [ -n "$REGISTRY_PREFIX" ]; then
  SED_CMD="$SED_CMD; s|image: payment-api:latest|image: ${REGISTRY_PREFIX}payment-api:latest|g"
fi
sed "$SED_CMD" k8s/payment-api.yaml | kubectl apply -f -

echo "Deploying Payment Dashboard..."
SED_CMD="s/namespace: payment-system/namespace: $NAMESPACE/g"
if [ "$USE_GCR" = "true" ] && [ -n "$REGISTRY_PREFIX" ]; then
  SED_CMD="$SED_CMD; s|image: payment-dashboard:latest|image: ${REGISTRY_PREFIX}payment-dashboard:latest|g"
fi
sed "$SED_CMD" k8s/payment-dashboard.yaml | kubectl apply -f -

echo ""
echo "Deployment complete!"
echo ""
echo "Waiting for LoadBalancer services to get external IPs..."
echo "Run the following command to get external IPs:"
echo "  kubectl get svc -n $NAMESPACE"
echo ""
echo "Available services:"
echo "  - Payment API: LoadBalancer (check external IP with: kubectl get svc payment-api -n $NAMESPACE)"
echo "  - Dashboard: LoadBalancer (check external IP with: kubectl get svc payment-dashboard -n $NAMESPACE)"
echo "  - Temporal UI: NodePort on port 30088"
echo ""
echo "To check status: kubectl get pods -n $NAMESPACE"
echo "To view logs: kubectl logs -f <pod-name> -n $NAMESPACE"
echo "To get service URLs: kubectl get svc -n $NAMESPACE"
