# Push Notification Latency Measurement

Measures how long push notifications actually take to reach a fleet of devices, and how often they never arrive at all. See [`specs/001-push-notification-latency/spec.md`](specs/001-push-notification-latency/spec.md) for the full behavioral specification.

- `server/` — the coordinator: a Go + SQLite service that enrolls devices, triggers rounds, records receipts, and serves latency/success-rate reports (see [`contracts/openapi.yaml`](specs/001-push-notification-latency/contracts/openapi.yaml)).
- `android/` — the Android client: enrolls automatically via FCM, receives rounds in the background, and shows a categorized on-screen log.

## Server: environment variables

| Variable | Required | Default | Purpose |
|---|---|---|---|
| `DB_PATH` | no | `./coordinator.db` | Path to the SQLite database file |
| `MIGRATIONS_DIR` | no | `migrations` | Directory of `*.sql` migrations applied on startup |
| `LISTEN_ADDR` | no | `:8080` | HTTP listen address |
| `OPERATOR_TOKEN` | **yes** | — | Bearer token required to call `POST /v1/rounds` (FR-029) |
| `FCM_SERVICE_ACCOUNT_JSON` | **yes** | — | Path to a Firebase service-account credentials file (FR-030 — never commit this file) |

None of these are ever hardcoded in source; supply them via your shell, a `.env` file consumed by your process manager, or your deployment platform's secret store.

## Android: per-build configuration

The coordinator's base URL is set per Gradle product flavor (`dev`, `staging`, `prod`) via `BuildConfig.COORDINATOR_BASE_URL` in `android/app/build.gradle.kts` (FR-028). The Firebase project is likewise selected per flavor by placing that environment's `google-services.json` at `android/app/google-services.json` before building (see `.github/workflows/android-release.yml` for how CI supplies this from a secret).

Release signing is supplied entirely through environment variables — `RELEASE_KEYSTORE_PATH`, `RELEASE_KEYSTORE_PASSWORD`, `RELEASE_KEY_ALIAS`, `RELEASE_KEY_PASSWORD` — so the keystore itself is never committed. A release build without those variables set simply produces an unsigned artifact.

## Scheduling repeated rounds (FR-003)

Triggering a round is a single authenticated `POST /v1/rounds` call, so any external scheduler works — a cron job, a scheduled GitHub Actions workflow, etc.:

```sh
watch -n 30 curl -sX POST "$COORDINATOR_URL/v1/rounds" -H "authorization: Bearer $OPERATOR_TOKEN"
```

## Building, testing, linting

```sh
make build   # build-server + build-android
make test    # server only — Android has no test coverage requirement in this iteration
make lint    # lint-server (golangci-lint) + lint-android (Android Gradle lint)
```

See [`quickstart.md`](specs/001-push-notification-latency/quickstart.md) for a full local walkthrough, including a curl-driven end-to-end example.

## CI/CD

- `.github/workflows/server-release.yml` — lints and tests the server on every push/PR touching `server/`, and publishes a Docker image to GHCR on pushes to `main` or `server-v*` tags.
- `.github/workflows/android-release.yml` — lints the Android app on every push/PR touching `android/`, and builds + attaches a signed release AAB/APK to a GitHub Release on `android-v*` tags.
