-- SQLite 3.35+ supports DROP COLUMN; the bundled mattn/go-sqlite3 ships well
-- past that.
ALTER TABLE messages DROP COLUMN selection_seq;
ALTER TABLE messages DROP COLUMN selection_end;
ALTER TABLE messages DROP COLUMN selection_start;

ALTER TABLE card_references DROP COLUMN selection_seq;
ALTER TABLE card_references DROP COLUMN selection_end;
ALTER TABLE card_references DROP COLUMN selection_start;
