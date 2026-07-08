CREATE TABLE IF NOT EXISTS devices (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    install_id  TEXT NOT NULL UNIQUE,
    platform    TEXT NOT NULL,
    push_token  TEXT NOT NULL,
    created_at  TIMESTAMP NOT NULL,
    updated_at  TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS rounds (
    id                     INTEGER PRIMARY KEY AUTOINCREMENT,
    public_id              TEXT NOT NULL UNIQUE,
    sent_at                TIMESTAMP NOT NULL,
    targeted_device_count  INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS deliveries (
    round_id    INTEGER NOT NULL REFERENCES rounds(id),
    device_id   INTEGER NOT NULL REFERENCES devices(id),
    sent_at     TIMESTAMP NOT NULL,
    received_at TIMESTAMP,
    latency_ms  INTEGER,
    status      TEXT NOT NULL DEFAULT 'pending',
    PRIMARY KEY (round_id, device_id)
);

CREATE INDEX IF NOT EXISTS idx_deliveries_round ON deliveries(round_id);
CREATE INDEX IF NOT EXISTS idx_rounds_sent_at ON rounds(sent_at);
