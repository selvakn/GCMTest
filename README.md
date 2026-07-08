# Push Notification Latency Measurement

Measures how long push notifications actually take to reach a fleet of devices, and how
often they never arrive at all. It is a measurement tool, not a messaging product: the
notifications it sends carry no user-facing content, only an identifier and a timestamp.

See [`specs/001-push-notification-latency/spec.md`](specs/001-push-notification-latency/spec.md)
for the full behavioral specification, and
[`specs/001-push-notification-latency/plan.md`](specs/001-push-notification-latency/plan.md)
for the technical design.

## Sample results

From a real 100-round test (5-second intervals, ~8.5 minutes) run against the
`docker-compose` setup below — one coordinator container, one real device (Motorola Edge
30 Ultra, Android 15) connected over USB via `adb reverse`, and one `docker-android`
emulator (Android 14) on the compose network:

**Delivery success**

| Device | Platform | Rounds targeted | Received | Success rate |
|---|---|---|---|---|
| Real device (Motorola Edge 30 Ultra) | Android 15 | 100 | 100 | 100% |
| Emulator (docker-android) | Android 14 | 100 | 100 | 100% |

Both devices received every round. 19 of the real device's 100 receipts, and all 100 of
the emulator's, were flagged **anomalous** (non-positive computed latency) rather than
counted as a plain success — not because delivery failed, but because the device's clock
was measurably behind the coordinator's at that instant (the emulator's virtualized clock
ran a consistent ~145ms behind; the real device drifted more randomly, by a few tens of
milliseconds). This is the anomaly detection (FR-026) working as intended: the original
app being replaced miscategorized this exact case as a successful "good" result, which is
the bug this behavior explicitly corrects.

**Latency, real device, excluding clock-skew anomalies (n=81)**

| Min | Median | Average | p95 | Max |
|---|---|---|---|---|
| 1 ms | 57 ms | 63.2 ms | 131 ms | 296 ms |

**Slowest individual deliveries**

| Round | Sent | Received | Latency |
|---|---|---|---|
| `69578cc2` | 16:55:01.553 | 16:55:01.850 | 296 ms |
| `882e29ee` | 16:49:26.317 | 16:49:26.520 | 202 ms |
| `3ad0a4bb` | 16:52:21.814 | 16:52:22.005 | 190 ms |
| `22d84a16` | 16:49:10.759 | 16:49:10.922 | 163 ms |
| `a62f5f17` | 16:50:28.125 | 16:50:28.257 | 131 ms |

