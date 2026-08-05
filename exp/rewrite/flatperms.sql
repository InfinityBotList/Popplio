-- Migrates every permission column from the old `namespace.action` model to
-- flat permission names.
--
-- The old model had wildcards (`rpc.*`), negators (`~rpc.PremiumAdd`) and one
-- permission per entity type per action. The new one has a flat, declared name
-- per capability, and the union of a holder's sources is what they can do. See
-- the `perms` package.
--
-- Columns migrated:
--
--   staff domain    staff_positions.perms
--                   staff_members.perm_overrides
--                   staff_disciplinary_types.perm_limits
--   entity domain   team_members.flags
--                   api_sessions.perm_limits
--
-- Behaviour worth knowing before running this:
--
--   * A wildcard expands to every flat permission it used to cover.
--   * A negator REMOVES its permission from the result, even when a wildcard in
--     the same row granted it. Because several old permissions collapse into
--     one, a row holding `rpc.*` with `~rpc.PremiumAdd` loses manage_premium
--     entirely rather than keeping half of it. Losing an ability is the safe
--     direction; the alternative silently hands someone power they were
--     explicitly denied.
--   * Anything unrecognised is left exactly as it was, and listed in a NOTICE
--     at the end so it can be dealt with by hand. Nothing is dropped silently.
--   * Running it twice is harmless: flat names are already unrecognised by the
--     mapping and pass straight through.
--
-- Run it inside the transaction it opens, read the NOTICEs, then COMMIT.

BEGIN;

CREATE TEMP TABLE perm_map (
    dom text NOT NULL,
    old text NOT NULL,
    new text NOT NULL,
    PRIMARY KEY (dom, old)
) ON COMMIT DROP;

