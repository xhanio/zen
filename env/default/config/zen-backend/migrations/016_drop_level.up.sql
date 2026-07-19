-- v0.10: level is now derived from the catalog via level_entry_id. The
-- startup backfill (RunV10Backfill) runs between 015 and 016; when this
-- migration lands, cards.level_entry_id already holds the identity.

ALTER TABLE cards DROP COLUMN level;
