#!/bin/bash

set -e

echo "Building Docker images for Kubernetes..."

# Build images
docker build -t payment-worker:latest -f Dockerfile.worker .
docker build -t payment-api:latest -f Dockerfile.api .
docker build -t payment-dashboard:latest -f Dockerfile.dashboard .

echo ""
echo "Docker images built successfully!"
echo ""
echo "If using minikube, run: eval \$(minikube docker-env)"
echo "If using kind, run: kind load docker-image payment-worker:latest payment-api:latest payment-dashboard:latest"
