-- v0.8.2: add a short human-authored summary to each card. Displayed on
-- card tiles in place of the auto preview when non-empty; empty summary
-- falls back to the existing content-derived preview. Server enforces a
-- 30-word cap.

ALTER TABLE cards ADD COLUMN summary TEXT NOT NULL DEFAULT '';
