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

echo "Deleting Kubernetes resources in context: $CURRENT_CONTEXT"
echo ""

kubectl delete -f k8s/payment-dashboard.yaml --ignore-not-found=true
kubectl delete -f k8s/payment-api.yaml --ignore-not-found=true
kubectl delete -f k8s/payment-worker.yaml --ignore-not-found=true
kubectl delete -f k8s/temporal-ui.yaml --ignore-not-found=true
kubectl delete -f k8s/temporal.yaml --ignore-not-found=true
kubectl delete -f k8s/postgresql.yaml --ignore-not-found=true
kubectl delete -f k8s/namespace.yaml --ignore-not-found=true

echo "All resources deleted."
