-- v0.7.1: References' conversation_id becomes optional. References created
-- from a non-chat path (SPA-driven derivations, future user-curated highlights)
-- should still anchor selections even without a conversation.

-- SQLite can't ALTER COLUMN; rebuild the table with the same FKs minus the
-- NOT NULL on conversation_id. CHECK constraints + indexes are reapplied.

CREATE TABLE card_references_new (
    id              TEXT PRIMARY KEY,
    source_card_id  TEXT NOT NULL REFERENCES cards(id)         ON DELETE CASCADE,
    derived_card_id TEXT NOT NULL REFERENCES cards(id)         ON DELETE CASCADE,
    conversation_id TEXT          REFERENCES conversations(id) ON DELETE CASCADE,
    selection_text  TEXT NOT NULL,
    created_at      DATETIME NOT NULL,
    CHECK (source_card_id <> derived_card_id),
    CHECK (length(selection_text) BETWEEN 1 AND 5000)
);

INSERT INTO card_references_new
    SELECT id, source_card_id, derived_card_id, conversation_id, selection_text, created_at
    FROM card_references;

DROP TABLE card_references;
ALTER TABLE card_references_new RENAME TO card_references;

CREATE INDEX card_references_source_idx       ON card_references(source_card_id);
CREATE INDEX card_references_derived_idx      ON card_references(derived_card_id);
CREATE INDEX card_references_conversation_idx ON card_references(conversation_id);
