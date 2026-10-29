# FieldNotes Go service

Requires Go 1.26 (verified with 1.26.1). Standard library only; no external modules.

Run from this directory:

```sh
go run ./cmd/fieldnotes
```

The service listens on `:8080` and creates `../data/issues.json` on the first successful write. An empty list is returned until an issue is explicitly created. A missing data file is valid; a malformed or incompatible existing file stops startup without overwriting it. Stored documents require explicit version, issue array, and idempotency object; stored positions require every coordinate, and retry snapshots must match their issue's immutable fields.

| Environment variable | Default | Meaning |
| --- | --- | --- |
| `ADDR` | `:8080` | HTTP listen address; use `127.0.0.1:8080` for local access only |
| `DATA_FILE` | `../data/issues.json` | JSON data file, resolved relative to the process working directory |
| `CORS_ORIGINS` | `http://localhost:5173` | Comma-separated browser origins, or `*` |
| `WEB_DIR` | unset | React build directory; enables assets and SPA navigation fallback |

For a built dashboard, from `server/`:

```sh
WEB_DIR=../web/dist go run ./cmd/fieldnotes
```

A phone uses `http://<computer-LAN-IP>:8080`; `localhost` on the phone refers to the phone. Native Unity requests do not need an Origin header. Browser origins must match the configured scheme, host, and port. The API has no authentication: use a trusted local network for this prototype.

API contract is in `../docs/SPEC.md`. JSON write requests require `Content-Type: application/json`. Request bodies are limited to 64 KiB. Every position coordinate must be present and finite. Title and description are trimmed before storage; title is required and limited to 120 Unicode code points, while optional description is limited to 2000. Limits apply after trimming and are also checked when loading persisted data. Unknown fields and additional JSON values are rejected. Errors use `{ "error": { "code": "...", "message": "...", "fields": {} } }`, with optional `fields` for validation.

A POST may include a UUID `Idempotency-Key`. Retrying the same decoded request with the same key returns the original creation response with HTTP 201, including after a restart or later status changes. Reusing that key with changed values returns 409. JSON key order and whitespace do not affect matching; field values do. Keep the same key when retrying an uncertain submission, and generate a fresh key for each new issue.

Persistence serializes each complete mutation under a mutex, writes a private temporary file in the data directory, syncs and closes it, atomically renames it, then publishes the new in-memory state. Failed pre-commit writes do not alter memory or reserve an idempotency key. Directory sync after rename is best-effort. Run only **one server process per data file**. The file, retry records, and full rewrite cost grow with issue count; this is a small local prototype, not a multi-user database. Back up the file before maintenance; never edit it while the server is running.

The HTTP server has header/read/write/idle timeouts and a ten-second graceful shutdown on SIGINT or SIGTERM. The health response contains only `{"status":"ok"}`. Unknown `/api` routes always return JSON, including with SPA hosting enabled.

Verify:

```sh
go test -race ./...
go vet ./...
go build ./cmd/fieldnotes
```

The tests cover create/list/status/restart, server IDs and UTC times, strict validation, HTTP errors, CORS, idempotency across restart and concurrent retries, distinct concurrent writes, deterministic disk-write failures, corrupted startup data, clock adjustments, and SPA routing. They simulate HTTP and filesystem behavior; they do not validate iPhone AR tracking or a production network deployment.
