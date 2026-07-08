# Feature Specification: Push Notification Latency Measurement

**Feature Branch**: `001-push-notification-latency`
**Created**: 2026-07-08
**Status**: Draft
**Input**: User description: "measure the latency of push notification services, we will start with android, and probably extend to ios, windows, etc later. The server / service should be able to handle all of them in generic fashion."

## Clarifications

### Session 2026-07-08

- Q: How should the system define "the same device" for de-duplication purposes across app reinstalls and platform token rotation (FR-002, FR-019)? → A: A stable, app-generated installation ID (created once, persisted locally) is the device's primary identity; the platform push token is just an attribute that can change without altering identity.
- Q: What is the target fleet scale (concurrently enrolled devices) the system must handle? → A: Small — fewer than 50 devices, consistent with an internal QA/testing fleet rather than a production-scale population.
- Q: Should a push-provider-level send failure (e.g., provider rejects the send, service outage) be recorded differently from a device simply never confirming receipt? → A: No — both are recorded as the same "never received" outcome for that round; no separate failure reason is tracked.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Measure round-trip delivery latency across a device fleet (Priority: P1)

An operator wants to know, right now, how long it takes for a push notification to actually reach devices in a test fleet. They trigger a round, and shortly after, they can see for each device whether it arrived and how long it took.

**Why this priority**: This is the entire reason the system exists. Without the ability to trigger a round and observe per-device latency, there is no product.

**Independent Test**: Enroll at least one device, trigger a round, and confirm that a delivery record appears showing sent time, received time, and computed latency for that device.

**Acceptance Scenarios**:

1. **Given** one or more devices are enrolled in the fleet, **When** an operator triggers a round, **Then** the system records a unique round with a send time and a pending delivery outcome for every currently enrolled device.
2. **Given** a round has been triggered, **When** a reachable device receives the notification, **Then** the device reports receipt and the system records the receipt time and computes a positive latency for that device/round pairing.
3. **Given** a round has been triggered, **When** a device is offline or unreachable at send time, **Then** that device's delivery outcome remains "never received" for that round rather than being delivered late or silently dropped from the data.

---

### User Story 2 - Automatic, duplicate-free fleet enrollment (Priority: P2)

A device running the client app should become part of the measured fleet automatically, without any manual setup, and rejoining (e.g., after reinstall) should never create a second, duplicate entry for the same physical device.

**Why this priority**: Latency measurement is only meaningful if the fleet roster accurately reflects real, distinct devices. Duplicate or missing enrollment corrupts every downstream statistic (User Story 1 depends on this being correct).

**Independent Test**: Install and open the app on a device to enroll it, confirm one fleet entry exists, then simulate a reinstall/re-enrollment and confirm the fleet still contains exactly one entry for that device.

**Acceptance Scenarios**:

1. **Given** a device has never enrolled before, **When** the app is opened for the first time, **Then** the device is automatically added to the fleet roster without operator intervention.
2. **Given** a device is already enrolled, **When** the app is opened again with its existing membership record intact, **Then** the app recognizes it is already enrolled and does not re-register or duplicate the fleet entry.
3. **Given** a device's underlying push identity changes (e.g., token rotation) after enrollment, **When** the app detects the new identity, **Then** the system updates the existing fleet membership to the new identity rather than creating a second entry.

---

### User Story 3 - Review aggregate delivery health and identify slow or failed deliveries (Priority: P3)

An operator (or an automated report) wants to look back over many rounds and understand overall delivery success rate and typical latency, including which rounds or time periods perform worse, so they can spot systemic problems in the push pipeline.

**Why this priority**: Single-round measurement (User Story 1) is useful in the moment, but the long-term value of this tool is trend and reliability analysis across many rounds over time.

**Independent Test**: After several rounds have been triggered with a mix of successful and non-responding devices, request an aggregate report and confirm it shows success rate and latency statistics, sliceable by when rounds were sent.

**Acceptance Scenarios**:

1. **Given** multiple rounds have completed with a mix of received and never-received outcomes, **When** an operator requests a delivery success report, **Then** the system shows the fraction of targeted devices that received each round, and overall.
2. **Given** rounds have been sent at different times of day, **When** an operator requests a report sliced by send time, **Then** the system exposes success rate and latency broken out by that time dimension.
3. **Given** many delivered rounds exist, **When** an operator looks for problem deliveries, **Then** the system can surface the individual deliveries with the highest latency.

