-- v0.12: add review_grade + reviewed_at to cards. LGTM is the floor state
-- (viewing = LGTM), DIGESTED is proper comprehension, GRILLED is critical
-- interrogation. reviewed_at tracks when the grade first rose above LGTM.
--
-- The conversation-driven backfill (recount user messages per card, set
-- floor) runs as a Go-side one-shot on first startup — see
-- pkg/services/repository/backfill_v12.go. This SQL only adds the columns.

ALTER TABLE cards ADD COLUMN review_grade TEXT NOT NULL DEFAULT 'LGTM'
  CHECK (review_grade IN ('LGTM', 'DIGESTED', 'GRILLED'));
ALTER TABLE cards ADD COLUMN reviewed_at TIMESTAMP;
