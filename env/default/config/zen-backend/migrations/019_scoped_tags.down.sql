DROP TABLE card_tags;
DROP TABLE tags;

CREATE TABLE tags (
    id   TEXT PRIMARY KEY,
    name TEXT NOT NULL
);
CREATE UNIQUE INDEX tags_name_uniq ON tags(name);

CREATE TABLE card_tags (
    card_id TEXT NOT NULL REFERENCES cards(id) ON DELETE CASCADE,
    tag_id  TEXT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (card_id, tag_id)
);
CREATE INDEX card_tags_tag_idx ON card_tags(tag_id);
