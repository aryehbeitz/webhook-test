#!/bin/bash

set -e

if [ -z "$1" ]; then
  echo "Usage: $0 <service>"
  echo "Services: api, worker, dashboard"
  exit 1
fi

SERVICE="$1"

# Get GCP project ID
PROJECT_ID="${GCP_PROJECT_ID:-}"
if [ -z "$PROJECT_ID" ]; then
  PROJECT_ID=$(gcloud config get-value project 2>/dev/null || echo "")
  if [ -z "$PROJECT_ID" ]; then
    echo "Error: GCP_PROJECT_ID not set and unable to get from gcloud config"
    exit 1
  fi
fi

# Use Artifact Registry
REGISTRY_LOCATION="${ARTIFACT_REGISTRY_LOCATION:-us-east1}"
REGISTRY_REPO="${ARTIFACT_REGISTRY_REPO:-honeycomb}"
REGISTRY="us-east1-docker.pkg.dev/$PROJECT_ID/$REGISTRY_REPO"

case "$SERVICE" in
  api)
    DOCKERFILE="Dockerfile.api"
    IMAGE_NAME="payment-api"
    ;;
  worker)
    DOCKERFILE="Dockerfile.worker"
    IMAGE_NAME="payment-worker"
    ;;
  dashboard)
    DOCKERFILE="Dockerfile.dashboard"
    IMAGE_NAME="payment-dashboard"
    ;;
  *)
    echo "Unknown service: $SERVICE"
    echo "Available services: api, worker, dashboard"
    exit 1
    ;;
esac

echo "Building and pushing $IMAGE_NAME to Artifact Registry..."
docker buildx build --platform linux/amd64 -f "$DOCKERFILE" -t "$REGISTRY/$IMAGE_NAME:latest" --push .

echo "✅ $IMAGE_NAME built and pushed successfully!"

