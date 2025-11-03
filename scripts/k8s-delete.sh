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

echo "Deleting Kubernetes resources in context: $CURRENT_CONTEXT"
echo "Deleting from namespace: $NAMESPACE"
echo ""

sed "s/namespace: payment-system/namespace: $NAMESPACE/g" k8s/payment-dashboard.yaml | kubectl delete -f - --ignore-not-found=true
sed "s/namespace: payment-system/namespace: $NAMESPACE/g" k8s/payment-api.yaml | kubectl delete -f - --ignore-not-found=true
sed "s/namespace: payment-system/namespace: $NAMESPACE/g" k8s/payment-worker.yaml | kubectl delete -f - --ignore-not-found=true
sed "s/namespace: payment-system/namespace: $NAMESPACE/g" k8s/temporal-ui.yaml | kubectl delete -f - --ignore-not-found=true
sed "s/namespace: payment-system/namespace: $NAMESPACE/g" k8s/temporal.yaml | kubectl delete -f - --ignore-not-found=true
sed "s/namespace: payment-system/namespace: $NAMESPACE/g" k8s/postgresql.yaml | kubectl delete -f - --ignore-not-found=true
kubectl delete namespace "$NAMESPACE" --ignore-not-found=true

echo "All resources deleted."
