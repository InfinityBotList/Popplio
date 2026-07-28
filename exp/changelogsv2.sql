CREATE TABLE changelogs (
    version TEXT PRIMARY KEY,
    added TEXT[] NOT NULL,
    updated TEXT[] NOT NULL,
    removed TEXT[] NOT NULL,
    github_html TEXT, -- Filled in if not present
    extra_description TEXT NOT NULL DEFAULT '',
    prerelease boolean not null default false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);