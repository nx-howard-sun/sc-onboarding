# ByteSized Security Central (Go)

Implementation using:
- Go
- go-kit (service/endpoint/transport layering)
- Ent (ORM + schema migration)
- PostgreSQL

## Architecture

This project uses a layered design so each part has one responsibility:

- **Transport layer**: `internal/transport/http.go`  
  Owns HTTP routes, parses path/body/query params, returns JSON responses.
- **Endpoint layer**: `internal/endpoint/endpoint.go`  
  go-kit adapter layer. Converts transport requests into service calls.
- **Service layer**: `internal/service/service.go`  
  Business logic (validation, run/evaluate audit, create issue).
- **Repository layer**: `internal/repository/repository.go`  
  Data-access layer for persistence and query execution.
  - Uses **Ent** for CRUD on our own entities (`audits`, `audit_runs`, `issues`).
  - Uses `database/sql` for executing the dynamic audit SQL query itself.
- **Schema layer**: `ent/schema/*.go`  
  Table definitions; Ent generates DB/query code from these schemas.

Design notes are documented in `docs/architecture.md`.

## Request Flow Example (`POST /audits/{id}/run`)

1. Route `/audits/{id}/run` matches in `internal/transport/http.go`.
2. Transport decoder extracts `id` into a typed request.
3. Endpoint `RunAudit` in `internal/endpoint/endpoint.go` is called.
4. Endpoint calls `service.RunAudit(...)`.
5. Service layer:
   - fetches audit definition from repository
   - creates an `audit_run` row with `running`
   - executes all query rules in `queries`
   - compares each query's `actual` vs `expected` based on expected type
   - updates run status to `passed`, `failed`, or `error`
   - creates an `issue` per failed query (with `query_name`)
6. Transport encodes final response as JSON.

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

### 5) Start API server

```bash
go run ./cmd/api
```

Server starts on `http://localhost:8080` by default (or `PORT` env var if set).

### 6) (Optional) Seed example table for demo audit

The service auto-creates tables from Ent schemas on startup, including:
- `audits`
- `audit_runs`
- `issues`
- `vm_inventory`

## Run APIs Manually

### Milestone 1 Endpoints

- `POST /audits`
- `GET /audits/{id}`
- `POST /audits/{id}/run`
- `GET /audits/{id}/run/{run_id}/status`
- `GET /issues/list?page=1&page_size=20`
- `GET /issues/{id}`

All responses are JSON.

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

### Sample request: check run status

```bash
curl http://localhost:8080/audits/1/run/1/status
```

### Sample request: list issues

```bash
curl "http://localhost:8080/issues/list?page=1&page_size=20"
```