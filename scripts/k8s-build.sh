#!/bin/bash

set -e

# Check if GCR registry should be used (default: true for GKE)
USE_GCR="${USE_GCR:-true}"
PROJECT_ID="${GCP_PROJECT_ID:-}"

if [ "$USE_GCR" = "true" ]; then
  if [ -z "$PROJECT_ID" ]; then
    # Try to get project from gcloud config
    PROJECT_ID=$(gcloud config get-value project 2>/dev/null || echo "")
    if [ -z "$PROJECT_ID" ]; then
      echo "Error: GCP_PROJECT_ID not set and unable to get from gcloud config"
      echo "Usage: GCP_PROJECT_ID=your-project-id $0"
      echo "Or set USE_GCR=false for local/minikube/kind"
      exit 1
    fi
  fi

  # Use Artifact Registry (modern) instead of GCR (deprecated)
  REGISTRY_LOCATION="${ARTIFACT_REGISTRY_LOCATION:-us-east1}"
  REGISTRY_REPO="${ARTIFACT_REGISTRY_REPO:-honeycomb}"
  REGISTRY="us-east1-docker.pkg.dev/$PROJECT_ID/$REGISTRY_REPO"
  echo "Building Docker images for GKE (Artifact Registry: $REGISTRY)..."

  # Build and push images directly to Artifact Registry (explicitly for linux/amd64)
  # This ensures the correct platform manifest is pushed
  echo "Building and pushing payment-worker..."
  docker buildx build --platform linux/amd64 -f Dockerfile.worker -t $REGISTRY/payment-worker:latest --push .

  echo "Building and pushing payment-api..."
  docker buildx build --platform linux/amd64 -f Dockerfile.api -t $REGISTRY/payment-api:latest --push .

  echo "Building and pushing payment-dashboard..."
  docker buildx build --platform linux/amd64 -f Dockerfile.dashboard -t $REGISTRY/payment-dashboard:latest --push .

  # Also tag locally for convenience (skip on ARM64 Macs since images are amd64-only)
  # Only pull if we're on amd64 or explicitly requested
  if [ "$(uname -m)" = "x86_64" ] || [ "${PULL_LOCAL_IMAGES:-false}" = "true" ]; then
    docker pull --platform linux/amd64 $REGISTRY/payment-worker:latest 2>/dev/null || echo "Skipping local pull for payment-worker (amd64 image on ARM64 host)"
    docker pull --platform linux/amd64 $REGISTRY/payment-api:latest 2>/dev/null || echo "Skipping local pull for payment-api (amd64 image on ARM64 host)"
    docker pull --platform linux/amd64 $REGISTRY/payment-dashboard:latest 2>/dev/null || echo "Skipping local pull for payment-dashboard (amd64 image on ARM64 host)"
    docker tag $REGISTRY/payment-worker:latest payment-worker:latest 2>/dev/null || true
    docker tag $REGISTRY/payment-api:latest payment-api:latest 2>/dev/null || true
    docker tag $REGISTRY/payment-dashboard:latest payment-dashboard:latest 2>/dev/null || true
  else
    echo "Skipping local image pull (images are amd64-only, running on $(uname -m))"
  fi

  echo ""
  echo "✅ Docker images built and pushed to Artifact Registry with linux/amd64 platform!"
else
  echo "Building Docker images for local/minikube/kind..."

  # Build images
  docker build -t payment-worker:latest -f Dockerfile.worker .
  docker build -t payment-api:latest -f Dockerfile.api .
  docker build -t payment-dashboard:latest -f Dockerfile.dashboard .

  echo ""
  echo "Docker images built successfully!"
  echo ""
  echo "If using minikube, run: eval \$(minikube docker-env) && $0"
  echo "If using kind, run: kind load docker-image payment-worker:latest payment-api:latest payment-dashboard:latest"
fi
