-- v0.6.0's cards_fts triggers fail to handle soft-deleted rows symmetrically:
-- a row that was never indexed (because deleted_at was set when it landed)
-- still has its old.rowid sent to the 'delete' command on DELETE / UPDATE,
-- which corrupts the FTS5 index ("database disk image is malformed").
--
-- Fix: gate the delete-side of each trigger on old.deleted_at IS NULL so
-- we only ever ask FTS5 to delete rows it actually indexed.

-- Drop the FTS5 vtable and rebuild it from scratch. Any rows that the
-- broken triggers half-indexed have left page-level corruption ("database
-- disk image is malformed"); only a full DROP+CREATE clears it.
DROP TABLE IF EXISTS cards_fts;
CREATE VIRTUAL TABLE cards_fts USING fts5(
    title,
    content,
    content='cards',
    content_rowid='rowid',
    tokenize='unicode61'
);

DROP TRIGGER IF EXISTS cards_ai;
DROP TRIGGER IF EXISTS cards_ad;
DROP TRIGGER IF EXISTS cards_au;

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

-- Repopulate the fresh FTS5 index from every live card.
INSERT INTO cards_fts(rowid, title, content)
SELECT rowid, title, search_hint FROM cards WHERE deleted_at IS NULL;
