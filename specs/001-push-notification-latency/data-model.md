# Data Model: Push Notification Latency Measurement

Derived from spec.md Key Entities, refined by Clarifications session 2026-07-08.

## Device

Represents one enrolled fleet member. Identity is the app-generated `install_id`, not the platform push token (Clarification Q1).

| Field | Type | Notes |
|---|---|---|
| `id` | integer, PK | Internal surrogate key |
| `install_id` | text, unique, not null | App-generated UUID; stable across token rotation; primary de-dup key |
| `platform` | text, not null | `"android"` today; enum-like, extensible per FR-016 (`"ios"`, `"windows"`, ...) |
| `push_token` | text, not null | Current platform push token/registration ID; mutable, updated in place on rotation |
| `created_at` | timestamp, not null | First enrollment time |
| `updated_at` | timestamp, not null | Last time this record was touched (re-enrollment or token update) |

**Rules**:
- `install_id` is globally unique; enrollment with a known `install_id` updates `push_token`/`updated_at` on the existing row (FR-002) rather than inserting a new row.
- No delete/aging-out lifecycle in this iteration (Assumption, spec.md).

## Round

One "send now to everyone" instance.

| Field | Type | Notes |
|---|---|---|
| `id` | integer, PK | Internal surrogate key |
| `public_id` | text, unique, not null | Externally-exposed unique round identifier (e.g., ULID/UUID), assigned at trigger time (FR-006) |
| `sent_at` | timestamp, not null | Recorded before any device could plausibly have responded (FR-006) |
| `targeted_device_count` | integer, not null | Snapshot of fleet size at trigger time, for later success-rate denominators even if devices are later added/removed |

**Rules**:
- Rounds are independent and may overlap; there is no "one active round at a time" constraint (spec places no limit on this).
- Immutable once created except through Delivery updates.

## Delivery

One device's participation in one round. This is the append-only fact table latency/success reporting is built on.

| Field | Type | Notes |
|---|---|---|
| `round_id` | integer, FK → Round | |
| `device_id` | integer, FK → Device | |
| `sent_at` | timestamp, not null | Copied from Round.sent_at at creation time |
| `received_at` | timestamp, nullable | Set once, from the client-reported arrival timestamp (Research §6); null = still pending |
| `latency_ms` | integer, nullable | Computed once on receipt: `received_at - sent_at`, in milliseconds |
| `status` | text, not null | One of: `pending`, `received_good`, `received_anomalous` (see State Transitions) |

**Primary key**: (`round_id`, `device_id`) — enforces "exactly one delivery outcome per device per round" (spec.md Round definition) and makes duplicate receipt reports naturally idempotent (FR-012).

**State Transitions**:

```
[created at round-trigger time] → status = pending, received_at = NULL
        │
        │  device reports receipt (first time only; further reports for
        │  the same round_id+device_id are no-ops per FR-012)
        ▼
status = received_good        (latency_ms > slow_threshold_ms → still "received", but
   (0 < latency_ms <= threshold)   flagged slow in reporting — threshold is a reporting-time
        or                          concern per FR-026/FR-027, not a stored status value)
status = received_anomalous   (latency_ms <= 0 — clock-skew/measurement artifact, FR mandates
                                this is never shown as "good")
```

- A `pending` row that never receives a report simply stays `pending` forever — this is itself a meaningful, permanent result (spec.md §4.2), not an error state requiring cleanup.
- Unknown-device or unknown/nonexistent-round receipt reports never create or mutate a Delivery row (FR-011) — they are rejected at the API layer before reaching storage.
- No separate "send failure" status exists; an FCM-level send error and ordinary non-response are both simply `pending` forever (Clarification Q3, Research §5).

**Derived field (reporting-time only, not stored)**: `speed_category` = `slow` if `status = received_good` and `latency_ms > slow_threshold_ms`, else `good` if `status = received_good`, else `anomalous` if `status = received_anomalous`, else `pending`. The threshold is a configuration value (FR-027), not a database column, so changing it doesn't require rewriting historical rows.
