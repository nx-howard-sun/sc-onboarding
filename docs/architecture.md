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

## SQL Execution Safety Assumptions (Milestone 1)
- Only `SELECT` queries are allowed for audit SQL.
- Single statement only (no semicolon-delimited multi-statements).
- Query result is expected to be a single scalar value (1 row, 1 column).
- Evaluation is type-aware based on each query's `expected_result.type`.

## API Mapping (Milestone 1)
- `POST /audits` -> create audit
- `GET /audits/{id}` -> fetch audit
- `POST /audits/{id}/run` -> asynchronous dispatch to gRPC worker
- `GET /audits/{id}/run/{run_id}/status` -> run status/details
- `GET /issues/list` -> paginated issue list
- `GET /issues/{id}` -> issue details

## Extendability by Milestone
- Milestone 2: move `queries` JSON into normalized `audit_queries` table if you need stronger query-level indexing/history.
- Milestone 3: implemented gRPC worker for asynchronous execution.
- Milestone 4: add `policies`, `policy_audits`, and policy run aggregate status.
- Milestone 5: add auth middleware in transport layer and role checks per endpoint.
