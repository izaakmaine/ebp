CREATE TABLE IF NOT EXISTS ebp_idea (
    id            TEXT PRIMARY KEY,
    owner         TEXT NOT NULL,
    claim         TEXT NOT NULL,
    source        TEXT,
    born          TEXT NOT NULL,
    contains_final_truth_claim INTEGER NOT NULL DEFAULT 0,
    promoted      INTEGER NOT NULL DEFAULT 0,
    promoted_at   TEXT,
    created_at    TEXT NOT NULL,
    updated_at    TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS ebp_debt_item (
    id         TEXT PRIMARY KEY,
    idea_id    TEXT NOT NULL REFERENCES ebp_idea(id),
    item       TEXT NOT NULL,
    added_at   TEXT NOT NULL,
    retired    INTEGER NOT NULL DEFAULT 0,
    retired_at TEXT,
    evidence   TEXT,
    UNIQUE(idea_id, item)
);

CREATE TABLE IF NOT EXISTS ebp_event (
    id         TEXT PRIMARY KEY,
    idea_id    TEXT NOT NULL REFERENCES ebp_idea(id),
    action     TEXT NOT NULL,
    detail     TEXT,
    actor      TEXT,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_debt_idea ON ebp_debt_item(idea_id);
CREATE INDEX IF NOT EXISTS idx_event_idea ON ebp_event(idea_id);
CREATE INDEX IF NOT EXISTS idx_event_action ON ebp_event(action);
