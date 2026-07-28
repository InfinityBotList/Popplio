CREATE TABLE entity_votes (
    id UUID NOT NULL DEFAULT uuid_generate_v4(),
    target_id TEXT NOT NULL,
    target_type TEXT NOT NULL,
    author TEXT NOT NULL REFERENCES users(user_id) ON UPDATE CASCADE ON DELETE CASCADE,
    upvote BOOLEAN NOT NULL,
    void BOOLEAN NOT NULL DEFAULT FALSE,
    void_reason TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

POST-IMPL

- Remove pack_votes and votes tables