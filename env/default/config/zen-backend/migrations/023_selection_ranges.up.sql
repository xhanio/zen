-- v1.1.1: anchor a reference highlight to a character range in the card's
-- rendered text instead of searching for its text.
--
-- The range is captured by the SPA at drag time and recorded on the MESSAGE
-- (the only component that knows rendered offsets); a reference inherits it
-- via message_id. NULL start means "no range" — a permanent, supported state
-- for any reference not born from an SPA selection.
--
-- selection_seq is the card_snapshots.seq the selection was taken against. It
-- labels the reference; it is NOT used to decide whether to paint.

ALTER TABLE card_references ADD COLUMN selection_start INTEGER;
ALTER TABLE card_references ADD COLUMN selection_end   INTEGER;
ALTER TABLE card_references ADD COLUMN selection_seq   INTEGER;

ALTER TABLE messages ADD COLUMN selection_start INTEGER;
ALTER TABLE messages ADD COLUMN selection_end   INTEGER;
ALTER TABLE messages ADD COLUMN selection_seq   INTEGER;
