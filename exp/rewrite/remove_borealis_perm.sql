-- Strip the retired Borealis permission from every stored permission list.
--
-- `use_borealis` (and the pre-migration `borealis.*` it came from) gated the
-- Borealis service, which the port removed outright — see
-- arcadia/CONFORMANCE.md D11a. Nothing calls Borealis any more, so the
-- permission has gated nothing since; the catalogue no longer declares it.
--
-- This is needed because the permission model deliberately KEEPS names it does
-- not declare: an undeclared permission may belong to another service, so
-- `perms.Set` carries it through a read and a write rather than dropping it (see
-- Set.Undeclared). Left alone, `use_borealis` would sit in these columns for
-- good and show up under "Other services" in the staff bot and the panel.
--
--   USAGE
--     psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f exp/rewrite/remove_borealis_perm.sql
--
--   ON_ERROR_STOP=1 matters: without it psql keeps going after a failed
--   statement and would COMMIT a half-applied change.
--
-- Runs in one transaction and is idempotent: a second run reports zero rows.
-- Only staff-domain columns are touched — Borealis was never an entity
-- permission, so team_members.flags and api_sessions.perm_limits are left alone.
--
-- Losing this permission costs nobody anything: there is no Borealis to use.

\set ON_ERROR_STOP on

BEGIN;

CREATE TEMP TABLE borealis_before (
    source text NOT NULL,
    owner  text NOT NULL,
    perms  text[] NOT NULL
) ON COMMIT DROP;

-- Record what is about to change, so the report below is exact rather than an
-- estimate and there is something to read back if a row looks wrong afterwards.
INSERT INTO borealis_before (source, owner, perms)
SELECT 'staff_positions.perms', name, perms
  FROM staff_positions
 WHERE perms && ARRAY['use_borealis', 'borealis.*']
UNION ALL
SELECT 'staff_members.perm_overrides', user_id, perm_overrides
  FROM staff_members
 WHERE perm_overrides && ARRAY['use_borealis', 'borealis.*']
UNION ALL
SELECT 'staff_disciplinary_types.perm_limits', id::text, perm_limits
  FROM staff_disciplinary_types
 WHERE perm_limits && ARRAY['use_borealis', 'borealis.*'];

\echo ''
\echo '=== BEFORE: rows holding the Borealis permission ==='

SELECT source, owner, perms FROM borealis_before ORDER BY source, owner;

-- array_remove drops every occurrence and leaves an empty array rather than
-- NULL, which is what these columns expect.
UPDATE staff_positions
   SET perms = array_remove(array_remove(perms, 'use_borealis'), 'borealis.*')
 WHERE perms && ARRAY['use_borealis', 'borealis.*'];

UPDATE staff_members
   SET perm_overrides = array_remove(array_remove(perm_overrides, 'use_borealis'), 'borealis.*')
 WHERE perm_overrides && ARRAY['use_borealis', 'borealis.*'];

UPDATE staff_disciplinary_types
   SET perm_limits = array_remove(array_remove(perm_limits, 'use_borealis'), 'borealis.*')
 WHERE perm_limits && ARRAY['use_borealis', 'borealis.*'];

-- Verify before committing: any survivor aborts the transaction and leaves the
-- database exactly as it was.
DO $$
DECLARE
    remaining INT;
BEGIN
    SELECT
        (SELECT COUNT(*) FROM staff_positions WHERE perms && ARRAY['use_borealis', 'borealis.*'])
      + (SELECT COUNT(*) FROM staff_members WHERE perm_overrides && ARRAY['use_borealis', 'borealis.*'])
      + (SELECT COUNT(*) FROM staff_disciplinary_types WHERE perm_limits && ARRAY['use_borealis', 'borealis.*'])
    INTO remaining;

    IF remaining <> 0 THEN
        RAISE EXCEPTION 'Borealis permission still present in % row(s)', remaining;
    END IF;

    RAISE NOTICE 'Borealis permission removed from % row(s)', (SELECT COUNT(*) FROM borealis_before);
END
$$;

COMMIT;

\echo ''
\echo 'Done. Nothing else changed; every other permission on those rows is untouched.'

-- =====================================================================
-- ROLLBACK
--
-- Only meaningful in the same session as the run above, while the temp table
-- still exists. Otherwise put the values back from the BEFORE output by hand.
--
-- BEGIN;
--
-- UPDATE staff_positions p SET perms = b.perms
--   FROM borealis_before b
--  WHERE b.source = 'staff_positions.perms' AND b.owner = p.name;
--
-- UPDATE staff_members m SET perm_overrides = b.perms
--   FROM borealis_before b
--  WHERE b.source = 'staff_members.perm_overrides' AND b.owner = m.user_id;
--
-- UPDATE staff_disciplinary_types d SET perm_limits = b.perms
--   FROM borealis_before b
--  WHERE b.source = 'staff_disciplinary_types.perm_limits' AND b.owner = d.id::text;
--
-- COMMIT;
-- =====================================================================
