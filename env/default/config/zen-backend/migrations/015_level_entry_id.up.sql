-- v0.10: introduce a stable identity column linking each card to a catalog
-- entry in its group. Populated by a startup backfill; migration 016 drops
-- the old `level` column once backfill has run.

ALTER TABLE cards ADD COLUMN level_entry_id TEXT;
CREATE INDEX cards_level_entry_id_idx ON cards(level_entry_id);
