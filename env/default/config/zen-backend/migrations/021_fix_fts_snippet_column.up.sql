-- Search snippets leaked raw HTML with misaligned <mark> highlights. cards_fts
-- is an external-content table (content='cards'), so snippet()/highlight()
-- reconstruct text from the cards column named like the FTS column. That column
-- was named `content`, so snippet() read cards.content (raw HTML) — but the
-- index is fed from search_hint (the HTML-stripped text). Index and
-- reconstruction diverged: snippets showed CSS/markup and the <mark>s landed on
-- offsets from the stripped text applied to the HTML, highlighting wrong tokens.
--
-- Fix: name the FTS column `search_hint` so reconstruction reads
-- cards.search_hint — the same stripped text the triggers index. External-content
-- column names are fixed at create time, so DROP+CREATE and repopulate. Triggers
-- keep migration 011's `deleted_at` guard on the delete side.

DROP TRIGGER IF EXISTS cards_ai;
DROP TRIGGER IF EXISTS cards_ad;
DROP TRIGGER IF EXISTS cards_au;
DROP TABLE IF EXISTS cards_fts;

CREATE VIRTUAL TABLE cards_fts USING fts5(
    title,
    search_hint,
    content='cards',
    content_rowid='rowid',
    tokenize='unicode61'
);

CREATE TRIGGER cards_ai AFTER INSERT ON cards
WHEN new.deleted_at IS NULL BEGIN
    INSERT INTO cards_fts(rowid, title, search_hint)
    VALUES (new.rowid, new.title, new.search_hint);
END;

CREATE TRIGGER cards_ad AFTER DELETE ON cards
WHEN old.deleted_at IS NULL BEGIN
    INSERT INTO cards_fts(cards_fts, rowid, title, search_hint)
    VALUES('delete', old.rowid, old.title, old.search_hint);
END;

CREATE TRIGGER cards_au AFTER UPDATE ON cards BEGIN
    INSERT INTO cards_fts(cards_fts, rowid, title, search_hint)
    SELECT 'delete', old.rowid, old.title, old.search_hint
    WHERE old.deleted_at IS NULL;
    INSERT INTO cards_fts(rowid, title, search_hint)
    SELECT new.rowid, new.title, new.search_hint
    WHERE new.deleted_at IS NULL;
END;

-- Repopulate the fresh index from every live card.
INSERT INTO cards_fts(rowid, title, search_hint)
SELECT rowid, title, search_hint FROM cards WHERE deleted_at IS NULL;
