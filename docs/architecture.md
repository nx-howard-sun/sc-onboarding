# ByteSized Security Central Architecture (Go + go-kit + Ent)

## Goals
- Use Go as the primary backend language.
- Use `go-kit` for transport/endpoint/service layering.
- Use `ent` as ORM and schema migration tool for PostgreSQL.
- Deliver Milestone 1 now, while designing to extend cleanly through Milestones 2-5.

## High-Level Architecture
- `cmd/api`: application bootstrap and dependency wiring.
- `cmd/worker`: gRPC audit execution worker bootstrap.
- `internal/transport/http`: HTTP route registration and request/response codecs.
- `internal/endpoint`: go-kit endpoint adapters.
- `internal/service`: domain business logic (audit lifecycle, async dispatch, run evaluation, issue creation).
- `internal/repository`: persistence interfaces + ent-backed implementation.
- `internal/worker`: gRPC server and client for async audit execution.
- `ent/schema`: database schema definitions and migration source of truth.

Auth-specific flow components:
- `POST /login` endpoint issues JWT tokens.
- HTTP middleware validates JWT for protected routes.
- Middleware enforces RBAC (`admin` can POST/GET, `viewer` is GET-only).

Request flow:
1. HTTP request enters go-kit HTTP server.
2. Transport decoder maps JSON + path params to typed request.
3. Endpoint invokes `RunAudit` service method.
4. Service validates input, creates an `audit_run` row with status `running`, and dispatches to gRPC worker.
5. gRPC worker executes each query, evaluates expected/actual values, updates run status, and creates issues for deviations.
6. Status endpoint reads run state from repository.
7. Transport encodes JSON response.

## Data Model (Milestone 1)
### `audits`
- `id` (int, PK)
- `name` (string)
- `queries` (JSON array of query rules: `name`, `sql_query`, `expected_result{type,value}`)
- `created_at`, `updated_at`

### `audit_runs`
- `id` (int, PK)
- `audit_id` (FK -> audits)
- `status` (`running`, `passed`, `failed`, `error`)
- `actual_value` (nullable string)
- `error_message` (nullable string)
- `started_at`, `completed_at` (nullable), `created_at`, `updated_at`

### `issues`
- `id` (int, PK)
- `audit_id` (FK -> audits)
- `audit_run_id` (FK -> audit_runs)
- `query_name` (string, optional)
- `expected_value` (string)
- `actual_value` (string)
- `description` (string)
- `created_at`, `updated_at`

### `policies`
- `id` (int, PK)
- `name` (string)
- `audit_ids` (JSON array of audit IDs)
- `created_at`, `updated_at`

### `policy_runs`
- `id` (int, PK)
- `policy_id` (FK-like reference to policy ID)
- `status` (`running`, `passed`, `failed`, `error`)
- `audit_run_ids` (JSON array of child audit run IDs)
- `started_at`, `completed_at` (nullable)

### `users`
- `id` (int, PK)
- `username` (string, unique)
- `password` (base64-encoded string per assignment requirement)
- `role` (`admin` or `viewer`)

## SQL Execution Safety Assumptions (Milestone 1)
- Only `SELECT` queries are allowed for audit SQL.
- Single statement only (no semicolon-delimited multi-statements).
- Query result is expected to be a single scalar value (1 row, 1 column).
- Evaluation is type-aware based on each query's `expected_result.type`.

## API Mapping (Milestone 1-5)
- `POST /login` -> authenticate user and return JWT
- `POST /audits` -> create audit
- `GET /audits/{id}` -> fetch audit
- `POST /audits/{id}/run` -> asynchronous dispatch to gRPC worker
- `GET /audits/{id}/run/{run_id}/status` -> run status/details
- `GET /issues/list` -> paginated issue list
- `GET /issues/{id}` -> issue details
- `POST /policies` -> create policy with grouped audit IDs
- `GET /policies/{id}` -> get policy definition
- `POST /policies/{id}/run` -> start policy execution (creates child audit runs + async dispatch)
- `GET /policies/{id}/run/{run_id}/status` -> aggregate policy status + child run details

Authorization behavior:
- All endpoints except `POST /login` require Bearer JWT.
- `viewer` role is restricted from POST endpoints.
- `admin` role can call both GET and POST endpoints.

## Extendability by Milestone
- Milestone 2: move `queries` JSON into normalized `audit_queries` table if you need stronger query-level indexing/history.
- Milestone 3: implemented gRPC worker for asynchronous execution.
- Milestone 4: implemented `policies` + `policy_runs` and policy-level aggregate status evaluation.
- Milestone 5: implemented JWT auth, startup default users, and transport-layer RBAC enforcement.
