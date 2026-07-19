DROP INDEX IF EXISTS cards_source_conversation_idx;
DROP INDEX IF EXISTS cards_parent_card_idx;
-- SQLite ≥ 3.35 supports ALTER TABLE DROP COLUMN; the production runtime
-- bundles SQLite ≥ 3.40 via mattn/go-sqlite3 v1.14+. Older local toolchains
-- will fail this down migration — accepted for v0.2.
ALTER TABLE cards DROP COLUMN source_conversation_id;
ALTER TABLE cards DROP COLUMN parent_card_id;