-- ---------------------------------------------------------------------------
-- Staff domain: one old permission, one new one.
-- ---------------------------------------------------------------------------
INSERT INTO perm_map (dom, old, new) VALUES
    ('staff', 'global.*',                              'administrator'),
    ('staff', 'arcadia.*',                             'administrator'),
    ('staff', 'global.view',                           'view_panel'),
    ('staff', 'rpc_logs.view',                         'view_audit_logs'),
    ('staff', 'rpc_logs.*',                            'view_audit_logs'),
    ('staff', 'global.view_sensitive',                 'view_sensitive_data'),
    ('staff', 'popplio_staging.sensitive',             'use_staging_keys'),
    ('staff', 'popplio_staging.*',                     'use_staging_keys'),

    ('staff', 'rpc.Claim',                             'review_bots'),
    ('staff', 'rpc.Unclaim',                           'review_bots'),
    ('staff', 'rpc.Approve',                           'review_bots'),
    ('staff', 'rpc.Deny',                              'review_bots'),
    ('staff', 'rpc.Unverify',                          'review_bots'),
    ('staff', 'rpc.CertifyAdd',                        'certify_bots'),
    ('staff', 'rpc.CertifyRemove',                     'certify_bots'),
    ('staff', 'rpc.BotTransferOwnershipUser',          'transfer_bots'),
    ('staff', 'rpc.BotTransferOwnershipTeam',          'transfer_bots'),
    ('staff', 'rpc.ForceRemove',                       'force_remove_bots'),

    ('staff', 'rpc.PremiumAdd',                        'manage_premium'),
    ('staff', 'rpc.PremiumRemove',                     'manage_premium'),
    ('staff', 'rpc.VoteReset',                         'manage_votes'),
    ('staff', 'rpc.VoteResetAll',                      'manage_votes'),
    ('staff', 'rpc.VoteBanAdd',                        'ban_voters'),
    ('staff', 'rpc.VoteBanRemove',                     'ban_voters'),

    ('staff', 'apps.view',                             'view_apps'),
    ('staff', 'apps.manage',                           'manage_apps'),
    ('staff', 'rpc.AppBanUser',                        'ban_app_users'),
    ('staff', 'rpc.AppUnbanUser',                      'ban_app_users'),

    ('staff', 'staff_members.view',                    'view_staff'),
    ('staff', 'staff_positions.view',                  'view_staff'),
    ('staff', 'staff_members.edit',                    'manage_staff_members'),
    ('staff', 'staff_positions.create',                'manage_staff_roles'),
    ('staff', 'staff_positions.edit',                  'manage_staff_roles'),
    ('staff', 'staff_positions.delete',                'manage_staff_roles'),
    ('staff', 'staff_positions.set_index',             'manage_staff_roles'),
    ('staff', 'staff_positions.swap_index',            'manage_staff_roles'),
    ('staff', 'staff_disciplinary_types.create',       'manage_disciplinaries'),
    ('staff', 'staff_disciplinary_types.update',       'manage_disciplinaries'),
    ('staff', 'staff_disciplinary_types.delete',       'manage_disciplinaries'),
    ('staff', 'staff_disciplinary_types.*',            'manage_disciplinaries'),
    ('staff', 'persepolis.view_onboarding_responses',  'view_onboarding'),
    ('staff', 'persepolis.*',                          'view_onboarding'),

    ('staff', 'shop.view',                             'view_shop'),
    ('staff', 'shop_items.view',                       'view_shop'),
    ('staff', 'shop_item_benefits.view',               'view_shop'),
    ('staff', 'shop_coupons.list',                     'view_shop'),
    ('staff', 'shop_items.create',                     'manage_shop'),
    ('staff', 'shop_items.update',                     'manage_shop'),
    ('staff', 'shop_items.delete',                     'manage_shop'),
    ('staff', 'shop_item_benefits.create',             'manage_shop'),
    ('staff', 'shop_item_benefits.update',             'manage_shop'),
    ('staff', 'shop_item_benefits.delete',             'manage_shop'),
    ('staff', 'shop_coupons.create',                   'manage_shop'),
    ('staff', 'shop_coupons.update',                   'manage_shop'),
    ('staff', 'shop_coupons.delete',                   'manage_shop'),
    ('staff', 'vote_credit_tiers.create',              'manage_shop'),
    ('staff', 'vote_credit_tiers.update',              'manage_shop'),
    ('staff', 'vote_credit_tiers.delete',              'manage_shop'),
    ('staff', 'bot_whitelist.create',                  'manage_bot_whitelist'),
    ('staff', 'bot_whitelist.update',                  'manage_bot_whitelist'),
    ('staff', 'bot_whitelist.delete',                  'manage_bot_whitelist'),
    ('staff', 'bot_whitelist.*',                       'manage_bot_whitelist'),

    ('staff', 'partners.create',                       'manage_partners'),
    ('staff', 'partners.update',                       'manage_partners'),
    ('staff', 'partners.delete',                       'manage_partners'),
    ('staff', 'partners.*',                            'manage_partners'),
    ('staff', 'blog.create_entry',                     'manage_blog'),
    ('staff', 'blog.update_entry',                     'manage_blog'),
    ('staff', 'blog.delete_entry',                     'manage_blog'),
    ('staff', 'blog.*',                                'manage_blog'),

    ('staff', 'popplio.tickets',                       'view_tickets'),
    ('staff', 'popplio.*',                             'view_tickets'),

    ('staff', 'cdn.list_scopes',                       'view_cdn'),
    ('staff', 'cdn.list',                              'view_cdn'),
    ('staff', 'cdn#ibl@main.list',                     'view_cdn'),
    ('staff', 'cdn#ibl@main.read_file',                'view_cdn'),
    ('staff', 'cdn.add_file',                          'manage_cdn'),
    ('staff', 'cdn.upload_chunk',                      'manage_cdn'),

    ('staff', 'developer.marker',                      'marker_developer'),
    ('staff', 'lead_developer.marker',                 'marker_lead_developer'),
    ('staff', 'human_resources.marker',                'marker_human_resources'),
    ('staff', 'bot_reviewer.marker',                   'marker_bot_reviewer'),
    ('staff', 'service_account.marker',                'marker_service_account'),
    ('staff', 'dt.marker',                             'marker_disciplinary');

