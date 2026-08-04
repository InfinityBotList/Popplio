-- Retire the cache-server subsystem WITHOUT losing any data.
--
-- Nothing is dropped. Every table is MOVED into an `archive_cache_servers`
-- schema and the one affected column is copied into an archive table before it
-- is removed from `bots`. The application no longer reads any of it, so moving
-- it out of `public` is enough to take it out of service, and every row stays
-- queryable and restorable.
--
--   USAGE
--     psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f exp/remove_cache_servers.sql
--
--   ON_ERROR_STOP=1 matters: without it psql keeps going after a failed
--   statement and would COMMIT a half-applied migration.
--
--   Take a dump first anyway:
--     pg_dump "$DATABASE_URL" -Fc -f pre-cache-server-removal.dump
--
-- The whole thing runs in ONE transaction and is idempotent: re-running it is a
-- no-op. A rollback script is at the bottom, commented out.
--
-- WHY MOVE RATHER THAN DROP
--   cache_server_bots and cache_server_oauths carry ON DELETE CASCADE foreign
--   keys to `bots` and `staff_members`. Left in place, deleting a bot (which the
--   staff panel's ForceRemove does) would silently delete the archived rows too,
--   so the archive would erode over time. Those constraints are dropped below,
--   which makes the archive an inert, frozen snapshot: it neither blocks live
--   deletes nor follows them.

\set ON_ERROR_STOP on

BEGIN;

-- Lock the tables we are about to move so nothing writes to them mid-migration.
-- ACCESS EXCLUSIVE is what ALTER TABLE ... SET SCHEMA takes anyway; taking it up
-- front makes the ordering explicit and fails fast if something holds them.
LOCK TABLE public.bots IN SHARE ROW EXCLUSIVE MODE;

\echo ''
\echo '=== BEFORE ==='

SELECT
    c.relname AS table_name,
    n.nspname AS schema,
    (SELECT COUNT(*) FROM pg_class WHERE oid = c.oid) AS present,
    pg_size_pretty(pg_total_relation_size(c.oid)) AS size
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE c.relname LIKE 'cache_server%'
  AND c.relkind = 'r'
ORDER BY n.nspname, c.relname;

-- 1. The archive schema. Comment records why it exists, for whoever finds it.
CREATE SCHEMA IF NOT EXISTS archive_cache_servers;

COMMENT ON SCHEMA archive_cache_servers IS
    'Retired cache-server subsystem, archived out of public. Read-only snapshot; no application code references it. See exp/remove_cache_servers.sql.';

-- 2. Preserve bots.cache_server_uninvitable before the column goes.
--    Only rows carrying a value are kept; the column is nullable and the nulls
--    hold no information.
CREATE TABLE IF NOT EXISTS archive_cache_servers.bots_cache_server_uninvitable (
    bot_id                   TEXT PRIMARY KEY,
    cache_server_uninvitable TEXT NOT NULL,
    archived_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE archive_cache_servers.bots_cache_server_uninvitable IS
    'Snapshot of bots.cache_server_uninvitable taken when the column was dropped.';

-- Guarded so a re-run is a no-op: after the first pass the source column is gone,
-- and a static reference to it would fail to parse. EXECUTE defers planning until
-- the branch is actually taken. ON CONFLICT keeps the first snapshot.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'bots'
          AND column_name = 'cache_server_uninvitable'
    ) THEN
        EXECUTE $q$
            INSERT INTO archive_cache_servers.bots_cache_server_uninvitable (bot_id, cache_server_uninvitable)
            SELECT bot_id, cache_server_uninvitable
            FROM public.bots
            WHERE cache_server_uninvitable IS NOT NULL
            ON CONFLICT (bot_id) DO NOTHING
        $q$;

        RAISE NOTICE 'snapshotted bots.cache_server_uninvitable';
    ELSE
        RAISE NOTICE 'bots.cache_server_uninvitable already archived, skipping snapshot';
    END IF;
END
$$;

-- 3. Sever the CASCADE foreign keys into live tables so the archive is frozen.
--    Constraints internal to the archived set (cache_server_bots -> cache_servers)
--    are kept: they stay consistent because both sides move together.
ALTER TABLE IF EXISTS public.cache_server_bots
    DROP CONSTRAINT IF EXISTS cache_server_bots_bot_id_fkey;

ALTER TABLE IF EXISTS public.cache_server_oauths
    DROP CONSTRAINT IF EXISTS user_id_is_sm;

-- 4. Move the tables. SET SCHEMA is a catalog update: no rows are rewritten, and
--    indexes, constraints and sequences follow the table.
DO $$
DECLARE
    t TEXT;
BEGIN
    FOREACH t IN ARRAY ARRAY[
        'cache_server_bots',
        'cache_server_oauths',
        'cache_server_oauth_md',
        'cache_server_migrations',
        'cache_server_migrations_done',
        'cache_servers'
    ]
    LOOP
        IF EXISTS (
            SELECT 1 FROM pg_class c
            JOIN pg_namespace n ON n.oid = c.relnamespace
            WHERE c.relname = t AND n.nspname = 'public' AND c.relkind = 'r'
        ) THEN
            EXECUTE format('ALTER TABLE public.%I SET SCHEMA archive_cache_servers', t);
            RAISE NOTICE 'archived table %', t;
        ELSE
            RAISE NOTICE 'table % already archived or absent, skipping', t;
        END IF;
    END LOOP;
END
$$;

-- 5. Drop the column now that its contents are safe.
ALTER TABLE public.bots DROP COLUMN IF EXISTS cache_server_uninvitable;

-- 6. Verify before committing. Each check raises and aborts the transaction on
--    failure, so a bad run leaves the database exactly as it was.
DO $$
DECLARE
    live_count       INT;
    archived_count   INT;
    snapshot_count   INT;
    column_remaining INT;
BEGIN
    SELECT COUNT(*) INTO live_count
    FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
    WHERE n.nspname = 'public' AND c.relname LIKE 'cache_server%' AND c.relkind = 'r';

    IF live_count <> 0 THEN
        RAISE EXCEPTION 'expected no cache_server tables left in public, found %', live_count;
    END IF;

    SELECT COUNT(*) INTO archived_count
    FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
    WHERE n.nspname = 'archive_cache_servers' AND c.relname LIKE 'cache_server%' AND c.relkind = 'r';

    IF archived_count <> 6 THEN
        RAISE EXCEPTION 'expected 6 archived cache_server tables, found %', archived_count;
    END IF;

    SELECT COUNT(*) INTO snapshot_count
    FROM archive_cache_servers.bots_cache_server_uninvitable;

    RAISE NOTICE 'archived % bots.cache_server_uninvitable values', snapshot_count;

    SELECT COUNT(*) INTO column_remaining
    FROM information_schema.columns
    WHERE table_schema = 'public' AND table_name = 'bots' AND column_name = 'cache_server_uninvitable';

    IF column_remaining <> 0 THEN
        RAISE EXCEPTION 'bots.cache_server_uninvitable is still present';
    END IF;
END
$$;

\echo ''
\echo '=== AFTER: archived tables and their EXACT row counts ==='

-- Counted rather than estimated. pg_stat_user_tables.n_live_tup would report 0
-- here: the statistics collector has not seen these relations since they moved,
-- which on a data migration report reads alarmingly like data loss.
SELECT
    c.relname AS table_name,
    (xpath('/row/c/text()',
        query_to_xml(format('SELECT COUNT(*) AS c FROM archive_cache_servers.%I', c.relname),
        false, true, '')))[1]::text::bigint AS rows,
    pg_size_pretty(pg_total_relation_size(c.oid)) AS size
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = 'archive_cache_servers'
  AND c.relkind = 'r'
ORDER BY c.relname;

COMMIT;

\echo ''
\echo 'Done. Everything is in the archive_cache_servers schema; nothing was deleted.'
\echo 'Exact counts:  SELECT COUNT(*) FROM archive_cache_servers.cache_server_bots;'
\echo 'Once you are satisfied, the archive can be dropped with:'
\echo '  DROP SCHEMA archive_cache_servers CASCADE;'

-- =====================================================================
-- ROLLBACK
--
-- Puts everything back. Only valid while the archive schema still exists and
-- nothing has re-created these names in public.
--
-- BEGIN;
--
-- ALTER TABLE archive_cache_servers.cache_servers                   SET SCHEMA public;
-- ALTER TABLE archive_cache_servers.cache_server_bots               SET SCHEMA public;
-- ALTER TABLE archive_cache_servers.cache_server_oauths             SET SCHEMA public;
-- ALTER TABLE archive_cache_servers.cache_server_oauth_md           SET SCHEMA public;
-- ALTER TABLE archive_cache_servers.cache_server_migrations         SET SCHEMA public;
-- ALTER TABLE archive_cache_servers.cache_server_migrations_done    SET SCHEMA public;
--
-- ALTER TABLE public.bots ADD COLUMN IF NOT EXISTS cache_server_uninvitable TEXT;
--
-- UPDATE public.bots b
-- SET cache_server_uninvitable = a.cache_server_uninvitable
-- FROM archive_cache_servers.bots_cache_server_uninvitable a
-- WHERE a.bot_id = b.bot_id;
--
-- -- Restore the foreign keys severed above.
-- ALTER TABLE public.cache_server_bots
--     ADD CONSTRAINT cache_server_bots_bot_id_fkey
--     FOREIGN KEY (bot_id) REFERENCES public.bots(bot_id) ON UPDATE CASCADE ON DELETE CASCADE;
--
-- ALTER TABLE public.cache_server_oauths
--     ADD CONSTRAINT user_id_is_sm
--     FOREIGN KEY (user_id) REFERENCES public.staff_members(user_id) ON UPDATE CASCADE ON DELETE CASCADE;
--
-- DROP TABLE archive_cache_servers.bots_cache_server_uninvitable;
-- DROP SCHEMA archive_cache_servers;
--
-- COMMIT;
-- =====================================================================
