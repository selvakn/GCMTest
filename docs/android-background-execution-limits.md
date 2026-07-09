# Android background execution limits, triggered by push notifications

A living research log. Question being answered: **when a push notification triggers work
on an Android device, what execution constraints does the OS actually enforce** — not
per documentation, but empirically, on real hardware, using this repo's coordinator +
FCM pipeline as the trigger mechanism? New findings should be appended as new dated
sections rather than overwriting old ones, since behavior can differ across Android
versions/OEMs and it's useful to keep the full history.

## Methodology

A diagnostic-only (not part of the product) set of services and an FCM hook were added
temporarily to the Android app (`android/app/src/main/java/com/tw/pushlatency/experiment/`)
to run this experiment:

- `BaseExperimentService` — shared logic for two concrete services:
  - `LongRunningTestService`, registered with `android:foregroundServiceType="specialUse"`.
  - `DataSyncTestService`, registered with `android:foregroundServiceType="dataSync"` —
    Android 15 gives this type a multi-hour cap before the system stops it, unlike
    `specialUse`; included to check for any *different* treatment within our 5-minute
    window (there wasn't).
  - Either can be started plain or promoted to a foreground service via an intent extra,
    and either can run a plain timer loop or hold an open TCP socket (see below).
- `ExperimentRunner` — the actual workload, one of:
  - **Timer loop**: logs a heartbeat every 10 seconds, up to a 300-second (5 minute) cap,
    then stops — proves the process/thread is alive, no I/O.
  - **Socket loop**: opens one long-lived `java.net.Socket` to a mock TCP echo server for
    the whole run, sending a ping line every 10 seconds and reading the reply — standing
    in for a websocket-style persistent connection (e.g. a chat app's realtime channel),
    to see whether active network I/O changes execution limits versus an idle timer.
  - The mock echo server is a small standalone Python script
    (`.gocache/test-results/mock_ws_echo_server.py`, not committed — regenerate from this
    doc if needed), run on the host and reached via `adb reverse tcp:9090 tcp:9090`.
- `PushMessagingService.onMessageReceived` — an FCM data message carrying an `experiment`
  field short-circuits normal round handling and starts the corresponding service/mode
  instead. Values used: `bg_service`, `fg_service`, `bg_service_socket`,
  `fg_service_socket`, `datasync_service`.
- Test messages were sent directly via the FCM Admin API (bypassing the coordinator's
  `/v1/rounds` — this is ad hoc infra testing, not part of the product's round data
  model), targeting one real device's push token directly.
- Observed via `adb logcat`, filtered to the `SvcExperiment`, `PushMessagingService`, and
  `ActivityManager` tags, watched live until either 300s completion or a kill/error.
- Doze mode was forced via `adb shell dumpsys deviceidle force-idle` (with
  `dumpsys battery unplug` first, since Doze requires unplugged state), and reverted with
  `dumpsys deviceidle unforce` + `dumpsys battery reset` afterward.

This harness is not committed to the app by default — it's reconstructable from this
document plus the git history of the commit that added it, if the experiment needs to be
repeated on a different device/Android version.

## 2026-07-08 — Motorola Edge 30 Ultra, Android 15 (API 35)

Real device connected over USB, coordinator reached via `adb reverse tcp:8080 tcp:18080`
(see main [README](../README.md#running-the-whole-stack-with-docker-compose)). Screen
kept awake and screen-off timeout extended during foreground/background tests; Doze
forced for the Doze tests. Each scenario capped at 300s (5 minutes) as requested — these
results say "no problem within 5 minutes," not "no problem ever."

| # | App state | Service type | Workload | Started OK? | Result | Killed by |
|---|---|---|---|---|---|---|
| 1 | Foreground | Plain background `Service` | timer | Yes | Survived full 300s | — (voluntary stop) |
| 2 | Backgrounded | Plain background `Service` | timer | Yes (temporary grace window) | **Killed at ~80s** (79.95s) | `ActivityManager`: `"Stopping service due to app idle"` |
| 3 | Backgrounded | Foreground (`specialUse`) | timer | Yes | Survived full 300s | — (voluntary stop) |
| 4 | Doze (forced deep idle) | Foreground (`specialUse`) | timer | Yes | Survived full 300s | — (voluntary stop) |
| 5 | Doze (forced deep idle) | Plain background `Service` | timer | Yes | **Killed at ~80s** (79.95s) | `ActivityManager`: `"Stopping service due to app idle"` |
| 6 | Backgrounded | Foreground (`dataSync`) | timer | Yes | Survived full 300s | — (voluntary stop) |
| 7 | Backgrounded | Plain background `Service` | **live socket** | Yes | **Killed at ~80s** (79.99s) | `ActivityManager`: `"Stopping service due to app idle"` (socket torn down as a side effect) |
| — | Doze | *(push delivery itself, not a service)* | — | — | Delivered in **<20ms** | n/a |

### Findings

1. **A plain background service started while the app is foregrounded has no execution
   limit** within the 5-minute window tested (#1) — expected, since the app process is
   already in a high-importance state.
2. **The real constraint appears once the app is backgrounded.** FCM's high-priority
   delivery does grant a temporary background-execution allowance — `startService()`
   (a plain, non-foreground service) succeeds even though the app has no foreground
   activity — but the OS kills that service roughly **80 seconds** later via its
   "app idle" background-execution-limit enforcement (#2). This was an explicit, named
   `ActivityManager` log line, not a silent process kill, and `Service.onDestroy()` was
   called (a graceful stop, not a hard kill) — the framework asked the service to stop
   cleanly rather than killing the process outright.
3. **Promoting to a foreground service (`startForegroundService` + `startForeground`)
   fully avoids the ~80s limit**, for both the `specialUse` (#3) and `dataSync` (#6)
   foreground service types — both survived the complete 300-second window with the app
   backgrounded, with no throttling observed. This is the standard, documented
   mitigation, confirmed here empirically rather than assumed from docs.
4. **Doze mode (forced deep idle) does not change either outcome.** A foreground service
   ran the full 300 seconds in Doze exactly as it did merely backgrounded (#4). More
   tellingly, a *plain* background service was killed at **the same ~80-second mark**
   whether or not the device was in Doze (#5 vs #2 — 79.95s both times) — the "app idle"
   background-execution limit is driven by the *app's* own background/idle state, not by
   *device-wide* Doze state. These are related but distinct mechanisms, and only the
   app-level one was observed to matter for service survival in this window.
5. **Active network I/O does not change the ~80-second limit.** A plain background
   service holding an open, actively-used TCP socket (ping/pong every 10s, simulating a
   websocket connection) was killed at **the same ~80-second mark** (#7, 79.99s) as the
   idle-timer version (#2, 79.95s) — within measurement noise of each other. The kill is
   about the service's execution-time budget, not specifically about revoking network
   access first. The socket itself closed as a direct consequence of the coroutine being
   cancelled when the service was torn down (`JobCancellationException`), not from a
   separate, earlier network-level restriction.
6. **Doze does not delay FCM message delivery.** A high-priority data message reached
   `onMessageReceived` in under 20ms even while the device was in forced deep idle —
   consistent with Google's documented Doze exemption for high-priority FCM messages
   (the device is briefly woken specifically to deliver them).

### Caveats / what this does *not* answer

- Only tested up to 5 minutes, as scoped. `dataSync`'s real advantage (a multi-hour cap
  before Android's newer FGS-timeout mechanism intervenes) wasn't approached — this only
  confirms it behaves like `specialUse` for the first 5 minutes, not that it's identical
  at the multi-hour boundary.
- Single device, single Android version (15 / API 35), single OEM (Motorola). Background
  execution behavior is known to vary by OEM battery-management customizations (Samsung,
  Xiaomi, and others are commonly reported as more aggressive than stock/near-stock
  Android). This result should not be assumed to generalize without testing on other
  hardware.
- The socket experiment (#7) used a plain TCP socket with request/response text lines,
  not a real WebSocket (HTTP Upgrade handshake, framing, ping/pong control frames) — close
  enough to test "does holding an active connection change service survival," but a real
  WebSocket library's own keepalive/reconnect behavior wasn't exercised.
- Foreground service + live socket, and Doze + live socket, were *not* tested — only the
  plain-background-service + socket combination (#7) was run, since that was the
  literal scenario asked about. It's plausible (based on #3/#4/#6 all showing foreground
  services are Doze-immune) that a foreground socket-holding service would also survive
  Doze, but that specific combination hasn't been empirically confirmed here.
- Doze's effects on things *other* than service/process survival (network access
  batching, wakelock deferral for cached/background processes, job/alarm batching) were
  not directly measured — only "does the service/socket keep running" was tested.
- Each figure is a single measurement per scenario, not an average across repeated
  trials — treat "~80s" as "somewhere around a minute and a half," not an exact constant
  Android guarantees. The two independent ~80s measurements (#2 at 79.95s, #5 at 79.95s,
  #7 at 79.99s) landing within 50ms of each other is a good sign it's a fairly stable
  timer-based threshold on this device/version rather than a noisy heuristic, but it
  should still not be hardcoded into product logic as a guarantee.

## Template for future entries

```markdown
## YYYY-MM-DD — <device>, Android <version> (API <level>)

<setup notes: real device vs emulator, OEM, anything different from the methodology above>

| # | App state | Service type | Workload | Started OK? | Result | Killed by |
|---|---|---|---|---|---|---|
| ... |

### Findings
...

### Caveats
...
```