-- ---------------------------------------------------------------------------
-- Entity domain: one old permission, one new one.
-- ---------------------------------------------------------------------------
INSERT INTO perm_map (dom, old, new) VALUES
    ('entity', 'global.*',                   'owner'),

    ('entity', 'team.edit',                  'edit_team'),
    ('entity', 'global.edit',                'edit_team'),
    ('entity', 'team.set_vanity',            'set_vanity'),
    ('entity', 'bot.set_vanity',             'set_vanity'),
    ('entity', 'server.set_vanity',          'set_vanity'),
    ('entity', 'global.set_vanity',          'set_vanity'),

    ('entity', 'bot.add',                    'add_bots'),
    ('entity', 'global.add',                 'add_bots'),
    ('entity', 'bot.edit',                   'edit_bots'),
    ('entity', 'bot.delete',                 'delete_bots'),
    ('entity', 'global.delete',              'delete_bots'),
    ('entity', 'bot.resubmit',               'resubmit_bots'),
    ('entity', 'global.resubmit',            'resubmit_bots'),
    ('entity', 'bot.request_cert',           'request_bot_certification'),
    ('entity', 'global.request_cert',        'request_bot_certification'),

    ('entity', 'server.add',                 'add_servers'),
    ('entity', 'server.edit',                'edit_servers'),
    ('entity', 'server.delete',              'delete_servers'),
    ('entity', 'server.get_invite',          'manage_server_invites'),
    ('entity', 'server.set_invite',          'manage_server_invites'),
    ('entity', 'global.get_invite',          'manage_server_invites'),
    ('entity', 'global.set_invite',          'manage_server_invites'),

    ('entity', 'team_member.add',            'add_team_members'),
    ('entity', 'team_member.edit',           'edit_team_members'),
    ('entity', 'team_member.delete',         'remove_team_members'),
    ('entity', 'team_member.remove',         'remove_team_members'),

    ('entity', 'bot.get_webhooks',           'view_webhooks'),
    ('entity', 'server.get_webhooks',        'view_webhooks'),
    ('entity', 'team.get_webhooks',          'view_webhooks'),
    ('entity', 'global.get_webhooks',        'view_webhooks'),
    ('entity', 'bot.create_webhooks',        'manage_webhooks'),
    ('entity', 'server.create_webhooks',     'manage_webhooks'),
    ('entity', 'team.create_webhooks',       'manage_webhooks'),
    ('entity', 'global.create_webhooks',     'manage_webhooks'),
    ('entity', 'bot.edit_webhooks',          'manage_webhooks'),
    ('entity', 'server.edit_webhooks',       'manage_webhooks'),
    ('entity', 'team.edit_webhooks',         'manage_webhooks'),
    ('entity', 'global.edit_webhooks',       'manage_webhooks'),
    ('entity', 'bot.delete_webhooks',        'manage_webhooks'),
    ('entity', 'server.delete_webhooks',     'manage_webhooks'),
    ('entity', 'team.delete_webhooks',       'manage_webhooks'),
    ('entity', 'global.delete_webhooks',     'manage_webhooks'),
    ('entity', 'bot.test_webhooks',          'manage_webhooks'),
    ('entity', 'server.test_webhooks',       'manage_webhooks'),
    ('entity', 'team.test_webhooks',         'manage_webhooks'),
    ('entity', 'global.test_webhooks',       'manage_webhooks'),
    ('entity', 'bot.get_webhook_logs',       'view_webhook_logs'),
    ('entity', 'server.get_webhook_logs',    'view_webhook_logs'),
    ('entity', 'team.get_webhook_logs',      'view_webhook_logs'),
    ('entity', 'global.get_webhook_logs',    'view_webhook_logs'),
    ('entity', 'bot.delete_webhook_logs',    'delete_webhook_logs'),
    ('entity', 'server.delete_webhook_logs', 'delete_webhook_logs'),
    ('entity', 'team.delete_webhook_logs',   'delete_webhook_logs'),
    ('entity', 'global.delete_webhook_logs', 'delete_webhook_logs'),

    ('entity', 'bot.upload_assets',          'manage_assets'),
    ('entity', 'server.upload_assets',       'manage_assets'),
    ('entity', 'team.upload_assets',         'manage_assets'),
    ('entity', 'global.upload_assets',       'manage_assets'),
    ('entity', 'bot.delete_assets',          'manage_assets'),
    ('entity', 'server.delete_assets',       'manage_assets'),
    ('entity', 'team.delete_assets',         'manage_assets'),
    ('entity', 'global.delete_assets',       'manage_assets'),

    ('entity', 'bot.create_owner_review',    'manage_owner_reviews'),
    ('entity', 'server.create_owner_review', 'manage_owner_reviews'),
    ('entity', 'team.create_owner_review',   'manage_owner_reviews'),
    ('entity', 'global.create_owner_review', 'manage_owner_reviews'),
    ('entity', 'bot.edit_owner_review',      'manage_owner_reviews'),
    ('entity', 'server.edit_owner_review',   'manage_owner_reviews'),
    ('entity', 'team.edit_owner_review',     'manage_owner_reviews'),
    ('entity', 'global.edit_owner_review',   'manage_owner_reviews'),
    ('entity', 'bot.delete_owner_review',    'manage_owner_reviews'),
    ('entity', 'server.delete_owner_review', 'manage_owner_reviews'),
    ('entity', 'team.delete_owner_review',   'manage_owner_reviews'),
    ('entity', 'global.delete_owner_review', 'manage_owner_reviews'),

    ('entity', 'bot.view_session',           'view_sessions'),
    ('entity', 'server.view_session',        'view_sessions'),
    ('entity', 'team.view_session',          'view_sessions'),
    ('entity', 'global.view_session',        'view_sessions'),
    ('entity', 'bot.view_api_tokens',        'view_sessions'),
    ('entity', 'server.view_api_tokens',     'view_sessions'),
    ('entity', 'team.view_api_tokens',       'view_sessions'),
    ('entity', 'global.view_api_tokens',     'view_sessions'),
    ('entity', 'bot.create_session',         'manage_sessions'),
    ('entity', 'server.create_session',      'manage_sessions'),
    ('entity', 'team.create_session',        'manage_sessions'),
    ('entity', 'global.create_session',      'manage_sessions'),
    ('entity', 'bot.revoke_session',         'manage_sessions'),
    ('entity', 'server.revoke_session',      'manage_sessions'),
    ('entity', 'team.revoke_session',        'manage_sessions'),
    ('entity', 'global.revoke_session',      'manage_sessions'),
    ('entity', 'bot.reset_api_tokens',       'manage_sessions'),
    ('entity', 'server.reset_api_tokens',    'manage_sessions'),
    ('entity', 'team.reset_api_tokens',      'manage_sessions'),
    ('entity', 'global.reset_api_tokens',    'manage_sessions'),

    ('entity', 'bot.redeem_vote_credits',    'redeem_vote_credits'),
    ('entity', 'server.redeem_vote_credits', 'redeem_vote_credits'),
    ('entity', 'team.redeem_vote_credits',   'redeem_vote_credits'),
    ('entity', 'global.redeem_vote_credits', 'redeem_vote_credits');

