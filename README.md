# Payment System with Temporal

A payment processing system built with Go and Temporal.io that receives payment requests, waits for a specified duration, and sends webhooks asynchronously.

## Features

- Accept payment requests with arbitrary parameters via REST API
- Configurable sleep duration (default: 5 seconds)
- Automatic webhook delivery after sleep period
- Web dashboard to view, cancel, and manage payments
- Built on self-hosted Temporal.io for reliability and scalability
- Docker, Docker Compose, and Kubernetes ready

## Architecture

- **Payment API** - REST API server (Go) for creating and managing payments
- **Payment Worker** - Temporal worker (Go) that processes payment workflows
- **Dashboard** - Web UI for managing payments
- **Temporal Server** - Self-hosted Temporal.io server
- **PostgreSQL** - Database for Temporal persistence

## Quick Start

### Prerequisites

- Docker and Docker Compose
- Go 1.21+ (for local development)
- kubectl (for Kubernetes deployment)

### Option 1: Docker Compose (Recommended)

1. Start all services:
```bash
bash scripts/docker-compose-up.sh
```

2. Access the services:
   - Dashboard: http://localhost:3000
   - Payment API: http://localhost:8080
   - Temporal UI: http://localhost:8088

3. Stop all services:
```bash
bash scripts/docker-compose-down.sh
```

### Option 2: Docker

1. Build images:
```bash
bash scripts/docker-build.sh
```

2. Run containers manually:
```bash
# Run PostgreSQL
docker run -d --name temporal-postgres \
  -e POSTGRES_PASSWORD=temporal \
  -e POSTGRES_USER=temporal \
  -p 5432:5432 \
  postgres:13

# Run Temporal
docker run -d --name temporal \
  --link temporal-postgres:postgresql \
  -e DB=postgresql \
  -e DB_PORT=5432 \
  -e POSTGRES_USER=temporal \
  -e POSTGRES_PWD=temporal \
  -e POSTGRES_SEEDS=postgresql \
  -p 7233:7233 \
  temporalio/auto-setup:latest

# Run Payment Worker
docker run -d --name payment-worker \
  --link temporal:temporal \
  -e TEMPORAL_ADDRESS=temporal:7233 \
  payment-worker:latest

# Run Payment API
docker run -d --name payment-api \
  --link temporal:temporal \
  -e TEMPORAL_ADDRESS=temporal:7233 \
  -p 8080:8080 \
  payment-api:latest

# Run Dashboard
docker run -d --name payment-dashboard \
  -p 3000:80 \
  payment-dashboard:latest
```

### Option 3: Kubernetes

1. Build and load images:
```bash
bash scripts/k8s-build.sh

# For minikube:
eval $(minikube docker-env)
bash scripts/k8s-build.sh

# For kind:
bash scripts/k8s-build.sh
kind load docker-image payment-worker:latest payment-api:latest payment-dashboard:latest
```

2. Deploy to cluster:
```bash
bash scripts/k8s-deploy.sh
```

3. Access services (NodePort):
   - Dashboard: http://<node-ip>:30000
   - Payment API: http://<node-ip>:30080
   - Temporal UI: http://<node-ip>:30088

4. Delete deployment:
```bash
bash scripts/k8s-delete.sh
```

## API Usage

### Create Payment

```bash
curl -X POST http://localhost:8080/payment \
  -H "Content-Type: application/json" \
  -d '{
    "webhook_url": "https://webhook.site/your-unique-url",
    "sleep": 10,
    "data": {
      "amount": 100,
      "currency": "USD",
      "customer_id": "12345"
    }
  }'
```

Response:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000"
}
```

### Get Payment Status

```bash
curl http://localhost:8080/payment/{id}
```

Response:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "Running",
  "run_id": "abc123...",
  "result": "{\"id\":\"...\",\"webhook_sent\":true,\"webhook_response\":\"...\"}"
}
```

### Cancel Payment

```bash
curl -X POST http://localhost:8080/payment/{id}/cancel
```

### List Payments

```bash
curl http://localhost:8080/payments
```

## Local Development

### Prerequisites

- Go 1.21+
- Running Temporal server (use Docker Compose)

### Setup

1. Install dependencies:
```bash
go mod download
```

2. Start Temporal server:
```bash
docker-compose up -d postgresql temporal temporal-ui
```

3. Run the worker:
```bash
go run worker/main.go
```

4. Run the API:
```bash
go run api/main.go
```

5. Open dashboard:
```bash
cd dashboard
python3 -m http.server 3000
# or use any static file server
```

### Environment Variables

**Worker & API:**
- `TEMPORAL_ADDRESS` - Temporal server address (default: `localhost:7233`)

**API:**
- `PORT` - API server port (default: `8080`)

## Project Structure

```
.
├── activities/          # Temporal activities (webhook sender)
├── api/                 # REST API server
├── dashboard/           # Web dashboard
├── k8s/                 # Kubernetes manifests
├── scripts/             # Deployment scripts
├── temporal-config/     # Temporal configuration
├── workflows/           # Temporal workflows
├── worker/              # Temporal worker
├── docker-compose.yml   # Docker Compose configuration
├── Dockerfile.api       # API Dockerfile
├── Dockerfile.worker    # Worker Dockerfile
├── Dockerfile.dashboard # Dashboard Dockerfile
├── go.mod               # Go module definition
└── README.md
```

## Webhook Payload

When the sleep period expires, the system sends a POST request to the specified webhook URL with this payload:

```json
{
  "payment_id": "550e8400-e29b-41d4-a716-446655440000",
  "timestamp": "2025-11-03T10:30:00Z",
  "data": {
    "amount": 100,
    "currency": "USD",
    "customer_id": "12345"
  }
}
```

## Dashboard Features

- Create new payments with custom data
- View all payments and their status
- See detailed payment information
- Cancel running payments
- Auto-refresh every 5 seconds

## Monitoring

Access the Temporal Web UI to:
- View workflow executions
- Debug failed workflows
- Monitor worker health
- View task queues

## Scaling

### Horizontal Scaling

**Workers:** Increase replicas to process more payments concurrently:
```bash
# Docker Compose
docker-compose up -d --scale payment-worker=5

# Kubernetes
kubectl scale deployment payment-worker -n payment-system --replicas=5
```

**API:** Scale API servers for higher request throughput:
```bash
# Kubernetes
kubectl scale deployment payment-api -n payment-system --replicas=3
```

### Temporal Clustering

For production, consider:
- Running Temporal in a clustered configuration
- Using a managed PostgreSQL service
- Setting up proper monitoring and alerting

## Troubleshooting

### Worker not processing workflows
- Check if worker is connected to Temporal: `docker logs payment-worker`
- Verify Temporal server is running: `curl http://localhost:7233`
- Check worker logs for errors

### API cannot reach Temporal
- Ensure Temporal is running and accessible
- Check `TEMPORAL_ADDRESS` environment variable
- Verify network connectivity between containers

### Webhook not sent
- Check worker logs for webhook errors
- Verify webhook URL is accessible from worker container
- Check Temporal UI for workflow execution details

## License

MIT