### Edge Cases

- A device reports receipt for a round it was never targeted by, or that doesn't exist (e.g., malformed or replayed report) — the system must not crash and must not corrupt other devices' or rounds' data.
- A device reports receipt for the same round more than once (e.g., retried network call) — the recorded receipt time and latency must not be corrupted or duplicated.
- A device's computed latency is zero or negative (receipt timestamp at or before send timestamp, indicating clock skew or a measurement artifact) — this must be visually and analytically distinguished from a genuine successful delivery, never counted as "good."
- A round is triggered when zero devices are currently enrolled — the round should still be recorded, simply with no targeted devices.
- A device that received a notification is offline when it tries to report receipt — the report must be retried rather than silently dropped, since a lost report would incorrectly count as non-delivery.
- An unauthenticated or unauthorized caller attempts to trigger a round — this administrative action must be rejected.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST allow any device running the client app to join the fleet automatically, with no manual approval step by the operator.
- **FR-002**: The system MUST treat re-enrollment of an already-known device (e.g., after reinstall, or after the device's underlying push identity changes) as an update to the existing fleet membership, never as a new, duplicate entry. Devices are identified for this purpose by a stable, app-generated installation ID persisted locally, not by the platform push token (which may change independently of device identity).
- **FR-003**: The system MUST provide a way to trigger a round on demand that is trivially automatable for repeated, unattended invocation (e.g., on a fixed schedule).
- **FR-004**: Triggering a round MUST notify every device currently in the fleet as close to simultaneously as the underlying push mechanism allows.
- **FR-005**: Round notifications MUST be delivered on an immediate, best-effort basis — a device unreachable at send time is treated as a non-delivery for that round and MUST NOT receive it later ("fresh or dropped," no queuing for later delivery).
- **FR-006**: Triggering a round MUST assign the round a unique identifier and record its send time before any device could plausibly have responded.
- **FR-007**: Triggering a round MUST create one pending delivery outcome per targeted device, so non-responding devices remain visible in the data as "still pending," not absent.
- **FR-008**: The round-trigger action MUST return an immediate confirmation that the round was dispatched, independent of any individual device's eventual outcome.
- **FR-009**: The system MUST provide a way for a device to report that it received a specific round, identifying both itself and the round.
- **FR-010**: Recording a receipt report MUST store the receipt time against that specific device/round pairing and allow the resulting latency to be computed.
- **FR-011**: The system MUST handle receipt reports for an unknown device, an unknown round, or a round that has already been settled for that device gracefully (e.g., ignored or reported as invalid) without crashing or corrupting other data.
- **FR-012**: The system MUST treat repeated receipt reports for the same device/round pairing as safe to retry — a duplicate report must not corrupt the originally recorded receipt time or latency.
- **FR-013**: The system MUST make individual delivery latency (per round/device pairing) queryable, including the ability to identify the slowest deliveries.
- **FR-014**: The system MUST make delivery success rate queryable, both overall and sliceable by when the round was sent (e.g., time of day).
- **FR-015**: The system's reporting MUST be extensible to richer statistical views (e.g., percentile latency, trends over time) beyond simple min/max, without requiring ad-hoc raw data queries as the primary interface.
- **FR-016**: The system MUST target devices generically across push platforms (starting with Android, extensible to iOS, Windows, and others) without requiring platform-specific handling in the round-triggering or reporting logic.
- **FR-017**: The client app MUST enroll itself with the coordinator automatically on first launch, or whenever it has no valid local membership record, with no user action beyond installing/opening the app.
- **FR-018**: The client app MUST recognize when it is already enrolled and skip re-enrollment, informing the user that enrollment is already active.
- **FR-019**: The client app MUST detect when the platform issues a new device identity/token after enrollment and re-enroll using the new identity so the fleet roster stays accurate.
- **FR-020**: The client app MUST surface enrollment failures to the user in a lightweight, non-blocking way without crashing or blocking use of the app.
- **FR-021**: The client app MUST detect a round notification arriving in the background, without requiring the app to be in the foreground or the user to tap the system notification.
- **FR-022**: Upon receiving a round notification, the client app MUST capture the arrival time as precisely as possible and report receipt back to the coordinator.
- **FR-023**: The client app MUST retry receipt reporting on failure (e.g., transient network issues) rather than dropping it silently, since a lost report is indistinguishable from non-delivery in the fleet-wide data.
- **FR-024**: The client app MUST display a running, most-recent-first list of every round the device has received, showing at minimum: the round identifier, the time sent, the time received, and the computed latency.
- **FR-025**: The client app MUST show a clear empty state when no rounds have been received yet, rather than a blank screen.
- **FR-026**: The client app MUST visually categorize each entry's latency as slow/concerning (above a configurable threshold), good/acceptable (at or below the threshold), or anomalous (non-positive latency) — anomalous results must never be shown as "good."
- **FR-027**: The slow-latency threshold MUST be configurable without requiring a rebuild of the app.
- **FR-028**: The identity of the push project/provider the app registers against, and the address of the coordinator it reports to, MUST be configurable per build/environment.
- **FR-029**: Triggering a round MUST be restricted to authorized operators/callers; it MUST NOT be an operation any arbitrary caller can invoke.
- **FR-030**: Any credentials the coordinator uses to talk to underlying push platforms MUST be stored as secrets, not embedded in source code.

### Key Entities

- **Device**: A uniquely and durably identifiable member of the test fleet, capable of receiving push notifications and reporting receipt back. Identified primarily by a stable, app-generated installation ID persisted locally (re-identifiable across app restarts/reinstalls to prevent duplicate fleet entries); separately holds a platform-specific push token/delivery identity that may be rotated by the platform without affecting the device's identity.
- **Round**: One instance of "send a test notification to every currently known device now." Has a unique identifier, exactly one send time, and produces exactly one delivery outcome per targeted device.
- **Delivery**: The record of one device's participation in one round — links a Device and a Round, and holds the send time, the (optional, if/when it happens) receipt time, and the derived latency and its categorization (good/slow/anomalous/pending). A delivery that never receives a receipt report is simply "pending/never received," regardless of whether the underlying cause was device unreachability or a push-provider-level send failure — no separate failure reason is tracked.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: An operator can trigger a round and see confirmation that it was dispatched in under 2 seconds.
- **SC-002**: For a fleet of reachable devices, at least 95% of individual delivery latencies become visible in reporting within 30 seconds of actual receipt on the device.
- **SC-003**: Re-enrolling the same physical device (e.g., simulated reinstall) never results in more than one active fleet entry for that device, verified across repeated trials.
- **SC-004**: Zero data-corruption incidents (crashes, corrupted records, or lost unrelated data) occur when the system is sent malformed, duplicate, or out-of-order receipt reports during testing.
- **SC-005**: An operator can determine, for any given round, the fraction of targeted devices that received it, and can identify the slowest deliveries across an arbitrary set of rounds, without needing to write custom data queries.
- **SC-006**: An operator can compare delivery success rate across two different time-of-day windows using only the system's built-in reporting.
- **SC-007**: The system supports adding a new device platform (beyond the initially supported one) to the fleet without changes to how rounds are triggered or how delivery/latency data is reported.

## Assumptions

- "Android first, extensible to iOS/Windows/etc." means the coordinator's data model and round-triggering/reporting behavior must be platform-agnostic from the start, even though only one platform's client app is built initially.
- The fleet is a controlled test population (internal devices/emulators), not the general public; there is no end-user-facing privacy/consent flow beyond what's already implied by installing an internal test app.
- The fleet is small — fewer than 50 concurrently enrolled devices — so the coordinator's fan-out and reporting do not need to be designed for large-scale, high-throughput device populations.
- "Trivially automatable" round triggering means an authenticated HTTP-style call or equivalent that a scheduler can invoke on a timer; no specific scheduling mechanism is mandated by this spec.
- A single global slow-latency threshold (default derived from the original system: 2 seconds) is sufficient; per-round or per-device thresholds are out of scope unless later requested.
- Long-term staleness/aging-out of devices that never respond over an extended period is out of scope for this iteration, consistent with the original system's behavior, and can be revisited later.
- Authorization for triggering rounds and enrolling devices can use any reasonable access-control mechanism (e.g., API key, token); the specific mechanism is a planning/implementation decision, not a specification requirement.
