-- FTS5 virtual tables cannot be ALTERed in place. Drop the existing
-- cards_fts / documents_fts and their triggers, recreate against the new
-- search_hint column, and backfill from the source rows.

DROP TRIGGER IF EXISTS cards_ai;
DROP TRIGGER IF EXISTS cards_ad;
DROP TRIGGER IF EXISTS cards_au;
DROP TABLE   IF EXISTS cards_fts;

DROP TRIGGER IF EXISTS documents_ai;
DROP TRIGGER IF EXISTS documents_ad;
DROP TRIGGER IF EXISTS documents_au;
DROP TABLE   IF EXISTS documents_fts;

CREATE VIRTUAL TABLE cards_fts USING fts5(
    title,
    content,
    content='cards',
    content_rowid='rowid',
    tokenize='unicode61'
);

CREATE TRIGGER cards_ai AFTER INSERT ON cards BEGIN
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
    VALUES (new.rowid, new.title, new.search_hint);
END;

CREATE VIRTUAL TABLE documents_fts USING fts5(
    title,
    content,
    content='documents',
    content_rowid='rowid',
    tokenize='unicode61'
);

CREATE TRIGGER documents_ai AFTER INSERT ON documents BEGIN
    INSERT INTO documents_fts(rowid, title, content)
    VALUES (new.rowid, new.title, new.search_hint);
END;

CREATE TRIGGER documents_ad AFTER DELETE ON documents BEGIN
    INSERT INTO documents_fts(documents_fts, rowid, title, content)
    VALUES('delete', old.rowid, old.title, old.search_hint);
END;

CREATE TRIGGER documents_au AFTER UPDATE ON documents BEGIN
    INSERT INTO documents_fts(documents_fts, rowid, title, content)
    VALUES('delete', old.rowid, old.title, old.search_hint);
    INSERT INTO documents_fts(rowid, title, content)
    VALUES (new.rowid, new.title, new.search_hint);
END;

-- Backfill from existing rows.
INSERT INTO cards_fts(rowid, title, content)
    SELECT rowid, title, search_hint FROM cards;
INSERT INTO documents_fts(rowid, title, content)
    SELECT rowid, title, search_hint FROM documents;
