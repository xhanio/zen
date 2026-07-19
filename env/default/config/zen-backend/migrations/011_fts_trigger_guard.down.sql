-- Revert to v0.6.0 migration 010 triggers (un-guarded delete side).
DROP TRIGGER IF EXISTS cards_ai;
DROP TRIGGER IF EXISTS cards_ad;
DROP TRIGGER IF EXISTS cards_au;

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