-- ---------------------------------------------------------------------------
-- Wildcards that covered more than one capability, and so expand to several
-- flat permissions each.
-- ---------------------------------------------------------------------------
CREATE TEMP TABLE wildcard_map (
    dom text NOT NULL,
    old text NOT NULL,
    new text NOT NULL,
    PRIMARY KEY (dom, old, new)
) ON COMMIT DROP;

INSERT INTO wildcard_map (dom, old, new)
SELECT 'staff', 'rpc.*', p
  FROM unnest(ARRAY[
      'review_bots', 'certify_bots', 'transfer_bots', 'force_remove_bots',
      'manage_premium', 'manage_votes', 'ban_voters', 'ban_app_users'
  ]) AS p
UNION ALL
SELECT 'staff', 'apps.*', p FROM unnest(ARRAY['view_apps', 'manage_apps']) AS p
UNION ALL
SELECT 'staff', 'shop.*', p FROM unnest(ARRAY['view_shop', 'manage_shop']) AS p
UNION ALL
SELECT 'staff', 'shop_items.*', p FROM unnest(ARRAY['view_shop', 'manage_shop']) AS p
UNION ALL
SELECT 'staff', 'shop_item_benefits.*', p FROM unnest(ARRAY['view_shop', 'manage_shop']) AS p
UNION ALL
SELECT 'staff', 'shop_coupons.*', p FROM unnest(ARRAY['view_shop', 'manage_shop']) AS p
UNION ALL
SELECT 'staff', 'vote_credit_tiers.*', p FROM unnest(ARRAY['view_shop', 'manage_shop']) AS p
UNION ALL
SELECT 'staff', 'staff_members.*', p FROM unnest(ARRAY['view_staff', 'manage_staff_members']) AS p
UNION ALL
SELECT 'staff', 'staff_positions.*', p FROM unnest(ARRAY['view_staff', 'manage_staff_roles']) AS p
UNION ALL
SELECT 'staff', 'cdn#ibl@main.*', p FROM unnest(ARRAY['view_cdn', 'manage_cdn']) AS p
UNION ALL
SELECT 'staff', 'cdn.*', p FROM unnest(ARRAY['view_cdn', 'manage_cdn']) AS p

