CREATE TABLE conversations (
    id              TEXT PRIMARY KEY,
    title           TEXT NOT NULL,
    anchor_kind     TEXT,
    anchor_id       TEXT,
    created_at      DATETIME NOT NULL,
    last_message_at DATETIME NOT NULL,
    CHECK (
        (anchor_kind IS NULL AND anchor_id IS NULL)
        OR
        (anchor_kind IS NOT NULL AND anchor_id IS NOT NULL AND anchor_kind IN ('card','document','group'))
    )
);

CREATE INDEX conversations_anchor_idx ON conversations(anchor_kind, anchor_id);
CREATE INDEX conversations_last_message_at_idx ON conversations(last_message_at DESC);
