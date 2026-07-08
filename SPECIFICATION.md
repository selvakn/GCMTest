# Push Notification Latency Tester — Behavioral Specification

> This document describes the **intent and behavior** of the system, independent of any specific technology, API, or platform SDK. It exists so the system can be rebuilt on a current stack without inheriting assumptions from the original implementation. Nothing here should be read as prescribing a particular language, framework, database, or push provider.

## 1. Purpose

The system exists to answer one question, repeatedly, over time: **"When we send a push notification to a fleet of devices, how long does it actually take to arrive, and how often does it not arrive at all?"**

It is a measurement tool, not a messaging product. The notifications it sends carry no meaningful content — they exist purely to be timed. The system has two responsibilities:

1. Trigger notification "rounds" to every known device and record when each round was sent.
2. Detect, on each device, the moment a notification actually arrives, and report that moment back — so the gap between "sent" and "received" (the latency), and the fact of non-arrival, can be measured and analyzed.

## 2. Actors

- **Operator** — a person or scheduled process who decides when to trigger a new round of test notifications.
- **Device** — any phone/tablet/emulator running the client app, participating in the test fleet.
- **Coordinator** (the backend) — the always-on component that knows about all devices, triggers rounds, and records outcomes.

## 3. Core Concepts

- **Device**: something capable of receiving push notifications and reporting back. Each device must be uniquely identifiable and re-identifiable across app restarts/reinstalls, so that repeated participation doesn't create phantom duplicate devices in the fleet.
- **Round**: one instance of "send a test notification to every currently known device right now." A round has exactly one send time, and produces exactly one outcome per device (received-at-some-time, or never-received).
- **Delivery**: the record of one device's participation in one round — when it was sent to that device, and (if/when it happens) when that device confirmed it arrived.
- **Latency**: the time elapsed between a round's send moment and a given device's confirmed receipt moment, for devices that did receive it.

## 4. Required Behavior — Coordinator

### 4.1 Fleet membership
- Any device running the client app must be able to join the fleet automatically, without manual setup by the operator.
- Joining (or rejoining, e.g. after reinstall or losing local state) must not create duplicate entries for what is effectively the same device — rejoining should update the existing membership, not multiply it.
- The fleet roster must always reflect currently-known devices; there is no manual approval step.

### 4.2 Triggering a round
- The coordinator must expose a way to trigger a round on demand (invoked by the operator or a scheduler — the mechanism is not prescribed, but it must be trivially automatable for repeated, unattended use, e.g. "run this every N seconds/minutes indefinitely").
- Triggering a round must:
  - Notify every device currently in the fleet, as close to simultaneously as the underlying push mechanism allows.
  - Be delivered as an immediate, "now or never" notification — a device that is offline or unreachable at send time should be treated as a non-delivery for this round, not queued for later delivery. Late delivery would corrupt the latency measurement, so "fresh or dropped" is a deliberate requirement, not an incidental limitation.
  - Assign the round a unique identifier and record its send time, before any device could plausibly have already responded.
  - Record one pending delivery outcome per device targeted, so that devices which never respond are visibly "still pending" / "never arrived," not silently absent from the data.
- The trigger action's immediate response only needs to confirm the round was dispatched — actual per-device success is discovered later, asynchronously, as devices report in.

### 4.3 Recording receipt
- The coordinator must expose a way for a device to report "I received round X," identified by the device and the round.
- Reporting receipt must record the receipt time against that specific device+round pairing.
- Reporting receipt for an unknown device or an unknown/already-settled round must not crash or corrupt other data — it should be handled gracefully (e.g., ignored or reported as invalid), since network conditions can plausibly cause duplicate, delayed, or malformed receipt reports.
- Receipt reporting should be safe to retry — a device that reports receipt more than once for the same round (e.g., due to a retried network call) must not corrupt the recorded data.

### 4.4 Analysis / reporting
The coordinator must make the following information obtainable (via whatever reporting mechanism — API, dashboard, export — fits the modern stack):
- **Latency of individual deliveries** — for any round/device pairing that was received, how long it took, and the ability to find the slowest deliveries.
- **Delivery success rate over time** — what fraction of targeted devices ever received a given round, or rounds overall, sliceable by when the round was sent (e.g., "rounds sent in the evening succeed less often than rounds sent midday") so time-of-day / system-state effects on deliverability are visible.
- Ideally, richer statistical views than min/max (e.g., typical/percentile latency, trends over time) should be easy to add, since raw ad-hoc querying is not an acceptable long-term interface for this data.

## 5. Required Behavior — Client App

