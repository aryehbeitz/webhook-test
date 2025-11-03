#!/bin/bash

set -e

echo "Building Docker images..."

docker build -t payment-worker:latest -f Dockerfile.worker .
docker build -t payment-api:latest -f Dockerfile.api .
docker build -t payment-dashboard:latest -f Dockerfile.dashboard .

echo "Docker images built successfully!"