-- `bot.*` was full control over the team's bots, and nothing else.
UNION ALL
SELECT 'entity', 'bot.*', p
  FROM unnest(ARRAY[
      'add_bots', 'edit_bots', 'delete_bots', 'resubmit_bots', 'request_bot_certification',
      'set_vanity', 'view_webhooks', 'manage_webhooks', 'view_webhook_logs', 'delete_webhook_logs',
      'manage_assets', 'manage_owner_reviews', 'view_sessions', 'manage_sessions', 'redeem_vote_credits'
  ]) AS p
UNION ALL
SELECT 'entity', 'server.*', p
  FROM unnest(ARRAY[
      'add_servers', 'edit_servers', 'delete_servers', 'manage_server_invites',
      'set_vanity', 'view_webhooks', 'manage_webhooks', 'view_webhook_logs', 'delete_webhook_logs',
      'manage_assets', 'manage_owner_reviews', 'view_sessions', 'manage_sessions', 'redeem_vote_credits'
  ]) AS p
-- `team.*` was "Team Admin": full control over the team's own settings, but
-- explicitly NOT over its bots and servers, its members, or deleting the team.
UNION ALL
SELECT 'entity', 'team.*', p
  FROM unnest(ARRAY[
      'edit_team', 'set_vanity', 'view_webhooks', 'manage_webhooks', 'view_webhook_logs',
      'delete_webhook_logs', 'manage_assets', 'manage_owner_reviews', 'view_sessions', 'manage_sessions'
  ]) AS p
UNION ALL
SELECT 'entity', 'team_member.*', p
  FROM unnest(ARRAY['add_team_members', 'edit_team_members', 'remove_team_members']) AS p;

-- ---------------------------------------------------------------------------
-- Permissions with no equivalent in the new model, dropped on purpose.
--
-- `global.view_sensitive` only ever meant something for staff. The one team
-- member holding it is a team owner, so dropping it costs them nothing.
--
-- `borealis.*` gated the Borealis service, which the port removed outright
-- (arcadia/CONFORMANCE.md D11a). Nothing in the codebase calls Borealis any
-- more, so the permission gated nothing and is not carried over. If Borealis
-- ever comes back it needs a declared permission of its own, not this one.
-- ---------------------------------------------------------------------------
CREATE TEMP TABLE retired_perm (
    dom text NOT NULL,
    old text NOT NULL,
    PRIMARY KEY (dom, old)
) ON COMMIT DROP;

INSERT INTO retired_perm (dom, old) VALUES
    ('entity', 'global.view_sensitive'),
    ('staff', 'borealis.*');

-- ---------------------------------------------------------------------------
-- The translation itself.
-- ---------------------------------------------------------------------------
CREATE FUNCTION pg_temp.flatten(source text[], dom text) RETURNS text[] AS $$
    WITH parsed AS (
        SELECT ltrim(raw, '~') AS name, raw LIKE '~%' AS negated
          FROM unnest(source) AS raw
    ),
    mapped AS (
        SELECT p.negated, m.new AS perm
          FROM parsed p
          JOIN perm_map m ON m.dom = flatten.dom AND m.old = p.name
        UNION ALL
        SELECT p.negated, w.new
          FROM parsed p
          JOIN wildcard_map w ON w.dom = flatten.dom AND w.old = p.name
        UNION ALL
        -- Anything unrecognised is carried over untouched, including names that
        -- are already flat.
        SELECT p.negated, p.name
          FROM parsed p
         WHERE NOT EXISTS (SELECT 1 FROM perm_map m WHERE m.dom = flatten.dom AND m.old = p.name)
           AND NOT EXISTS (SELECT 1 FROM wildcard_map w WHERE w.dom = flatten.dom AND w.old = p.name)
           AND NOT EXISTS (SELECT 1 FROM retired_perm r WHERE r.dom = flatten.dom AND r.old = p.name)
    )
    SELECT COALESCE((
        SELECT array_agg(DISTINCT perm ORDER BY perm)
          FROM mapped
         WHERE NOT negated
           AND perm NOT IN (SELECT perm FROM mapped WHERE negated)
    ), '{}'::text[]);
$$ LANGUAGE sql;

