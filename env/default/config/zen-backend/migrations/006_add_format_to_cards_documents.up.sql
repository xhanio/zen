ALTER TABLE cards ADD COLUMN format TEXT NOT NULL DEFAULT 'markdown'
    CHECK (format IN ('markdown', 'html', 'text'));
ALTER TABLE cards ADD COLUMN search_hint TEXT NOT NULL DEFAULT '';

ALTER TABLE documents ADD COLUMN format TEXT NOT NULL DEFAULT 'markdown'
    CHECK (format IN ('markdown', 'html', 'text'));
ALTER TABLE documents ADD COLUMN search_hint TEXT NOT NULL DEFAULT '';

-- Backfill: every existing row is markdown, so search_hint == content.
UPDATE cards     SET search_hint = content;
UPDATE documents SET search_hint = content;
