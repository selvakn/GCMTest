# Phase 0 Research: Push Notification Latency Measurement

All technology choices below were either specified directly by the user or are resolved here as concrete engineering decisions. No `NEEDS CLARIFICATION` markers remain in Technical Context.

## 1. Server language & HTTP layer

- **Decision**: Go 1.22+, using the standard library `net/http` with its Go 1.22 enhanced `ServeMux` (method + path-pattern routing) rather than a third-party router.
- **Rationale**: The API surface is small (enroll, trigger round, report receipt, two report endpoints). Go 1.22's stdlib mux covers method-based routing and path parameters natively, avoiding an extra dependency for a service this size.
- **Alternatives considered**: `chi` / `gin` — more ergonomic middleware chaining, but unnecessary weight and an extra dependency to lint/update for five routes.

## 2. SQLite driver

- **Decision**: `modernc.org/sqlite` (pure Go, CGO-free SQLite driver behind `database/sql`).
- **Rationale**: A CGO-free driver keeps `Dockerfile` builds and cross-compilation simple (`CGO_ENABLED=0`, static binary, minimal base image) — important since this project also needs a straightforward Docker publish pipeline. Throughput needs are trivial (fleet <50 devices per clarification), so the modest performance gap versus a CGO driver is irrelevant.
- **Alternatives considered**: `mattn/go-sqlite3` — faster and more mature, but requires a C toolchain in the build image and disables trivial static cross-compilation; rejected given the low-throughput, small-fleet scope.

## 3. Push delivery (FCM)

- **Decision**: Firebase Admin SDK for Go (`firebase.google.com/go/v4/messaging`), sending each round as a **data message** with `Android.TTL = 0` and `Android.Priority = "high"`.
- **Rationale**: `TTL = 0` instructs FCM to drop the message immediately if the device isn't currently connected, rather than queuing it — this is the direct mechanism for FR-005's "fresh or dropped, no queuing" requirement. The Admin SDK handles service-account OAuth2 token refresh, retries, and the HTTP v1 payload shape, which would otherwise be reimplemented by hand.
- **Alternatives considered**: Raw HTTP calls to the FCM v1 REST endpoint — full control, but reinvents auth/retry logic the SDK already provides; rejected as unnecessary complexity at this scope. A generic "PushSender" interface will still be defined at the Go package boundary so a future iOS (APNs) or Windows (WNS) sender can be added without touching round-triggering or reporting logic (FR-016).

## 4. Device identity (implements Clarification Q1 & Q2, spec.md)

- **Decision**: The Android client generates a UUID once on first launch and persists it in local app storage as the device's `install_id`; this is the primary key the coordinator uses for de-duplication. The FCM registration token is stored as a separate, mutable attribute on the same device record and is refreshed in place (via `FirebaseMessagingService.onNewToken`) without changing `install_id`.
- **Rationale**: Matches the clarified answer directly and cleanly solves the common real-world case this system must handle: FCM token rotation while the app stays installed (FR-019).
- **Known limitation (documented, not re-litigated)**: An `install_id` stored only in local app storage cannot survive a full uninstall + local-data wipe — a genuine reinstall will generate a new `install_id` and enroll as a new fleet member. Cross-reinstall continuity would require a hardware-level identifier, which was explicitly considered and rejected in clarification (Option C) as disproportionate for an internal, small (<50 device), anonymous test fleet. This is accepted as a reasonable tradeoff, not a defect.

## 5. Failure-outcome granularity (implements Clarification Q3, spec.md)

- **Decision**: The `Delivery` row has exactly one terminal-pending state (no receipt yet) regardless of whether the cause was an FCM send-level rejection or the device simply never confirming. The FCM send call's own error (if any) is logged for operator debugging but is **not** modeled as a distinct delivery status.
- **Rationale**: Directly implements the clarified answer; keeps the Delivery state machine to three observable states (`pending`, `received-good`, `received-anomalous`) instead of branching failure taxonomy the spec explicitly said not to track.

## 6. Latency computation — which clock

- **Decision**: The Android client captures its own arrival timestamp (`received_at`) at the moment the FCM message is handled in the background service, and sends that timestamp (not just "now" server-side) in the receipt report. The coordinator computes `latency = received_at - round.sent_at` using the client-supplied timestamp.
- **Rationale**: The spec's own edge case ("non-positive computed latency... a clock-skew or measurement artifact") only makes sense if the receipt timestamp originates from the device's own (independent, potentially skewed) clock — a server-stamped receipt time compared to a server-stamped send time could never go negative. Server-side receipt-processing time is deliberately excluded from the latency measurement so that reporting-pipeline delay doesn't pollute the push-delivery-latency number the tool exists to measure.
- **Alternatives considered**: Stamping receipt time on the server when the HTTP call arrives — simpler, avoids trusting client clocks, but conflates "time to receive the push" with "time to make an HTTP call back," and cannot produce the clock-skew anomaly the spec explicitly requires detecting.

## 7. Android background reception & retry

- **Decision**: A `FirebaseMessagingService` subclass handles `onMessageReceived` in the background; receipt reporting to the coordinator is performed through `WorkManager` with exponential backoff so a transient network failure is retried rather than dropped (FR-023). The on-screen message log is persisted locally (Room over SQLite) so it survives app restarts.
- **Rationale**: `WorkManager` is the standard Android-recommended mechanism for guaranteed, retryable background work, matching FR-023 exactly. Room gives a persisted, queryable local log with minimal boilerplate.
- **Alternatives considered**: Custom retry loop with `AlarmManager` — reinvents what `WorkManager` already does reliably; rejected.

## 8. Build & CI tooling

- **Decision**: A root `Makefile` exposes `build`, `test`, `lint` (each with `-server` / `-android` suffixed variants, and the bare target running both where applicable — `test` only runs the server since Android has no test coverage requirement per this feature's scope). Server lint via `golangci-lint`; Android lint via the Android Gradle Plugin's built-in `lint` task.
- **Rationale**: Directly matches the user's explicit request for unified build/test/lint entrypoints usable both locally and from CI.
- **GitHub Actions**: Two workflows — `server-release.yml` (build + push a Docker image to GHCR) and `android-release.yml` (build a release APK/AAB and attach it to a GitHub Release). Both gate on the corresponding `make lint`/`make test` targets passing first. Signing material (Android keystore) and push credentials (FCM service-account JSON) are supplied exclusively via repository secrets, never committed (FR-030).
