-- v1.1.0: Card snapshots — a full post-state copy of a card on every content
-- change, attributed to the conversation that caused it (if any). Diff is
-- computed at write time and cached on the row; the snapshots themselves are
-- the source of truth, so the diff can be recomputed by a later backfill.
--
-- NOTE: production SQLite runs WITHOUT the foreign_keys pragma, so the
-- ON DELETE CASCADE below never fires there. It documents intent and holds in
-- tests (testutil enables the pragma). Card purge deletes snapshots via an
-- explicit repository call — see DeleteSnapshotsForCard.

CREATE TABLE card_snapshots (
    id              TEXT PRIMARY KEY,
    card_id         TEXT NOT NULL REFERENCES cards(id) ON DELETE CASCADE,
    seq             INTEGER NOT NULL,
    title           TEXT NOT NULL,
    summary         TEXT NOT NULL DEFAULT '',
    content         TEXT NOT NULL DEFAULT '',
    format          TEXT NOT NULL,
    actor           TEXT NOT NULL DEFAULT 'user' CHECK (actor IN ('user','agent','system')),
    conversation_id TEXT,
    change_kind     TEXT NOT NULL CHECK (change_kind IN ('create','update','decompose','baseline')),
    diff            TEXT NOT NULL DEFAULT '',
    diff_truncated  INTEGER NOT NULL DEFAULT 0,
    lines_added     INTEGER NOT NULL DEFAULT 0,
    lines_removed   INTEGER NOT NULL DEFAULT 0,
    created_at      DATETIME NOT NULL
);

CREATE UNIQUE INDEX card_snapshots_card_seq_idx  ON card_snapshots(card_id, seq);
CREATE INDEX        card_snapshots_conv_time_idx ON card_snapshots(conversation_id, created_at);
