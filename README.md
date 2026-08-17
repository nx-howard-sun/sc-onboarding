# ByteSized Security Central (Go)

Implementation using:
- Go
- go-kit (service/endpoint/transport layering)
- Ent (ORM + schema migration)
- PostgreSQL
- gRPC (async worker for audit execution)
- JWT Authentication + Role-based Authorization
- Docker & Kubernetes Orchestration (Containerized microservices)

## Architecture

This project uses a layered design so each part has one responsibility:

- **Transport layer**: `internal/transport/http.go`  
  Owns HTTP routes, parses path/body/query params, returns JSON responses.
- **Endpoint layer**: `internal/endpoint/endpoint.go`  
  go-kit adapter layer. Converts transport requests into service calls.
- **Service layer**: `internal/service/service.go`  
  Business logic (validation, async run dispatch, policy orchestration, issue creation, auth/token generation).
- **Repository layer**: `internal/repository/repository.go`  
  Data-access layer for persistence and query execution.
  - Uses **Ent** for CRUD on our own entities (`audits`, `audit_runs`, `issues`, `policies`, `policy_runs`, `users`).
  - Uses `database/sql` for executing the dynamic audit SQL query itself.
- **Schema layer**: `ent/schema/*.go`  
  Table definitions; Ent generates DB/query code from these schemas.

Design notes are documented in `docs/architecture.md`.

## Request Flow Example (`POST /audits/{id}/run`) - Async (Milestone 3)

1. Route `/audits/{id}/run` matches in `internal/transport/http.go`.
2. Transport decoder extracts `id` into a typed request.
3. Endpoint `RunAudit` in `internal/endpoint/endpoint.go` is called.
4. Endpoint calls `service.RunAudit(...)`.
5. Service layer:
   - fetches audit definition from repository
   - creates an `audit_run` row with `running`
   - dispatches `audit_id` + `run_id` to gRPC worker
   - returns immediately to caller with accepted status
6. gRPC worker receives request and executes audit in background:
   - runs all query rules in `queries`
   - compares each query's `actual` vs `expected` based on expected type
   - updates run status to `passed`, `failed`, or `error`
   - creates an `issue` per failed query (with `query_name`)
7. `GET /audits/{id}/run/{run_id}/status` reflects latest async state.

## Setup Instructions

### 1) Prerequisites

- Go installed (same major version as `go.mod` or newer)
- PostgreSQL running locally

### 2) Create database

```bash
createdb security_central
```

### 3) Configure database connection

Default DSN used by app (if `DATABASE_URL` is not set):

```text
postgres://postgres:postgres@localhost:5432/security_central?sslmode=disable
```

If your username/password/port differs, set:

```bash
export DATABASE_URL='postgres://<user>:<pass>@localhost:<port>/security_central?sslmode=disable'
```

### 4) Install dependencies

```bash
go mod tidy
```

### 5) Start gRPC worker (required for async run)

```bash
go run ./cmd/worker
```

Worker listens on `:9090` by default (or `WORKER_PORT` env var).

### 6) Start API server

```bash
go run ./cmd/api
```

Server starts on `http://localhost:8080` by default (or `PORT` env var if set).  
API connects to worker at `AUDIT_WORKER_ADDR` (default `localhost:9090`).

On startup, the API seeds default users if they do not exist:
- admin / `admin123` (role: `admin`)
- viewer / `viewer123` (role: `viewer`)

### 7) (Optional) Seed example table for demo audit

The service auto-creates tables from Ent schemas on startup, including:
- `audits`
- `audit_runs`
- `issues`
- `vm_inventory`
- `policies`
- `policy_runs`
- `users`

## Kubernetes Deployment

### Apply and restart

```bash
# Apply all Kubernetes manifests
kubectl apply -f k8s/

# Restart deployments to pull fresh local images
kubectl rollout restart deployment/api
kubectl rollout restart deployment/worker
```

### Monitor startup

```bash
# Monitor cluster startup until all pods show 1/1 Running
kubectl get pods -w
```

### Deployment endpoint

Use port-forward to expose the API service locally:

```bash
kubectl port-forward svc/api-service 8080:8080
```

Once port-forward is active, your deployment endpoint is:
- `http://localhost:8080`

All API routes in this README work against that endpoint.

### Access PostgreSQL in cluster

```bash
# Connect directly via psql shell inside the pod
kubectl exec -it deployment/postgres -- psql -U postgres -d security_central
```

## Run APIs Manually

### Auth Endpoint (Milestone 5)

- `POST /login`

### Audit + Issue Endpoints

- `POST /audits`
- `GET /audits/{id}`
- `POST /audits/{id}/run`
- `GET /audits/{id}/run/{run_id}/status`
- `GET /issues/list?page=1&page_size=20`
- `GET /issues/{id}`

