# FieldNotes Implementation Plan

此文件保留立项时的实施计划，以下复选框为原始计划内容。当前完成状态、测试证据和待办事项以根目录 README、docs/VERIFICATION.md 与 docs/SUBMISSION.md 为准。

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a truthful, testable Unity AR → Go → React inspection prototype for iPhone 15 Pro Max.

**Architecture:** Go owns the shared Issue contract and a single-process atomic JSON store. React uses the same REST API as Unity, with a Vite proxy in development and optional Go static serving in production. Unity scene generation creates real ARSession/XROrigin/plane/raycast components and uGUI; no fake 3D fallback is presented as AR.

**Tech Stack:** Go 1.26 (standard library), React 19/TypeScript/Vite, Unity 6 LTS and matching AR Foundation/ARKit 6.0.8.

---

### Task 1: Shared contract and reliable API

Files: `server/go.mod`, `server/cmd/fieldnotes/main.go`, `server/internal/inspection/{model,store,http}.go`, matching `_test.go`.

- [ ] Write HTTP and store tests first: create/list/update/reopen; whitespace title; bad enums; malformed/oversized JSON; missing/nonfinite coordinates; not found; CORS; concurrency; failed write rollback; corrupted startup rejection; idempotent retry and mismatch.
- [ ] Run `cd server && go test ./...`, confirm missing implementation failure.
- [ ] Implement contract in `docs/SPEC.md`; cap body at 64 KiB, reject unknown/trailing JSON, validate complete position and string limits. Do not leak internal filesystem errors.
- [ ] Store issue and idempotency metadata together, lock mutations, persist before publish, crypto-random IDs and UTC RFC3339 timestamps. Read corrupt data fails startup rather than overwriting it. Bind `:8080`, configurable DATA_FILE/ADDR/CORS_ORIGINS/WEB_DIR, graceful shutdown and HTTP timeouts; optional static SPA hosting.
- [ ] Run `go test -race ./...` and `go vet ./...`, commit server files only.

### Task 2: React workspace

Files: `web/package.json`, lockfile, `web/src/{App,api,types}.tsx/ts`, CSS, `web/src/*.test.tsx`, Vite/TS config.

- [ ] Build API error/loading/retry and state update tests before implementation (Vitest/Testing Library); verify failing results.
- [ ] Implement real API list with title/priority/status, counts, priority/status/search filters, empty/loading/error states, refresh; select row → accessible details with description, position, timestamps and state control. Preserve selected detail on refresh; disable duplicate updates; server response is authoritative; failed updates keep old status.
- [ ] Restrained Chinese operational UI, responsive table/list, keyboard access, readable text contrast, focus indication, no external image/font dependency and no automatic seed data.
- [ ] Run `npm test -- --run` and `npm run build`, commit web files only.

### Task 3: Unity iOS AR client

Files: `unity/Packages/manifest.json`, `unity/ProjectSettings/ProjectVersion.txt`, `unity/Assets/FieldNotes/Scripts/{IssueModels,IssueApi,InspectionController,RuntimeUI,PlaneOutline}.cs`, `Editor/ProjectSetup.cs`, EditMode tests, bootstrap scene.

- [ ] Write EditMode tests for validation, URL normalization, payload serialization, placement gate (UI, tracking, submission, touch phase), before implementing helpers.
- [ ] Pin compatible packages/editor and provide a committed bootstrap scene with stable meta GUID. Menu and batch setup build a horizontal-plane AR scene with camera/background, XROrigin and raycasting. Add ARKit loader settings and safe-area portrait UI, device unsupported/permission/tracking guidance. Markers sit on hit surface, colors reflect priority.
- [ ] Touch began only, single touch, explicit EventSystem.RaycastAll at touch coordinates; only tracked horizontal plane polygon hits place/reposition a pending marker. Input/actions/settings/submission block placement. Pending marker persists on failure; submitted markers remain; new report resets draft.
- [ ] Endpoint entry saved locally, health check, create with timeout and idempotency key; preserve snapshot on timeout retry, freeze placement during send, validate title/enums/finite coordinates; readable request errors and success ID. No real default production endpoint.
- [ ] Add iOS build/postprocess: camera and local-network purpose strings, LAN HTTP development exception scoped to local networks, arm64/min iOS/portrait, explicit Xcode signing walkthrough; no Apple identity committed.
- [ ] If editor available run EditMode tests and export Xcode project; otherwise independently review package API signatures and mark Unity compilation/device tests unverified, never claim passed.

### Task 4: End-to-end, delivery and documentation

Files: root `README.md`, `AI_LOG.md`, `docs/{DEMO,VERIFICATION,EXPLAIN,SUBMISSION}.md`, `scripts/{dev,verify,smoke,package}.sh`, CI workflow, Dockerfile.

- [ ] Record actual user prompt originals in AI_LOG in order, sensitive details redacted; separately record agent directives/review corrections. Do not invent user feedback or claim user reviewed code. Identify AI-driven review as such and flag user review requirement outstanding.
- [ ] Smoke real server create → list → status → kill/restart same file → list; assert validation and CORS. Isolate disposable test data.
- [ ] Run browser against real Go: load empty list; create explicit smoke issue; refresh/detail/change state/filter; stop backend and verify error/retry. Save evidence screenshot without pretending Unity originated sample.
- [ ] Run all Go/web tests/build; assemble source zip preserving Git history with `scripts/package.sh`; inspect no credentials/build caches/user data. Provide source startup README and two 3–5 minute real recording scripts/checklist, known missing real device/video/public link.
- [ ] Initialize meaningful local commit history, no remote/publish/email without user instruction; final state includes clear paths and exact remaining hands-on steps.
