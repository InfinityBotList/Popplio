CREATE TABLE webhooks (
    id UUID NOT NULL DEFAULT uuid_generate_v4(),
    target_id TEXT NOT NULL,
    target_type TEXT NOT NULL,
    url TEXT NOT NULL CHECK (url <> ''),
    secret TEXT NOT NULL CHECK (secret <> ''),
    broken BOOLEAN NOT NULL DEFAULT FALSE, -- Whether or not the webhook is broken
    simple_auth BOOLEAN NOT NULL DEFAULT FALSE, -- Whether or not the webhook should use simple auth
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE (target_id, target_type)
);
