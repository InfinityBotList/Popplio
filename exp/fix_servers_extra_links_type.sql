-- Fixes servers.extra_links: it's still text[] on the live database, but the
-- application (and data/seed-ci.json's own schema conformance check) has
-- expected jsonb here for a while, matching bots/teams/users' extra_links
-- columns, which are all already jsonb. Every other table with this exact
-- []types.Link{Name, Value} pattern already made this switch; servers,
-- being a newer feature, never got the same migration.
--
-- Checked first: only one row currently has a non-null extra_links, and it's
-- empty ('{}'), so there is no real data at risk here. The USING clause
-- still handles a non-empty text[] safely by wrapping each element as a
-- bare JSON string, on the off chance a row was missed by that check
-- between now and when this runs — those wouldn't match the
-- {"name": ..., "value": ...} shape the Go code expects, so the NOTICE
-- below flags them instead of silently corrupting anything.
--
--   USAGE
--     psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f exp/fix_servers_extra_links_type.sql

\set ON_ERROR_STOP on

BEGIN;

DO $$
DECLARE
    row record;
    found boolean := false;
BEGIN
    FOR row IN
        SELECT server_id, extra_links
          FROM servers
         WHERE extra_links IS NOT NULL AND extra_links <> '{}'
    LOOP
        found := true;
        RAISE NOTICE 'servers.% has non-empty extra_links that will be converted to a JSON array of strings, not {name,value} objects: %', row.server_id, row.extra_links;
    END LOOP;

    IF NOT found THEN
        RAISE NOTICE 'No servers have non-empty extra_links; nothing to preserve.';
    END IF;
END
$$;

-- Matches bots.extra_links exactly (jsonb, NOT NULL, no default) per
-- data/seed-ci.json — not teams/users, which do have a '[]'::jsonb default.
ALTER TABLE servers
    ALTER COLUMN extra_links TYPE jsonb
    USING COALESCE(to_jsonb(extra_links), '[]'::jsonb);

ALTER TABLE servers
    ALTER COLUMN extra_links SET NOT NULL;

COMMIT;

\echo ''
\echo 'Done. servers.extra_links is now jsonb.'
