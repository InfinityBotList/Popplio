-- COMMIT;
-- =====================================================================
-- Strip retired permissions from every stored permission list.
--
-- Removes:
--   - view_shop
--   - manage_shop
--   - manage_bot_whitelist
--   - view_cdn
--   - manage_cdn
--
-- USAGE
--   psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f exp/rewrite/remove_retired_permissions.sql
--
-- Runs in one transaction and is idempotent.

\set ON_ERROR_STOP on

BEGIN;

CREATE TEMP TABLE retired_perms_before (
    source text NOT NULL,
    owner  text NOT NULL,
    perms  text[] NOT NULL
) ON COMMIT DROP;

INSERT INTO retired_perms_before (source, owner, perms)
SELECT 'staff_positions.perms', name, perms
  FROM staff_positions
 WHERE perms && ARRAY[
    'view_shop',
    'manage_shop',
    'manage_bot_whitelist',
    'view_cdn',
    'manage_cdn'
]
UNION ALL
SELECT 'staff_members.perm_overrides', user_id, perm_overrides
  FROM staff_members
 WHERE perm_overrides && ARRAY[
    'view_shop',
    'manage_shop',
    'manage_bot_whitelist',
    'view_cdn',
    'manage_cdn'
]
UNION ALL
SELECT 'staff_disciplinary_types.perm_limits', id::text, perm_limits
  FROM staff_disciplinary_types
 WHERE perm_limits && ARRAY[
    'view_shop',
    'manage_shop',
    'manage_bot_whitelist',
    'view_cdn',
    'manage_cdn'
];

\echo ''
\echo '=== BEFORE: rows holding retired permissions ==='

SELECT source, owner, perms
  FROM retired_perms_before
 ORDER BY source, owner;

UPDATE staff_positions
   SET perms = array_remove(
                   array_remove(
                   array_remove(
                   array_remove(
                   array_remove(
                       perms,
                       'view_shop'),
                       'manage_shop'),
                       'manage_bot_whitelist'),
                       'view_cdn'),
                       'manage_cdn')
 WHERE perms && ARRAY[
    'view_shop',
    'manage_shop',
    'manage_bot_whitelist',
    'view_cdn',
    'manage_cdn'
];

UPDATE staff_members
   SET perm_overrides = array_remove(
                            array_remove(
                            array_remove(
                            array_remove(
                            array_remove(
                                perm_overrides,
                                'view_shop'),
                                'manage_shop'),
                                'manage_bot_whitelist'),
                                'view_cdn'),
                                'manage_cdn')
 WHERE perm_overrides && ARRAY[
    'view_shop',
    'manage_shop',
    'manage_bot_whitelist',
    'view_cdn',
    'manage_cdn'
];

UPDATE staff_disciplinary_types
   SET perm_limits = array_remove(
                         array_remove(
                         array_remove(
                         array_remove(
                         array_remove(
                             perm_limits,
                             'view_shop'),
                             'manage_shop'),
                             'manage_bot_whitelist'),
                             'view_cdn'),
                             'manage_cdn')
 WHERE perm_limits && ARRAY[
    'view_shop',
    'manage_shop',
    'manage_bot_whitelist',
    'view_cdn',
    'manage_cdn'
];

DO $$
DECLARE
    remaining INT;
BEGIN
    SELECT
        (SELECT COUNT(*)
           FROM staff_positions
          WHERE perms && ARRAY[
              'view_shop',
              'manage_shop',
              'manage_bot_whitelist',
              'view_cdn',
              'manage_cdn'
          ])
      + (SELECT COUNT(*)
           FROM staff_members
          WHERE perm_overrides && ARRAY[
              'view_shop',
              'manage_shop',
              'manage_bot_whitelist',
              'view_cdn',
              'manage_cdn'
          ])
      + (SELECT COUNT(*)
           FROM staff_disciplinary_types
          WHERE perm_limits && ARRAY[
              'view_shop',
              'manage_shop',
              'manage_bot_whitelist',
              'view_cdn',
              'manage_cdn'
          ])
    INTO remaining;

    IF remaining <> 0 THEN
        RAISE EXCEPTION 'Retired permissions still present in % row(s)', remaining;
    END IF;

    RAISE NOTICE 'Retired permissions removed from % row(s)',
        (SELECT COUNT(*) FROM retired_perms_before);
END
$$;

COMMIT;

\echo ''
\echo 'Done. Nothing else changed; every other permission on those rows is untouched.'

-- =====================================================================
-- ROLLBACK
--
-- Only meaningful in the same session as the run above while the temp table
-- still exists.
--
-- BEGIN;
--
-- UPDATE staff_positions p
--    SET perms = b.perms
--   FROM retired_perms_before b
--  WHERE b.source = 'staff_positions.perms'
--    AND b.owner = p.name;
--
-- UPDATE staff_members m
--    SET perm_overrides = b.perms
--   FROM retired_perms_before b
--  WHERE b.source = 'staff_members.perm_overrides'
--    AND b.owner = m.user_id;
--
-- UPDATE staff_disciplinary_types d
--    SET perm_limits = b.perms
--   FROM retired_perms_before b
--  WHERE b.source = 'staff_disciplinary_types.perm_limits'
--    AND b.owner = d.id::text;
--
-- COMMIT;
-- =====================================================================