DROP TABLE card_tags;
DROP TABLE tags;

CREATE TABLE tags (
    id       TEXT PRIMARY KEY,
    group_id TEXT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    name     TEXT NOT NULL
);
CREATE UNIQUE INDEX tags_group_name_uniq ON tags(group_id, name);
CREATE INDEX tags_group_idx ON tags(group_id);

CREATE TABLE card_tags (
    card_id TEXT NOT NULL REFERENCES cards(id) ON DELETE CASCADE,
    tag_id  TEXT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (card_id, tag_id)
);
CREATE INDEX card_tags_tag_idx ON card_tags(tag_id);
