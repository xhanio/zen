DROP INDEX IF EXISTS groups_sibling_name_uniq;
ALTER TABLE groups DROP COLUMN parent_id;
CREATE UNIQUE INDEX groups_name_uniq ON groups(name);