### 5.1 Fleet enrollment
- On first launch (or whenever it has no valid local membership record), the app must enroll itself with the coordinator automatically — no user action required beyond installing/opening the app.
- If already enrolled, the app should recognize this and skip re-enrollment, informing the user enrollment is already active.
- If the underlying platform issues a new device identity/token at any point after enrollment (identity rotation is a normal occurrence on real push platforms), the app must detect this and re-enroll with the new identity, so the coordinator's fleet roster stays accurate. (The original implementation did not handle this case — it is called out here as required behavior, not carried forward as a gap.)
- Enrollment failures should be surfaced to the user in some lightweight, non-blocking way (the original used transient toasts; a modern equivalent — snackbar, status indicator, etc. — is fine) but should not crash the app or block its use.

### 5.2 Receiving a round
- The app must detect a round's notification arriving in the background, without requiring the app to be in the foreground, and without requiring the user to tap/open a system notification first.
- Upon arrival, the app must:
  - Capture the arrival time as precisely as possible.
  - Report receipt back to the coordinator, identifying itself and the round.
  - Reflect the new message in the on-screen list immediately if the app is open, or make it visible immediately upon next open otherwise.
- Reporting receipt to the coordinator is important data, not a nice-to-have — a failure to report (e.g., transient network issue) should be retried rather than silently dropped, since a dropped report permanently and incorrectly counts as "never delivered" in the fleet-wide statistics.

### 5.3 On-screen message log
- The app must show a running, most-recent-first list of every round this device has received during the current app session (and, ideally, persisted across restarts).
- When no rounds have been received yet, the app must show a clear empty state rather than a blank screen.
- Each entry in the list must show, at minimum:
  - An identifier for the round.
  - The time the round was sent.
  - The time this device received it.
  - The computed latency between those two times.
- **Latency must be visually categorized**, not just shown as a raw number, so a glance at the list reveals problem deliveries:
  - Latency above some threshold (originally 2 seconds) should be visually flagged as slow/concerning.
  - Latency at or below that threshold should be visually flagged as good/acceptable.
  - A non-positive computed latency (received-time appears at or before sent-time — a clock-skew or measurement artifact rather than a real result) must be visually distinguished as an anomaly, and must **not** be presented as a "good" outcome. This is a correction relative to the original app, which miscategorized this case as success.
  - The threshold for "slow" should be easy to change (configuration, not a recompiled constant).
- The list is the app's primary purpose — there is no other significant screen or navigation required.

### 5.4 Configuration
- The identity of the push project/provider the app registers against, and the address of the coordinator it reports to, must be configurable per build/environment rather than fixed permanently in code, so the same app can be pointed at different test environments.

## 6. End-to-End Behavior (intended flow)

1. A device installs and opens the app → it enrolls itself with the coordinator → the coordinator's fleet roster now includes it.
2. An operator (or a schedule) triggers a round → the coordinator records the round's send time and a pending outcome for every currently-enrolled device → a notification is dispatched to all of them at once, on a best-effort, immediate/no-queueing basis.
3. Each device that is reachable receives the notification → records its arrival time → updates its own on-screen log with a correctly categorized latency → reports receipt to the coordinator.
4. Devices that were unreachable at send time never receive that round; their outcome for that round remains "pending/never," which is itself a meaningful result, not an error.
5. Over many rounds, the operator (or an automated report) reviews aggregate delivery success and latency to understand the health and timeliness of the push pipeline.

## 7. Explicit Non-Goals

- No user accounts, authentication of end users, or personalization — every enrolled device is an anonymous, interchangeable fleet member for the purpose of this tool.
- No message content, rich payloads, or user-facing notification text — the payload is purely an identifier and a timestamp for measurement purposes.
- No manual device management UI — enrollment is fully automatic in both directions (join and, implicitly, staleness of devices that never respond over a long period may need aging-out, though the original system did not do this either — a rewrite should consider whether it's in scope).

## 8. Requirements Carried Forward as Explicit Corrections

These are behaviors the original implementation got wrong or omitted; they are stated here as requirements for the rewrite, not as historical bugs to research further:

- Duplicate fleet entries from re-enrollment must not occur.
- Unknown/invalid receipt reports must not crash the coordinator.
- Receipt reporting from the client must be retried on failure, not dropped silently.
- Non-positive latency must be flagged as anomalous, not shown as a success.
- Triggering a round and enrolling a device are privileged/administrative and fleet-membership operations respectively — access to trigger rounds in particular should not be open to arbitrary callers.
- Any credentials the coordinator needs to talk to the underlying push platform must be stored as secrets, not embedded in source.
