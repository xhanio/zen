ALTER TABLE cards ADD COLUMN parent_card_id TEXT REFERENCES cards(id) ON DELETE SET NULL;
ALTER TABLE cards ADD COLUMN source_conversation_id TEXT REFERENCES conversations(id) ON DELETE SET NULL;

CREATE INDEX cards_parent_card_idx ON cards(parent_card_id);
CREATE INDEX cards_source_conversation_idx ON cards(source_conversation_id);
