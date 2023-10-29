CREATE TABLE vanity (
    itag UUID NOT NULL DEFAULT uuid_generate_v4(),
    target_id TEXT NOT NULL,
    target_type TEXT NOT NULL,
    code CITEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (target_id, target_type)
);

- After migration, add a foreign key to bot that references the vanity to ensure that all bots have a vanity
- This should be called vanity_ref