CREATE TABLE changelogs (
    version TEXT PRIMARY KEY,
    added TEXT[] NOT NULL,
    updated TEXT[] NOT NULL,
    removed TEXT[] NOT NULL,
    github_html TEXT, -- Filled in if not present
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);