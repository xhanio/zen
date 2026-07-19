-- v0.7.0: References — first-class entity linking a source card + derived
-- card + conversation + verbatim selection text. AI-attested via explicit
-- reference.create after card.create. Cascade-deletes when any referent
-- is hard-purged; survives soft-deletes.

CREATE TABLE card_references (
    id              TEXT PRIMARY KEY,
    source_card_id  TEXT NOT NULL REFERENCES cards(id)         ON DELETE CASCADE,
    derived_card_id TEXT NOT NULL REFERENCES cards(id)         ON DELETE CASCADE,
    conversation_id TEXT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    selection_text  TEXT NOT NULL,
    created_at      DATETIME NOT NULL,
    CHECK (source_card_id <> derived_card_id),
    CHECK (length(selection_text) BETWEEN 1 AND 5000)
);

CREATE INDEX card_references_source_idx       ON card_references(source_card_id);
CREATE INDEX card_references_derived_idx      ON card_references(derived_card_id);
CREATE INDEX card_references_conversation_idx ON card_references(conversation_id);
