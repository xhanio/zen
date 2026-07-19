-- Rollback restores the column but not its data. Cards will read as
-- Unfiled (level_entry_id remains authoritative) until re-leveled.
ALTER TABLE cards ADD COLUMN level REAL;
