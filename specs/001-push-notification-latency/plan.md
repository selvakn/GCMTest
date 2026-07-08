# Implementation Plan: Push Notification Latency Measurement

**Branch**: `001-push-notification-latency` | **Date**: 2026-07-08 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/001-push-notification-latency/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Build a two-sided system that measures push notification delivery latency: an Android client (FCM) that auto-enrolls, receives timed test notifications in the background, and shows a categorized on-screen log; and a Go + SQLite coordinator service that triggers rounds, records per-device delivery outcomes (including permanent "never received" for offline devices), and exposes latency/success-rate reporting — built generically enough (a `Platform` field and a `PushSender` interface) that iOS/Windows can be added later without touching round-triggering or reporting logic.

## Technical Context

**Language/Version**: Go 1.22+ (server); Kotlin 1.9+ / Android Gradle Plugin, min SDK 26 (server-side/client-side chosen here, not spec-mandated)
**Primary Dependencies**: Server: stdlib `net/http` (Go 1.22 routing), `database/sql` + `modernc.org/sqlite`, `firebase.google.com/go/v4/messaging` (FCM Admin SDK). Android: Firebase Cloud Messaging SDK, `WorkManager` (retryable receipt reporting), Room (persisted local message log)
**Storage**: SQLite, single file, plain-SQL migrations applied on server startup
**Testing**: Server: Go `testing` + `net/http/httptest` + table-driven tests against a real temp-file/`:memory:` SQLite DB, `go test -cover`. Android: no test coverage required for this iteration (explicit user instruction) — `./gradlew lint` only.
**Target Platform**: Linux container (server, published as a Docker image); Android 8.0+ (API 26+) devices/emulators (client)
**Project Type**: Mobile + API (two sibling projects: `server/` and `android/`)
**Performance Goals**: Round-trigger confirmation < 2s (SC-001); 95% of individual delivery latencies visible in reporting within 30s of actual device receipt (SC-002); fleet size < 50 concurrently enrolled devices (Clarification Q2) — no high-throughput fan-out engineering required
**Constraints**: FCM messages sent with `TTL=0` / high priority so unreachable devices are dropped, never queued (FR-005); round-trigger endpoint requires bearer-token auth (FR-029); device enrollment endpoint is open, no auth (spec.md §8); all push-provider credentials and operator tokens supplied via environment/secrets, never committed (FR-030)
**Scale/Scope**: < 50 devices; unbounded round/delivery history accumulation (no retention/aging-out policy in this iteration, per spec Assumptions)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

`.specify/memory/constitution.md` contains only unfilled template placeholders — no project-specific principles have been ratified. There are no constitutional gates to evaluate against. Standard engineering practices implied directly by the spec (secrets never committed — FR-030; round-triggering restricted to authorized callers — FR-029; graceful handling of malformed/duplicate input — FR-011/FR-012) are treated as hard requirements via the functional requirements themselves, not a separate gate.

**Result**: PASS (no gates defined; spec-level requirements carried through to design below).

## Project Structure

### Documentation (this feature)

```text
specs/001-push-notification-latency/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output (/speckit-plan command)
├── data-model.md        # Phase 1 output (/speckit-plan command)
├── quickstart.md        # Phase 1 output (/speckit-plan command)
├── contracts/
│   └── openapi.yaml     # Phase 1 output (/speckit-plan command)
└── tasks.md             # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)

```text
server/
├── cmd/
│   └── coordinator/
│       └── main.go            # entrypoint: load config, run migrations, start HTTP server
├── internal/
│   ├── api/                   # HTTP handlers + routing (net/http ServeMux)
│   │   ├── devices.go
│   │   ├── rounds.go
│   │   ├── receipts.go
│   │   ├── reports.go
│   │   └── *_test.go
│   ├── store/                 # SQLite persistence (database/sql + modernc.org/sqlite)
│   │   ├── device_store.go
│   │   ├── round_store.go
│   │   ├── delivery_store.go
│   │   └── *_test.go
│   ├── push/                  # generic PushSender interface + FCM implementation (FR-016)
│   │   ├── sender.go
│   │   ├── fcm.go
│   │   └── fcm_test.go
│   └── model/                 # Device, Round, Delivery domain types
├── migrations/
│   └── 0001_init.sql
├── Dockerfile
├── go.mod
└── go.sum

android/
├── app/
│   ├── src/main/java/com/tw/pushlatency/
│   │   ├── MainActivity.kt              # on-screen message log (RecyclerView/Compose)
│   │   ├── enrollment/                  # install_id generation, enroll/re-enroll on launch
│   │   ├── messaging/PushMessagingService.kt  # FirebaseMessagingService: onMessageReceived, onNewToken
│   │   ├── receipt/ReceiptReportWorker.kt     # WorkManager job: report receipt, retries on failure
│   │   ├── data/                        # Room entities/DAO for persisted message log
│   │   └── network/CoordinatorApi.kt    # HTTP client for coordinator endpoints
│   ├── src/main/res/
│   └── build.gradle.kts
├── build.gradle.kts
├── settings.gradle.kts
└── gradle/

Makefile
.github/workflows/
├── server-release.yml   # lint + test + build + publish Docker image (GHCR)
└── android-release.yml  # lint + build + publish APK/AAB as a GitHub Release asset
```

**Structure Decision**: Mobile + API layout (Option 3) — `server/` and `android/` as independent sibling projects at the repo root, each with its own build tooling, unified by a root `Makefile` and by two separate GitHub Actions publish workflows, per the user's explicit request.

## Complexity Tracking

*No constitution gates were violated (none are defined); this section is not applicable.*