Reproduce this yourself with `docker compose up`, the `/v1/reports/deliveries` and
`/v1/reports/success-rate` endpoints, and the same `curl` loop described under
[Scheduling repeated rounds](#scheduling-repeated-rounds-fr-003).

## Architecture

```
server/     Go + SQLite coordinator: enrolls devices, triggers rounds, records
            receipts, serves latency/success-rate reports.
android/    Android client: enrolls automatically via FCM, receives rounds in
            the background, shows a categorized on-screen log.
```

The coordinator targets devices generically across push platforms (a `Platform` field
and a `PushSender` interface); only Android/FCM is implemented today, but iOS/Windows
clients can be added without changing round-triggering or reporting logic. See
[`specs/001-push-notification-latency/contracts/openapi.yaml`](specs/001-push-notification-latency/contracts/openapi.yaml)
for the full API contract.

## Prerequisites

- Go 1.24+
- JDK 17, Android SDK (platform 34, build-tools 34.0.0) for the Android client
- Docker, only if using `.devcontainer/` for one-off infra tasks (see below)

## Getting started

```sh
export FCM_SERVICE_ACCOUNT_JSON=/path/to/service-account.json
export OPERATOR_TOKEN=some-local-dev-secret

make build-server
make test-server
./server/coordinator
```

Place `google-services.json` under `android/app/` (matching the Firebase project
referenced above), then:

```sh
make build-android
```

See [`quickstart.md`](specs/001-push-notification-latency/quickstart.md) for a full
walkthrough, including a curl-driven end-to-end example (enroll a device, trigger a
round, inspect the reports).

## Running the whole stack with Docker Compose

`docker-compose.yml` brings up the coordinator alongside an Android emulator:

```sh
echo "OPERATOR_TOKEN=$(openssl rand -hex 24)" > .env
docker compose up -d
```

- `coordinator` — built from `server/Dockerfile`, published on `localhost:18080`, backed
  by the `fcm_service_account` secret (`secrets/fcm-service-account.json`, see
  `secrets/README.md`) and a named volume for the SQLite file.
- `android` — a `budtmo/docker-android` emulator (Android 14, `google_apis_playstore`).
  The Play-Store-certified system image is required for real FCM delivery — the generic
  `google_apis` image gets `AUTHENTICATION_FAILED` from Google's servers when requesting
  an FCM token. Once it has booted, install the `docker` Gradle flavor's APK
  (`COORDINATOR_BASE_URL=http://coordinator:8080/v1`, resolved via the compose network's
  DNS).

To use a **real device** instead of/alongside the emulator: connect it over USB with
debugging enabled, then forward its loopback back to the coordinator regardless of the
device's own network:

```sh
adb reverse tcp:8080 tcp:18080
```

and build/install the `device` Gradle flavor (`COORDINATOR_BASE_URL=http://localhost:8080/v1`,
which resolves through that reverse tunnel).

## Server: environment variables

| Variable | Required | Default | Purpose |
|---|---|---|---|
| `DB_PATH` | no | `./coordinator.db` | Path to the SQLite database file |
| `MIGRATIONS_DIR` | no | `migrations` | Directory of `*.sql` migrations applied on startup |
| `LISTEN_ADDR` | no | `:8080` | HTTP listen address |
| `OPERATOR_TOKEN` | yes | none | Bearer token required to call `POST /v1/rounds` (FR-029) |
| `FCM_SERVICE_ACCOUNT_JSON` | yes | none | Path to a Firebase service-account credentials file (FR-030 — never commit this file) |

None of these are ever hardcoded in source; supply them via your shell, a `.env` file
consumed by your process manager, or your deployment platform's secret store.

## Android: per-build configuration

The coordinator's base URL is set per Gradle product flavor (`dev`, `staging`, `prod`)
via `BuildConfig.COORDINATOR_BASE_URL` in `android/app/build.gradle.kts` (FR-028). The
Firebase project is selected by placing `google-services.json` at
`android/app/google-services.json` before building — a single Firebase project can
register one Android app per flavor's package name and still produce one config file
covering all of them.

Release signing is supplied entirely through environment variables —
`RELEASE_KEYSTORE_PATH`, `RELEASE_KEYSTORE_PASSWORD`, `RELEASE_KEY_ALIAS`,
`RELEASE_KEY_PASSWORD` — so the keystore itself is never committed. A release build
without those variables set simply produces an unsigned artifact.

## Scheduling repeated rounds (FR-003)

Triggering a round is a single authenticated `POST /v1/rounds` call, so any external
scheduler works — a cron job, a scheduled GitHub Actions workflow, etc.:

```sh
watch -n 30 curl -sX POST "$COORDINATOR_URL/v1/rounds" -H "authorization: Bearer $OPERATOR_TOKEN"
```

## Research notes

- [`docs/android-background-execution-limits.md`](docs/android-background-execution-limits.md) —
  empirical findings on how long Android lets FCM-triggered background work run before
  the OS intervenes, across foreground/background/Doze states.

## Building, testing, linting

```sh
make build   # build-server + build-android
make test    # server only — Android has no test coverage requirement in this iteration
make lint    # lint-server (golangci-lint) + lint-android (Android Gradle lint)
```

## CI/CD

- `.github/workflows/server-release.yml` — lints and tests the server on every push/PR
  touching `server/`; on pushes to `main` or `server-v*` tags, also builds and publishes
  a Docker image to GHCR.
- `.github/workflows/android-release.yml` — lints the Android app on every push/PR
  touching `android/`; on `android-v*` tags, also builds and attaches a signed release
  AAB/APK to a GitHub Release.

Both workflows read Firebase/signing credentials from repository secrets
(`GOOGLE_SERVICES_JSON_BASE64`, `FCM_SERVICE_ACCOUNT_JSON_BASE64`,
`ANDROID_RELEASE_KEYSTORE_BASE64`, `ANDROID_RELEASE_KEYSTORE_PASSWORD`,
`ANDROID_RELEASE_KEY_ALIAS`, `ANDROID_RELEASE_KEY_PASSWORD`) — none of this material is
ever committed to the repository.

## Provisioning credentials (`.devcontainer/`)

`.devcontainer/` builds a throwaway tooling image (`gcloud`, `gh`, `keytool`) used for
one-off infrastructure tasks — registering the Firebase project, rotating credentials,
pushing the GitHub secrets above — without installing any of that tooling on a
developer's host machine. It is not needed for day-to-day development.

Locally fetched credentials (for building against the real Firebase project without
touching CI) live in a gitignored `secrets/` directory; see `secrets/README.md` (created
alongside the credentials, not checked in) for exact env vars to export.

## Repository layout

```
server/                 Go coordinator (cmd/, internal/{api,store,push,model,config}/)
android/                Android client (Kotlin, Gradle)
specs/                  Spec-kit artifacts: spec, plan, research, data model, tasks
.github/workflows/      CI: build/test/lint, Docker + Android release publishing
.devcontainer/          One-off infra tooling image (see above)
```

This project follows a spec-driven workflow: behavioral requirements live in
`specs/001-push-notification-latency/spec.md`, the technical plan in `plan.md`, and the
task breakdown in `tasks.md`. Read those before making non-trivial changes.
