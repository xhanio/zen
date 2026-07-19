DROP INDEX IF EXISTS groups_name_uniq;
ALTER TABLE groups ADD COLUMN parent_id TEXT REFERENCES groups(id) ON DELETE RESTRICT;
CREATE UNIQUE INDEX groups_sibling_name_uniq ON groups(IFNULL(parent_id, ''), name);
