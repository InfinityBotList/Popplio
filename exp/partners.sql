CREATE TABLE partners (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    short TEXT NOT NULL,
    user_id TEXT NOT NULL REFERENCES users(user_id),
    image TEXT NOT NULL,
    links jsonb NOT NULL,
    type TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);