CREATE TABLE groups (
    id         TEXT PRIMARY KEY,
    parent_id  TEXT REFERENCES groups(id) ON DELETE RESTRICT,
    name       TEXT NOT NULL,
    position   INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);

CREATE UNIQUE INDEX groups_sibling_name_uniq
    ON groups(IFNULL(parent_id, ''), name);

CREATE TABLE tags (
    id   TEXT PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE UNIQUE INDEX tags_name_uniq
    ON tags(name);

CREATE TABLE documents (
    id         TEXT PRIMARY KEY,
    title      TEXT NOT NULL,
    content    TEXT NOT NULL DEFAULT '',
    group_id   TEXT NOT NULL REFERENCES groups(id) ON DELETE RESTRICT,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);

CREATE INDEX documents_group_idx ON documents(group_id);

CREATE TABLE cards (
    id          TEXT PRIMARY KEY,
    title       TEXT NOT NULL,
    content     TEXT NOT NULL DEFAULT '',
    document_id TEXT REFERENCES documents(id) ON DELETE SET NULL,
    group_id    TEXT NOT NULL REFERENCES groups(id) ON DELETE RESTRICT,
    position    INTEGER NOT NULL DEFAULT 0,
    created_at  DATETIME NOT NULL,
    updated_at  DATETIME NOT NULL
);

CREATE INDEX cards_group_idx ON cards(group_id);
CREATE INDEX cards_document_idx ON cards(document_id);

CREATE TABLE card_tags (
    card_id TEXT NOT NULL REFERENCES cards(id) ON DELETE CASCADE,
    tag_id  TEXT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (card_id, tag_id)
);

CREATE INDEX card_tags_tag_idx ON card_tags(tag_id);
