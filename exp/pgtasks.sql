CREATE TABLE tasks (
    task_id UUID PRIMARY KEY NOT NULL DEFAULT uuid_generate_v4(),
    task_name TEXT NOT NULL,
    output JSONB,
    statuses JSONB[] NOT NULL DEFAULT '{}',
    for_user TEXT,
    expiry INTERVAL NOT NULL,
    state TEXT NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);