### Policy Endpoints (Milestone 4)

- `POST /policies`
- `GET /policies/{id}`
- `POST /policies/{id}/run`
- `GET /policies/{id}/run/{run_id}/status`

### Scheduler Endpoint (Bonus Milestone)

- `POST /schedule`

All responses are JSON.

### Milestone 3 Async Behavior

- `POST /audits/{id}/run` now triggers execution asynchronously via gRPC worker.
- API returns immediately with:
  - `run.status = "running"` when dispatch succeeds
  - `accepted = true`
- Poll `GET /audits/{id}/run/{run_id}/status` until status is `passed`, `failed`, or `error`.

### Milestone 4 Policy Behavior

- `POST /policies/{id}/run` creates one child audit run per audit inside the policy.
- Each child run is dispatched to the same gRPC audit worker used by Milestone 3.
- Policy status is aggregated from child audit runs:
  - `passed` when all child runs pass
  - `failed` when at least one child run fails (and none error)
  - `error` when at least one child run errors
  - `running` while any child run is still running
- `GET /policies/{id}/run/{run_id}/status` returns:
  - policy run metadata (`status`, `started_at`, `completed_at`, `audit_run_ids`)
  - expanded `audit_runs` array with child run status details

### Milestone 5 Authentication & Authorization

- `/login` is public (no token required).
- All other endpoints require `Authorization: Bearer <jwt-token>`.
- JWT includes username and role claims and expires in 24 hours.
- Role rules:
  - `admin`: can call all GET/POST endpoints.
  - `viewer`: read-only; can call GET endpoints only.
- POST requests from non-admin users return `403 Forbidden`.

Note: passwords are base64-encoded to satisfy assignment requirements.  
This is not secure hashing and should be replaced with a real password hash (e.g. bcrypt/argon2) in production.

### Scheduler Behavior (Bonus Milestone)

- `POST /schedule` creates a recurring schedule for either:
  - `target_type = "audit"` with `target_id = <audit_id>`
  - `target_type = "policy"` with `target_id = <policy_id>`
- `interval_seconds` controls how often the target is executed.
- Scheduler loop checks for due schedules periodically and triggers:
  - `RunAudit(...)` for audit targets
  - `RunPolicy(...)` for policy targets
- After execution, `next_run_at` is advanced by `interval_seconds`.

### Sample request: create audit (multi-query)

```bash
curl -X POST http://localhost:8080/audits \
  -H "Content-Type: application/json" \
  -d '{
    "name":"Memory Compliance",
    "queries":[
      {
        "name":"vms_over_50_memory",
        "sql_query":"SELECT count(*) FROM vm_inventory WHERE memory > 50",
        "expected_result":{"type":"int","value":"0"}
      },
      {
        "name":"powered_off_vms",
        "sql_query":"SELECT count(*) FROM vm_inventory WHERE power_state = '\''off'\''",
        "expected_result":{"type":"int","value":"0"}
      }
    ]
  }'
```

### Sample request: run audit

```bash
curl -X POST http://localhost:8080/audits/1/run
```

Example response:

```json
{
  "run": {
    "id": 12,
    "audit_id": 1,
    "status": "running",
    "started_at": "2026-08-14T15:30:00Z"
  },
  "accepted": true,
  "message": "audit run accepted for asynchronous execution"
}
```

### Sample request: check run status

```bash
curl http://localhost:8080/audits/1/run/1/status
```

### Sample request: list issues

```bash
curl "http://localhost:8080/issues/list?page=1&page_size=20"
```

### Sample request: create policy

```bash
curl -X POST http://localhost:8080/policies \
  -H "Content-Type: application/json" \
  -d '{
    "name":"baseline_security_policy",
    "audit_ids":[1,2]
  }'
```

### Sample request: run policy

```bash
curl -X POST http://localhost:8080/policies/1/run
```

### Sample request: policy run status

```bash
curl http://localhost:8080/policies/1/run/1/status
```

### Sample request: create schedule

```bash
TOKEN="<admin-jwt-token>"
curl -X POST http://localhost:8080/schedule \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "target_type":"audit",
    "target_id":1,
    "interval_seconds":300
  }'
```

### Sample request: login (admin)

```bash
curl -X POST http://localhost:8080/login \
  -H "Content-Type: application/json" \
  -d '{
    "username":"admin",
    "password":"admin123"
  }'
```

Example response:

```json
{
  "token": "<jwt-token>"
}
```

### Sample request: call protected endpoint with JWT

```bash
TOKEN="<jwt-token>"
curl http://localhost:8080/issues/list \
  -H "Authorization: Bearer ${TOKEN}"
```