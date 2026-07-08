---

description: "Task list template for feature implementation"
---

# Tasks: Push Notification Latency Measurement

**Input**: Design documents from `/specs/001-push-notification-latency/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/openapi.yaml, quickstart.md

**Tests**: Server-side tests are explicitly requested (user: "add test coverage for the server, no need for test coverage on the android app") and are included for every server user-story phase. No Android test tasks are included by design.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

Mobile + API layout per plan.md: `server/` (Go coordinator) and `android/` (Kotlin client) as sibling projects at repo root.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [X] T001 Create top-level directory structure per plan.md: `server/cmd/coordinator/`, `server/internal/{api,store,push,model,config}/`, `server/migrations/`, `android/app/src/main/java/com/tw/pushlatency/`, `.github/workflows/`
- [X] T002 Initialize Go module in `server/go.mod` (module path, Go 1.22) with dependencies `modernc.org/sqlite` and `firebase.google.com/go/v4`
- [X] T003 [P] Initialize Android Gradle project (`android/settings.gradle.kts`, `android/build.gradle.kts`, `android/app/build.gradle.kts`) with min SDK 26, Firebase BoM, `firebase-messaging`, `androidx.work:work-runtime-ktx`, Room, and the `google-services` plugin applied
- [X] T004 [P] Add `server/.golangci.yml` golangci-lint configuration
- [X] T005 Create root `Makefile` with `build`, `build-server`, `build-android`, `test`, `test-server`, `lint`, `lint-server`, `lint-android` targets (`test` and `test-server` are the same target; there is no `test-android` per scope)

**Checkpoint**: Repo skeleton and toolchains in place.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T006 Write SQLite schema migration `server/migrations/0001_init.sql` creating `devices`, `rounds`, and `deliveries` tables per data-model.md (deliveries PK = (round_id, device_id))
- [X] T007 Implement DB bootstrap + migration runner in `server/internal/store/db.go` (opens `modernc.org/sqlite` at `DB_PATH`, applies pending migrations from `server/migrations/` on startup)
- [X] T008 [P] Define domain types `Device`, `Round`, `Delivery`, `DeliveryStatus` in `server/internal/model/model.go` per data-model.md
- [X] T009 [P] Define the generic `PushSender` interface (`Send(ctx, device Device, roundID string) error`) in `server/internal/push/sender.go` (FR-016 — platform-agnostic boundary)
- [X] T010 Implement configuration loading from environment (`DB_PATH`, `OPERATOR_TOKEN`, `FCM_SERVICE_ACCOUNT_JSON`, `LISTEN_ADDR`) in `server/internal/config/config.go`
- [X] T011 Implement HTTP server bootstrap, `net/http` `ServeMux` routing skeleton, and a `/healthz` endpoint in `server/cmd/coordinator/main.go`
- [X] T012 [P] Implement bearer-token auth middleware (validates against `OPERATOR_TOKEN`) in `server/internal/api/auth.go` (FR-029)
- [X] T013 [P] Scaffold `MainActivity` (empty list container + empty-state view, FR-025) and `Application` class in `android/app/src/main/java/com/tw/pushlatency/MainActivity.kt`
- [X] T014 [P] Define Room `MessageEntity` + DAO for the persisted on-screen log in `android/app/src/main/java/com/tw/pushlatency/data/MessageDao.kt` and `android/app/src/main/java/com/tw/pushlatency/data/MessageEntity.kt`

**Checkpoint**: Foundation ready — user story implementation can now begin.

---

## Phase 3: User Story 1 - Measure round-trip delivery latency across a device fleet (Priority: P1) 🎯 MVP

**Goal**: An operator can trigger a round, have it dispatched via FCM to the fleet, and see per-device receipt/latency recorded as devices report back.

**Independent Test**: Enroll a device (via `POST /v1/devices`), trigger a round, and confirm a delivery record appears with sent time, received time, and computed latency once the device reports receipt.

### Tests for User Story 1 ⚠️

> Write these tests FIRST, ensure they FAIL before implementation

- [X] T015 [P] [US1] Store test: round creation + one pending Delivery per targeted device in `server/internal/store/round_store_test.go`
- [X] T016 [P] [US1] Store test: delivery receipt recording, latency computation, and idempotent duplicate reports (FR-012) in `server/internal/store/delivery_store_test.go`
- [X] T017 [P] [US1] API test: `POST /v1/rounds` requires bearer auth, returns 202 with round_id/sent_at/targeted_device_count in `server/internal/api/rounds_test.go`
- [X] T018 [P] [US1] API test: `POST /v1/rounds/{round_id}/receipts` handles unknown round/device gracefully (FR-011) and is retry-safe (FR-012) in `server/internal/api/receipts_test.go`
- [X] T019 [P] [US1] Test: FCM payload construction sets `Android.TTL = 0` and high priority (FR-005), using a fake `messaging.Client` transport in `server/internal/push/fcm_test.go`

### Implementation for User Story 1

- [X] T020 [P] [US1] Implement round store: create round with unique `public_id`, snapshot `targeted_device_count`, insert one `pending` Delivery per current device in `server/internal/store/round_store.go` (depends on T006-T008)
- [X] T021 [P] [US1] Implement delivery store: record receipt keyed by (round_id, device_id), compute `latency_ms`, classify `received_good`/`received_anomalous`, no-op on duplicate in `server/internal/store/delivery_store.go` (depends on T006-T008)
- [X] T022 [US1] Implement FCM `PushSender` (data message, `Android.TTL=0`, high priority, payload = round `public_id`) in `server/internal/push/fcm.go` (depends on T009, T010)
- [X] T023 [US1] Implement `POST /v1/rounds` handler: auth check, create round via store, fan out sends concurrently via `PushSender`, return 202 immediately (FR-008) in `server/internal/api/rounds.go` (depends on T020, T022, T012)
- [X] T024 [US1] Implement `POST /v1/rounds/{round_id}/receipts` handler: validate round/device exist, delegate to delivery store, return `recorded`/`duplicate_ignored`/`invalid` in `server/internal/api/receipts.go` (depends on T021)
- [X] T025 [US1] Wire `/v1/rounds` and `/v1/rounds/{round_id}/receipts` routes into `server/cmd/coordinator/main.go` (depends on T023, T024)
- [X] T026 [P] [US1] Implement `PushMessagingService.onMessageReceived`: capture arrival timestamp, persist to Room log in `android/app/src/main/java/com/tw/pushlatency/messaging/PushMessagingService.kt` (depends on T014)
- [X] T027 [US1] Implement `ReceiptReportWorker` (WorkManager, exponential backoff retry, calls `POST /v1/rounds/{round_id}/receipts`) in `android/app/src/main/java/com/tw/pushlatency/receipt/ReceiptReportWorker.kt` (depends on T026; FR-023)
- [X] T028 [US1] Enqueue `ReceiptReportWorker` from `onMessageReceived` in `android/app/src/main/java/com/tw/pushlatency/messaging/PushMessagingService.kt` (depends on T026, T027)
- [X] T029 [US1] Implement on-screen message list (most-recent-first, latency categorized good/slow/anomalous per FR-026, empty state) in `android/app/src/main/java/com/tw/pushlatency/MainActivity.kt` (depends on T013, T014)

**Checkpoint**: User Story 1 is fully functional and independently testable/demoable.

---

## Phase 4: User Story 2 - Automatic, duplicate-free fleet enrollment (Priority: P2)

**Goal**: A device enrolls itself automatically on first launch, recognizes existing enrollment, and re-enrolls (updating, not duplicating) when its push token rotates.

**Independent Test**: Open the app to enroll a device, confirm exactly one fleet entry; simulate a token rotation and confirm the same entry is updated, not duplicated.

### Tests for User Story 2 ⚠️

- [X] T030 [P] [US2] Store test: device upsert-by-`install_id` semantics — new enrollment vs. token update on existing row (FR-002) in `server/internal/store/device_store_test.go`
- [X] T031 [P] [US2] API test: `POST /v1/devices` for new enrollment, re-enrollment/update, and malformed-request rejection in `server/internal/api/devices_test.go`

### Implementation for User Story 2

- [X] T032 [P] [US2] Implement device store: upsert by `install_id`, update `push_token`/`updated_at` on existing row in `server/internal/store/device_store.go` (depends on T006-T008)
- [X] T033 [US2] Implement `POST /v1/devices` handler (open, no auth, per spec.md §8) in `server/internal/api/devices.go` (depends on T032)
- [X] T034 [US2] Wire `/v1/devices` route into `server/cmd/coordinator/main.go` (depends on T033)
- [X] T035 [P] [US2] Implement `InstallIdProvider`: generate a UUID once, persist locally, reuse thereafter (Clarification Q1) in `android/app/src/main/java/com/tw/pushlatency/enrollment/InstallIdProvider.kt`
- [X] T036 [US2] Implement `EnrollmentManager`: call `POST /v1/devices` on first launch, skip with "already enrolled" status if a valid local record exists (FR-017, FR-018) in `android/app/src/main/java/com/tw/pushlatency/enrollment/EnrollmentManager.kt` (depends on T035)
- [X] T037 [US2] Override `onNewToken` to re-enroll via `EnrollmentManager` with the rotated token, same `install_id` (FR-019) in `android/app/src/main/java/com/tw/pushlatency/messaging/PushMessagingService.kt` (depends on T036)
- [X] T038 [US2] Surface enrollment failures via a non-blocking Snackbar/status indicator (FR-020) in `android/app/src/main/java/com/tw/pushlatency/MainActivity.kt` (depends on T036)

**Checkpoint**: User Stories 1 AND 2 both work independently.

---

## Phase 5: User Story 3 - Review aggregate delivery health and identify slow or failed deliveries (Priority: P3)

**Goal**: An operator can review success rate and latency trends across many rounds, sliced by send time, and find the slowest individual deliveries.

**Independent Test**: After several rounds with a mix of received/never-received outcomes, request the report endpoints and confirm success rate and slowest-deliveries are visible.

### Tests for User Story 3 ⚠️

- [X] T039 [P] [US3] Store test: deliveries query filtering by `round_id`, ordering by latency/sent_at, and limit (FR-013) in `server/internal/store/delivery_store_test.go`
- [X] T040 [P] [US3] Store test: success-rate bucket aggregation by hour/day with `since`/`until` bounds (FR-014) in `server/internal/store/report_store_test.go`
- [X] T041 [P] [US3] API test: `GET /v1/reports/deliveries` and `GET /v1/reports/success-rate` request/response shapes and param validation in `server/internal/api/reports_test.go`

### Implementation for User Story 3

- [X] T042 [P] [US3] Add deliveries query (filter/order/limit) to `server/internal/store/delivery_store.go` (depends on T021)
- [X] T043 [P] [US3] Implement success-rate bucket aggregation query in `server/internal/store/report_store.go` (depends on T006-T008)
- [X] T044 [US3] Implement `GET /v1/reports/deliveries` handler in `server/internal/api/reports.go` (depends on T042)
- [X] T045 [US3] Implement `GET /v1/reports/success-rate` handler in `server/internal/api/reports.go` (depends on T043)
- [X] T046 [US3] Wire `/v1/reports/deliveries` and `/v1/reports/success-rate` routes into `server/cmd/coordinator/main.go` (depends on T044, T045)

**Checkpoint**: All three user stories are independently functional.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Build/test/lint automation and publish pipelines requested for both sides of the project

- [X] T047 [P] Finalize `Makefile` targets: `build-server` → `go build ./...`; `build-android` → `./gradlew :app:assembleRelease`; `test`/`test-server` → `go test ./... -cover`; `lint-server` → `golangci-lint run`; `lint-android` → `./gradlew :app:lint`
- [X] T048 [P] Write multi-stage `server/Dockerfile` (`CGO_ENABLED=0` static Go build → minimal runtime image, e.g. `scratch`/`distroless`)
- [X] T049 [P] Create `.github/workflows/server-release.yml`: run `make lint-server` and `make test-server`, then build and push the Docker image to GHCR on push to main/tag
- [X] T050 [P] Create `.github/workflows/android-release.yml`: run `make lint-android`, then build a signed release APK/AAB (keystore + credentials from repository secrets, FR-030) and attach it to a GitHub Release on tag
- [X] T051 Run `quickstart.md` end-to-end validation (enroll → trigger round → device receives & reports → reports reflect it) against a locally built server and app
- [X] T052 [P] Write root `README.md` documenting server environment variables, Android per-build configuration (FR-028), and how to schedule repeated rounds (FR-003)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately
- **Foundational (Phase 2)**: Depends on Setup — BLOCKS all user stories
- **User Story 1 (Phase 3)**: Depends on Foundational only
- **User Story 2 (Phase 4)**: Depends on Foundational only (independent of US1's round/receipt logic, though it shares the `devices` table created in T006)
- **User Story 3 (Phase 5)**: Depends on Foundational; its store/report queries read data produced by US1 (rounds/deliveries) to be meaningful, but the endpoints themselves can be implemented and unit-tested independently
- **Polish (Phase 6)**: Depends on all desired user stories being complete

### Within Each User Story

- Tests written first, confirmed failing, before implementation
- Store layer before API handlers before route wiring
- Android background reception before receipt reporting before UI display

### Parallel Opportunities

- T003, T004 (Setup) can run alongside T002
- T008, T009 (Foundational) can run in parallel; T012, T013, T014 likewise
- All `[P]`-marked tests within a story phase can be written in parallel (distinct files)
- T020/T021 (round store / delivery store) can be implemented in parallel; likewise T042/T043 in US3
- US1 and US2 implementation can proceed in parallel once Foundational is done (US1 touches rounds/deliveries/push; US2 touches devices/enrollment) — they share only the `devices` table schema already fixed in T006
- T047-T050 (Polish) can all run in parallel

---

## Parallel Example: User Story 1

```bash
# Tests together:
Task: "Store test: round creation + pending deliveries in server/internal/store/round_store_test.go"
Task: "Store test: receipt recording + idempotency in server/internal/store/delivery_store_test.go"
Task: "API test: POST /v1/rounds auth + response shape in server/internal/api/rounds_test.go"
Task: "API test: POST /v1/rounds/{round_id}/receipts graceful handling in server/internal/api/receipts_test.go"
Task: "FCM payload TTL/priority test in server/internal/push/fcm_test.go"

# Store implementations together:
Task: "Round store in server/internal/store/round_store.go"
Task: "Delivery store in server/internal/store/delivery_store.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (blocks everything)
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: trigger a round against a real enrolled device (manually `curl`-enrolled if US2 isn't built yet), confirm latency is recorded
5. Demo: this alone answers the core question the tool exists for

### Incremental Delivery

1. Setup + Foundational → foundation ready
2. User Story 1 → validate → MVP demo (round trigger + latency measurement, manual enrollment via curl)
3. User Story 2 → validate → real automatic enrollment replaces manual curl step
4. User Story 3 → validate → aggregate reporting available
5. Polish → Makefile, Docker/CI publishing, docs

### Total Task Count

52 tasks: 5 Setup, 9 Foundational, 15 US1 (5 tests + 10 impl), 9 US2 (2 tests + 7 impl), 8 US3 (3 tests + 5 impl), 6 Polish.
