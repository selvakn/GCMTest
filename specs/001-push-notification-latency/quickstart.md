# Quickstart: Push Notification Latency Measurement

## Prerequisites

- Go 1.22+
- Android Studio / Android SDK (min SDK 26) with a Firebase project configured for FCM
- A Firebase service-account JSON (for the server to send via FCM Admin SDK)
- `sqlite3` CLI (optional, for inspecting the local DB file)

## Server

```sh
export FCM_SERVICE_ACCOUNT_JSON=/path/to/service-account.json   # never commit this file
export OPERATOR_TOKEN=some-local-dev-secret                     # bearer token for POST /v1/rounds
export DB_PATH=./coordinator.db

make build-server
make test-server
make lint-server

./server/bin/coordinator   # starts HTTP server, applies sqlite migrations on boot
```

## Android app

```sh
# Place google-services.json (from the same Firebase project) under android/app/
make build-android
make lint-android
```

Point the app at the local server and Firebase project via per-build configuration (FR-028):

```
# android/local.properties or gradle -P flags, per environment
coordinator.baseUrl=http://10.0.2.2:8080/v1   # emulator → host loopback
```

## Try it end-to-end

```sh
# 1. Enroll a device (the app does this automatically on first launch; shown here for manual testing)
curl -X POST localhost:8080/v1/devices \
  -H 'content-type: application/json' \
  -d '{"install_id":"test-device-1","platform":"android","push_token":"<fcm-token-from-logcat>"}'

# 2. Trigger a round
curl -X POST localhost:8080/v1/rounds -H "authorization: Bearer $OPERATOR_TOKEN"

# 3. Watch the device's on-screen log update, and/or poll the reports
curl localhost:8080/v1/reports/deliveries?order=latency_desc
curl localhost:8080/v1/reports/success-rate?bucket=hour
```

## Repeated/unattended rounds (FR-003)

```sh
watch -n 30 curl -sX POST localhost:8080/v1/rounds -H "authorization: Bearer $OPERATOR_TOKEN"
```

Any external scheduler (cron, GitHub Actions on a schedule, etc.) works equally well — the trigger endpoint is the only integration point required.

## Whole-repo tasks

```sh
make build   # build-server + build-android
make test    # server only — no Android test coverage in this iteration
make lint    # lint-server + lint-android
```
