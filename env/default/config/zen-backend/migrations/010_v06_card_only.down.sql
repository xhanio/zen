-- 1. Rebuild conversations with 'document' anchor allowed.
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
        (anchor_kind IS NOT NULL AND anchor_id IS NOT NULL AND anchor_kind IN ('card','document','group'))
    )
);
INSERT INTO conversations_new SELECT * FROM conversations;
DROP TABLE conversations;
ALTER TABLE conversations_new RENAME TO conversations;
CREATE INDEX conversations_anchor_idx ON conversations(anchor_kind, anchor_id);
CREATE INDEX conversations_last_message_at_idx ON conversations(last_message_at DESC);

-- 2. Restore documents table + FTS5 mirror.
CREATE TABLE documents (
    id          TEXT PRIMARY KEY,
    title       TEXT NOT NULL,
    content     TEXT NOT NULL DEFAULT '',
    format      TEXT NOT NULL DEFAULT 'markdown' CHECK (format IN ('markdown','html','text')),
    search_hint TEXT NOT NULL DEFAULT '',
    group_id    TEXT NOT NULL REFERENCES groups(id) ON DELETE RESTRICT,
    created_at  DATETIME NOT NULL,
    updated_at  DATETIME NOT NULL
);
CREATE INDEX documents_group_idx ON documents(group_id);

CREATE VIRTUAL TABLE documents_fts USING fts5(
    title, content, content='documents', content_rowid='rowid', tokenize='unicode61'
);
CREATE TRIGGER documents_ai AFTER INSERT ON documents BEGIN
    INSERT INTO documents_fts(rowid, title, content) VALUES (new.rowid, new.title, new.search_hint);
END;
CREATE TRIGGER documents_ad AFTER DELETE ON documents BEGIN
    INSERT INTO documents_fts(documents_fts, rowid, title, content)
    VALUES('delete', old.rowid, old.title, old.search_hint);
END;
CREATE TRIGGER documents_au AFTER UPDATE ON documents BEGIN
    INSERT INTO documents_fts(documents_fts, rowid, title, content)
    VALUES('delete', old.rowid, old.title, old.search_hint);
    INSERT INTO documents_fts(rowid, title, content) VALUES (new.rowid, new.title, new.search_hint);
END;

-- 3. Cards: drop new columns, restore document_id.
DROP INDEX IF EXISTS cards_deleted_at_idx;
ALTER TABLE cards DROP COLUMN deleted_at;
ALTER TABLE cards DROP COLUMN genesis;
ALTER TABLE cards ADD COLUMN document_id TEXT REFERENCES documents(id) ON DELETE SET NULL;
CREATE INDEX cards_document_idx ON cards(document_id);

-- 4. Restore cards FTS to v0.5 form (no deleted_at gate).
DROP TRIGGER IF EXISTS cards_ai;
DROP TRIGGER IF EXISTS cards_ad;
DROP TRIGGER IF EXISTS cards_au;
DROP TABLE   IF EXISTS cards_fts;

CREATE VIRTUAL TABLE cards_fts USING fts5(
    title, content, content='cards', content_rowid='rowid', tokenize='unicode61'
);
CREATE TRIGGER cards_ai AFTER INSERT ON cards BEGIN
    INSERT INTO cards_fts(rowid, title, content) VALUES (new.rowid, new.title, new.search_hint);
END;
CREATE TRIGGER cards_ad AFTER DELETE ON cards BEGIN
    INSERT INTO cards_fts(cards_fts, rowid, title, content)
    VALUES('delete', old.rowid, old.title, old.search_hint);
END;
CREATE TRIGGER cards_au AFTER UPDATE ON cards BEGIN
    INSERT INTO cards_fts(cards_fts, rowid, title, content)
    VALUES('delete', old.rowid, old.title, old.search_hint);
    INSERT INTO cards_fts(rowid, title, content) VALUES (new.rowid, new.title, new.search_hint);
END;
INSERT INTO cards_fts(rowid, title, content) SELECT rowid, title, search_hint FROM cards;