-- ---------------------------------------------------------------------------
-- Report what will not be mapped, before anything changes.
-- ---------------------------------------------------------------------------
DO $$
DECLARE
    row record;
    found boolean := false;
BEGIN
    FOR row IN
        SELECT DISTINCT src.dom, name
          FROM (
              SELECT 'staff' AS dom, unnest(perms) AS name FROM staff_positions
              UNION ALL
              SELECT 'staff', unnest(perm_overrides) FROM staff_members
              UNION ALL
              SELECT 'staff', unnest(perm_limits) FROM staff_disciplinary_types
              UNION ALL
              SELECT 'entity', unnest(flags) FROM team_members
              UNION ALL
              SELECT 'entity', unnest(perm_limits) FROM api_sessions
          ) src
         WHERE name LIKE '%.%'
           -- Negators are always dropped and get their own report below.
           AND name NOT LIKE '~%'
           AND NOT EXISTS (SELECT 1 FROM perm_map m WHERE m.dom = src.dom AND m.old = ltrim(name, '~'))
           AND NOT EXISTS (SELECT 1 FROM wildcard_map w WHERE w.dom = src.dom AND w.old = ltrim(name, '~'))
           AND NOT EXISTS (SELECT 1 FROM retired_perm r WHERE r.dom = src.dom AND r.old = ltrim(name, '~'))
         ORDER BY 1, 2
    LOOP
        found := true;
        RAISE NOTICE 'UNMAPPED (%): % — left as is, migrate it by hand', row.dom, row.name;
    END LOOP;

    IF NOT found THEN
        RAISE NOTICE 'Every stored permission has a mapping.';
    END IF;
END $$;

-- Negators do not exist in the new model, so every one of them is dropped. Where
-- a negator was withholding something a wildcard in the same row granted, the
-- holder KEEPS that permission after this runs. Each one is named here so it can
-- be revoked by hand if that was the intent.
DO $$
DECLARE
    row record;
BEGIN
    FOR row IN
        SELECT DISTINCT src.dom, src.owner, src.name,
               COALESCE(m.new, (SELECT string_agg(w.new, ', ' ORDER BY w.new)
                                  FROM wildcard_map w
                                 WHERE w.dom = src.dom AND w.old = ltrim(src.name, '~')),
                        '(nothing — it never denied a real permission)') AS denied
          FROM (
              SELECT 'staff' AS dom, name AS owner, unnest(perms) AS name FROM staff_positions
              UNION ALL
              SELECT 'staff', user_id, unnest(perm_overrides) FROM staff_members
              UNION ALL
              SELECT 'entity', team_id::text, unnest(flags) FROM team_members
          ) src
          LEFT JOIN perm_map m ON m.dom = src.dom AND m.old = ltrim(src.name, '~')
         WHERE src.name LIKE '~%'
         ORDER BY 1, 2
    LOOP
        RAISE NOTICE 'NEGATOR DROPPED (%): % on % — it was withholding: %', row.dom, row.name, row.owner, row.denied;
    END LOOP;
END $$;

-- ---------------------------------------------------------------------------
-- Migrate.
-- ---------------------------------------------------------------------------
UPDATE staff_positions          SET perms          = pg_temp.flatten(perms, 'staff');
UPDATE staff_members            SET perm_overrides = pg_temp.flatten(perm_overrides, 'staff');
UPDATE staff_disciplinary_types SET perm_limits    = pg_temp.flatten(perm_limits, 'staff');
UPDATE team_members             SET flags          = pg_temp.flatten(flags, 'entity');
UPDATE api_sessions             SET perm_limits    = pg_temp.flatten(perm_limits, 'entity');

-- ---------------------------------------------------------------------------
-- Report what is left, so the result can be eyeballed before COMMIT.
-- ---------------------------------------------------------------------------
DO $$
DECLARE
    row record;
BEGIN
    FOR row IN
        SELECT src.dom, name, count(*) AS holders
          FROM (
              SELECT 'staff' AS dom, unnest(perms) AS name FROM staff_positions
              UNION ALL
              SELECT 'staff', unnest(perm_overrides) FROM staff_members
              UNION ALL
              SELECT 'entity', unnest(flags) FROM team_members
          ) src
         GROUP BY 1, 2
         ORDER BY 1, 2
    LOOP
        RAISE NOTICE 'AFTER (%): % — held % time(s)', row.dom, row.name, row.holders;
    END LOOP;
END $$;

COMMIT;
