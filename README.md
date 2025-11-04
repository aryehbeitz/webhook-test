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
  -e DB=postgres12 \
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

1. Build Docker images:
```bash
bash scripts/k8s-build.sh

# For minikube:
eval $(minikube docker-env)
bash scripts/k8s-build.sh

# For kind:
bash scripts/k8s-build.sh
kind load docker-image payment-worker:latest payment-api:latest payment-dashboard:latest

# For GCP GKE:
# Push images to Container Registry or Artifact Registry
# Example: gcloud builds submit --tag gcr.io/PROJECT_ID/payment-api:latest
```

2. Deploy to cluster (requires kubectl context):

   First, list available contexts:
   ```bash
   bash scripts/k8s-deploy.sh
   ```

   Then deploy with the desired context and namespace:
   ```bash
   bash scripts/k8s-deploy.sh <context-name> <namespace>
   ```

   Example for GCP GKE:
   ```bash
   bash scripts/k8s-deploy.sh gke_my-project_us-central1_cluster-name meetup1
   ```

#### NPM Scripts for Kubernetes Deployment

The project includes npm scripts for convenient Kubernetes deployment workflows:

**Build Commands:**
```bash
npm run k8s:build:api        # Build and push API image
npm run k8s:build:worker    # Build and push worker image
npm run k8s:build:dashboard # Build and push dashboard image
npm run k8s:build:all       # Build all images
```

**Deploy Commands (restart deployments):**
```bash
npm run k8s:deploy:api        # Restart API deployment
npm run k8s:deploy:worker     # Restart worker deployment
npm run k8s:deploy:dashboard  # Restart dashboard deployment
npm run k8s:deploy:all        # Restart all deployments
```

**Combined (build + deploy):**
```bash
npm run k8s:build-deploy:api        # Build and deploy API
npm run k8s:build-deploy:worker     # Build and deploy worker
npm run k8s:build-deploy:dashboard  # Build and deploy dashboard
```

**Utility Commands:**
```bash
npm run k8s:status          # Show pod status
npm run k8s:logs:api        # Follow API logs
npm run k8s:logs:worker     # Follow worker logs
npm run k8s:logs:dashboard  # Follow dashboard logs
```

**Custom Namespace:**
```bash
K8S_NAMESPACE=meetup2 npm run k8s:deploy:api
```

**Example Workflow:**
```bash
# Make changes to dashboard code, then:
npm run k8s:build-deploy:dashboard

# Or just restart API without rebuilding:
npm run k8s:deploy:api
```

3. Access services:

   **LoadBalancer Services** (for GCP GKE and cloud providers):

   Get external IPs:
   ```bash
   kubectl get svc -n payment-system
   ```

   - Dashboard: `http://<external-ip>` (from payment-dashboard service)
   - Payment API: `http://<external-ip>:8080` (from payment-api service)
   - Temporal UI: Still uses NodePort on port 30088

   **Port-forward** (alternative for local access):
   ```bash
   kubectl port-forward -n payment-system svc/payment-dashboard 3000:80
   kubectl port-forward -n payment-system svc/payment-api 8080:8080
   kubectl port-forward -n payment-system svc/temporal-ui 8088:8080
   ```
   Then access at http://localhost:3000, http://localhost:8080, http://localhost:8088

4. Verify deployment:
```bash
# Check all pods are running
kubectl get pods -n payment-system

# Check service status and external IPs
kubectl get svc -n payment-system

# View logs
kubectl logs -n payment-system -l app=payment-api
kubectl logs -n payment-system -l app=payment-worker
```

5. Delete deployment:
```bash
bash scripts/k8s-delete.sh <context-name>
```

**Note for GCP GKE:**
- LoadBalancer services will automatically get external IPs (may take 1-2 minutes)
- Worker pods have outgoing internet connectivity enabled by default for webhook POST requests
- Ensure your cluster has proper firewall rules for external access
- Consider using Ingress with SSL certificates for production deployments

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
