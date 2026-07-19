-- Revert to migration 011: FTS column named `content` (fed from search_hint).
-- This restores the snippet bug; it exists only so the migration is reversible.

DROP TRIGGER IF EXISTS cards_ai;
DROP TRIGGER IF EXISTS cards_ad;
DROP TRIGGER IF EXISTS cards_au;
DROP TABLE IF EXISTS cards_fts;

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

CREATE TRIGGER cards_ad AFTER DELETE ON cards
WHEN old.deleted_at IS NULL BEGIN
    INSERT INTO cards_fts(cards_fts, rowid, title, content)
    VALUES('delete', old.rowid, old.title, old.search_hint);
END;

CREATE TRIGGER cards_au AFTER UPDATE ON cards BEGIN
    INSERT INTO cards_fts(cards_fts, rowid, title, content)
    SELECT 'delete', old.rowid, old.title, old.search_hint
    WHERE old.deleted_at IS NULL;
    INSERT INTO cards_fts(rowid, title, content)
    SELECT new.rowid, new.title, new.search_hint
    WHERE new.deleted_at IS NULL;
END;

INSERT INTO cards_fts(rowid, title, content)
SELECT rowid, title, search_hint FROM cards WHERE deleted_at IS NULL;
