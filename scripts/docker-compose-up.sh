#!/bin/bash

set -e

echo "Starting services with Docker Compose..."

docker-compose up -d

echo ""
echo "Services are starting up. Please wait a moment for all services to be ready."
echo ""
echo "Available services:"
echo "  - Payment API: http://localhost:8080"
echo "  - Dashboard: http://localhost:3000"
echo "  - Temporal UI: http://localhost:8088"
echo ""
echo "To view logs: docker-compose logs -f"
echo "To stop: docker-compose down"
