-- 1. Drop documents + its FTS5 mirror (table, triggers, vtable).
DROP TRIGGER IF EXISTS documents_ai;
DROP TRIGGER IF EXISTS documents_ad;
DROP TRIGGER IF EXISTS documents_au;
DROP TABLE   IF EXISTS documents_fts;
DROP TABLE   IF EXISTS documents;

-- 2. Cards: drop document_id index + column; add genesis + deleted_at.
DROP INDEX IF EXISTS cards_document_idx;
ALTER TABLE cards DROP COLUMN document_id;
ALTER TABLE cards ADD COLUMN genesis TEXT NOT NULL DEFAULT '';
ALTER TABLE cards ADD COLUMN deleted_at DATETIME;
CREATE INDEX cards_deleted_at_idx ON cards(deleted_at);

-- 3. Conversations: tighten anchor_kind. SQLite needs table rebuild for CHECK changes.
CREATE TABLE conversations_new (
    id              TEXT PRIMARY KEY,
    title           TEXT NOT NULL,
    anchor_kind     TEXT,
    anchor_id       TEXT,
    created_at      DATETIME NOT NULL,
    last_message_at DATETIME NOT NULL,
    CHECK (
        (anchor_kind IS NULL AND anchor_id IS NULL)
        OR
        (anchor_kind IS NOT NULL AND anchor_id IS NOT NULL AND anchor_kind IN ('card','group'))
    )
);
INSERT INTO conversations_new SELECT id, title, anchor_kind, anchor_id, created_at, last_message_at
  FROM conversations
  WHERE anchor_kind IS NULL OR anchor_kind IN ('card','group');
DROP TABLE conversations;
ALTER TABLE conversations_new RENAME TO conversations;
CREATE INDEX conversations_anchor_idx ON conversations(anchor_kind, anchor_id);
CREATE INDEX conversations_last_message_at_idx ON conversations(last_message_at DESC);

-- 4. Rebuild cards FTS5 to exclude soft-deleted rows.
DROP TRIGGER IF EXISTS cards_ai;
DROP TRIGGER IF EXISTS cards_ad;
DROP TRIGGER IF EXISTS cards_au;
DROP TABLE   IF EXISTS cards_fts;

CREATE VIRTUAL TABLE cards_fts USING fts5(
    title,
    content,
    content='cards',
    content_rowid='rowid',
    tokenize='unicode61'
);

CREATE TRIGGER cards_ai AFTER INSERT ON cards
WHEN new.deleted_at IS NULL BEGIN
    INSERT INTO cards_fts(rowid, title, content)
    VALUES (new.rowid, new.title, new.search_hint);
END;

CREATE TRIGGER cards_ad AFTER DELETE ON cards BEGIN
    INSERT INTO cards_fts(cards_fts, rowid, title, content)
    VALUES('delete', old.rowid, old.title, old.search_hint);
END;

CREATE TRIGGER cards_au AFTER UPDATE ON cards BEGIN
    INSERT INTO cards_fts(cards_fts, rowid, title, content)
    VALUES('delete', old.rowid, old.title, old.search_hint);
    INSERT INTO cards_fts(rowid, title, content)
    SELECT new.rowid, new.title, new.search_hint
    WHERE new.deleted_at IS NULL;
END;

INSERT INTO cards_fts(rowid, title, content)
    SELECT rowid, title, search_hint FROM cards WHERE deleted_at IS NULL;
