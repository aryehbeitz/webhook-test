#!/bin/bash

set -e

echo "Deleting Kubernetes resources..."

kubectl delete -f k8s/payment-dashboard.yaml --ignore-not-found=true
kubectl delete -f k8s/payment-api.yaml --ignore-not-found=true
kubectl delete -f k8s/payment-worker.yaml --ignore-not-found=true
kubectl delete -f k8s/temporal-ui.yaml --ignore-not-found=true
kubectl delete -f k8s/temporal.yaml --ignore-not-found=true
kubectl delete -f k8s/postgresql.yaml --ignore-not-found=true
kubectl delete -f k8s/namespace.yaml --ignore-not-found=true

echo "All resources deleted."
