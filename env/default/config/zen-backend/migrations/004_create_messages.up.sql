CREATE TABLE messages (
    id              TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    role            TEXT NOT NULL,
    content         TEXT NOT NULL,
    selection_text  TEXT,
    created_at      DATETIME NOT NULL,
    CHECK (role IN ('user','assistant','system')),
    CHECK (selection_text IS NULL OR role = 'user')
);

CREATE INDEX messages_conversation_idx ON messages(conversation_id, created_at);

CREATE VIRTUAL TABLE messages_fts USING fts5(
    role,
    content,
    content='messages',
    content_rowid='rowid',
    tokenize='unicode61'
);

CREATE TRIGGER messages_ai AFTER INSERT ON messages BEGIN
    INSERT INTO messages_fts(rowid, role, content)
    VALUES (new.rowid, new.role, new.content);
END;

CREATE TRIGGER messages_ad AFTER DELETE ON messages BEGIN
    INSERT INTO messages_fts(messages_fts, rowid, role, content)
    VALUES('delete', old.rowid, old.role, old.content);
END;

CREATE TRIGGER messages_au AFTER UPDATE ON messages BEGIN
    INSERT INTO messages_fts(messages_fts, rowid, role, content)
    VALUES('delete', old.rowid, old.role, old.content);
    INSERT INTO messages_fts(rowid, role, content)
    VALUES (new.rowid, new.role, new.content);
END;
