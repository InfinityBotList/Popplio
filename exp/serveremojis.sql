alter table servers add column show_emojis boolean not null default false;
alter table servers add column emojis jsonb not null default '[]'::jsonb;
alter table servers add column stickers jsonb not null default '[]'::jsonb;
alter table servers add column emojis_synced_at timestamptz;
