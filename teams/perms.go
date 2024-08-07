package teams

import (
	"context"
	"errors"
	"fmt"
	"popplio/state"
	"popplio/types"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/exp/slices"

	perms "github.com/infinitybotlist/kittycat/go"
)

// Permission is a set of permissions in a team
//
// # A permission consists of the entity followed by a dot and then a permission
//
// For example `bot.add` is the ability to add a bot
type Permission = string

const (
	// Ability to add a new entity
	PermissionAdd Permission = "add"

	// Ability to edit settings for the entity
	PermissionEdit Permission = "edit"

	// Ability to get an invite to the entity
	PermissionGetInvite Permission = "get_invite"

	// Ability to set an invite for the entity
	PermissionSetInvite Permission = "set_invite"

	// Ability to resubmit the entity
	PermissionResubmit Permission = "resubmit"

	// Ability to set a vanity for the entity
	PermissionSetVanity Permission = "set_vanity"

	// Ability to request certification for the entity
	PermissionRequestCertification Permission = "request_cert"

	// Ability to get webhooks of the entity
	PermissionGetWebhooks Permission = "get_webhooks"

	// Ability to create webhooks of the entity
	PermissionCreateWebhooks Permission = "create_webhooks"

	// Ability to edit webhooks of the entity
	PermissionEditWebhooks Permission = "edit_webhooks"

	// Ability to delete webhooks of the entity
	PermissionDeleteWebhooks Permission = "delete_webhooks"

	// Ability to test webhooks for the entity
	PermissionTestWebhooks Permission = "test_webhooks"

	// Ability to get the logs of a webhook
	PermissionGetWebhookLogs Permission = "get_webhook_logs"

	// Ability to delete the logs of a webhook
	PermissionDeleteWebhookLogs Permission = "delete_webhook_logs"

	// Ability to upload assets for the entity
	PermissionUploadAssets Permission = "upload_assets"

	// Ability to delete assets for the entity
	PermissionDeleteAssets Permission = "delete_assets"

	// Ability to create owner reviews for the entity
	PermissionCreateOwnerReview Permission = "create_owner_review"

	// Ability to edit owner reviews for the entity
	PermissionEditOwnerReview Permission = "edit_owner_review"

	// Ability to delete owner reviews for the entity
	PermissionDeleteOwnerReview Permission = "delete_owner_review"

	// Ability to view sessions for the entity
	PermissionViewSession Permission = "view_session"

	// Ability to view sessions for the entity
	PermissionCreateSession Permission = "create_session"

	// Ability to revoke sessions for the entity
	PermissionRevokeSession Permission = "revoke_session"

	// Ability to redeem vote credits for the entity
	PermissionRedeemVoteCredits Permission = "redeem_vote_credits"

	// Ability to delete the entity
	PermissionDelete Permission = "delete"

	// Owner of the entity, this can either be global or entity specific
	//
	// Note that this must be '*' for kittycat resolution
	PermissionOwner Permission = "*"
)

var PermDetails = []types.PermissionData{
	{
		ID:   PermissionAdd,
		Name: "Add {entity}",
		Desc: "Add new {entity} to the team or allow transferring {entity} to this team",
		SupportedEntities: []string{
			"global",
			"bot",
			"server",
			"team_member",
		},
	},
	{
		ID:   PermissionEdit,
		Name: "Edit {entity}",
		Desc: "Edit settings for the {entity}",
		SupportedEntities: []string{
			"global",
			"bot",
			"server",
			"team",
			"team_member",
		},
	},
	{
		ID:   PermissionGetInvite,
		Name: "Get {entity} Invite",
		Desc: "Ability to get an invite to the {entity} (e.g. non-public {entity} etc.)",
		SupportedEntities: []string{
			"global",
			"server",
		},
	},
	{
		ID:   PermissionSetInvite,
		Name: "Set {entity} Invite",
		Desc: "Ability to set the invite for an {entity} (e.g. non-public {entity} etc.)",
		SupportedEntities: []string{
			"global",
			"server",
		},
	},
	{
		ID:                PermissionResubmit,
		Name:              "Resubmit {entity}",
		Desc:              "Resubmit {entity} on the team",
		SupportedEntities: []string{"bot", "server", "global"},
	},
	{
		ID:                PermissionSetVanity,
		Name:              "Set {entity} Vanity",
		Desc:              "Set vanity URL for a {entity} on the team",
		SupportedEntities: []string{"bot", "team", "server", "global"},
	},
	{
		ID:                PermissionRequestCertification,
		Name:              "Request Certification for {entity}",
		Desc:              "Request certification for a {entity} on the team",
		SupportedEntities: []string{"bot", "global"},
	},
	{
		ID:                PermissionGetWebhooks,
		Name:              "Get {entity} Webhooks",
		Desc:              "Get {entity} webhooks. This is independent of updating them (you can still create/edit without this permission using the API)",
		SupportedEntities: []string{"bot", "team", "server", "global"},
	},
	{
		ID:                PermissionCreateWebhooks,
		Name:              "Create {entity} Webhooks",
		Desc:              "Create {entity} webhooks. Note that 'Test {entity} Webhooks' is a separate permission and is required to test webhooks.",
		SupportedEntities: []string{"bot", "team", "server", "global"},
	},
	{
		ID:                PermissionEditWebhooks,
		Name:              "Edit {entity} Webhooks",
		Desc:              "Edit {entity} webhooks. Note that 'Test {entity} Webhooks' is a separate permission and is required to test webhooks.",
		SupportedEntities: []string{"bot", "team", "server", "global"},
	},
	{
		ID:                PermissionDeleteWebhooks,
		Name:              "Delete {entity} Webhooks",
		Desc:              "Delete {entity} webhooks. Note that 'Test {entity} Webhooks' is a separate permission and is required to test webhooks.",
		SupportedEntities: []string{"bot", "team", "server", "global"},
	},
	{
		ID:                PermissionTestWebhooks,
		Name:              "Test {entity} Webhooks",
		Desc:              "Test {entity} webhooks. Note that this is a separate permission from 'Edit {entity} Webhooks' and is required to test webhooks.",
		SupportedEntities: []string{"bot", "team", "server", "global"},
	},
	{
		ID:                PermissionGetWebhookLogs,
		Name:              "Get {entity} Webhook Logs",
		Desc:              "Get {entity} webhook logs.",
		SupportedEntities: []string{"bot", "team", "server", "global"},
	},
	{
		ID:                PermissionDeleteWebhookLogs,
		Name:              "Delete {entity} Webhook Logs",
		Desc:              "Delete {entity} webhook logs. Usually requires 'Get {entity} Webhook Logs' to be useful.",
		SupportedEntities: []string{"bot", "team", "server", "global"},
	},
	{
		ID:                PermissionUploadAssets,
		Name:              "Upload {entity} Assets",
		Desc:              "Upload assets for {entity} on the team",
		SupportedEntities: []string{"bot", "server", "team", "global"},
	},
	{
		ID:                PermissionDeleteAssets,
		Name:              "Delete {entity} Assets",
		Desc:              "Delete assets for {entity} on the team",
		SupportedEntities: []string{"bot", "server", "team", "global"},
	},
	{
		ID:   PermissionCreateOwnerReview,
		Name: "Create {entity} Owner Review",
		Desc: "Create an owner review for {entity} on the team.",
		SupportedEntities: []string{
			"bot",
			"server",
			"team",
			"global",
		},
	},
	{
		ID:   PermissionEditOwnerReview,
		Name: "Edit {entity} Owner Review",
		Desc: "Edit an owner review for {entity} on the team.",
		SupportedEntities: []string{
			"bot",
			"server",
			"team",
			"global",
		},
	},
	{
		ID:   PermissionDeleteOwnerReview,
		Name: "Delete {entity} Owner Review",
		Desc: "Delete an owner review for {entity} on the team.",
		SupportedEntities: []string{
			"bot",
			"server",
			"team",
			"global",
		},
	},
	{
		ID:   PermissionViewSession,
		Name: "View {entity} Sessions",
		Desc: "View sessions associated with the {entity}.",
		SupportedEntities: []string{
			"bot",
			"server",
			"team",
			"global",
		},
	},
	{
		ID:   PermissionCreateSession,
		Name: "Create {entity} Sessions",
		Desc: "Create new sessions for the {entity} allowing for API (programmatic) access to the entity.",
		SupportedEntities: []string{
			"bot",
			"server",
			"team",
			"global",
		},
	},
	{
		ID:   PermissionRevokeSession,
		Name: "Revoke {entity} Sessions",
		Desc: "Revoke existing sessions on the {entity}.",
		SupportedEntities: []string{
			"bot",
			"server",
			"team",
			"global",
		},
	},
	{
		ID:   PermissionRedeemVoteCredits,
		Name: "Redeem Vote Credits ({entity})",
		Desc: "Redeem vote credits for the entity",
		SupportedEntities: []string{
			"bot",
			"server",
			"global",
		},
	},
	{
		ID:   PermissionDelete,
		Name: "Delete {entity}",
		Desc: "Delete a {entity} from the team. This is a very dangerous permission and should usually never be given to anyone.",
		SupportedEntities: []string{
			"bot",
			"server",
			"team_member",
			"global",
		},
		DataOverride: map[string]*types.PermissionDataOverride{
			"global": {
				Name: "Delete Any",
				Desc: "Delete any entity from the team other than the entity itself. This is a very dangerous permission and should usually never be given to anyone.",
			},
		},
	},
	{
		ID:   PermissionOwner,
		Name: "{entity} Admin",
		Desc: "Has full control on {entity}'s.",
		SupportedEntities: []string{
			"bot",
			"server",
			"team_member",
			"team",
			"global",
		},
		DataOverride: map[string]*types.PermissionDataOverride{
			"global": {
				Name: "Global Owner",
				Desc: "Full control. This overrides all other permissions and also allows deletion of the team. This is a very dangerous permission and should usually never be given to anyone.",
			},
			"team": {
				Name: "Team Admin",
				Desc: "Has full control on team settings listed under this tab. Note that this DOES NOT allow deleting the team or managing entities within the team itself",
			},
		},
	},
}

// Parses a permission into an entity and the perm name
func ParsePerm(perm Permission) (string, string, bool) {
	pSplit := strings.Split(perm, ".")

	if len(pSplit) != 2 {
		return "", "", false
	}

	// Remove negators if present
	pSplit[0] = strings.TrimPrefix(pSplit[0], "~")

	return pSplit[0], pSplit[1], true
}

// Returns whether a permission is valid or not
func IsValidPerm(perm Permission) bool {
	entity, flag, ok := ParsePerm(perm)

	if !ok {
		return false
	}

	for _, p := range PermDetails {
		if p.ID == flag {
			if entity == "" || slices.Contains(p.SupportedEntities, entity) {
				return true
			} else {
				return false
			}
		}
	}

	return false
}

// Returns the permission of an entity
//
// - If the entity is a bot, it will return the permissions of the bot's owner (or the permissions a user has on the team)
// - If the entity is a team, it will return the permissions the user has on the team
//
// The returned positions are a resolved set of kittycat permissions that can then be used with standard
// kittycat functions
func GetEntityPerms(ctx context.Context, userId, targetType, targetId string) ([]perms.Permission, error) {
	var teamId string

	switch targetType {
	case "user":
		// Special case
		if targetId != userId {
			return nil, fmt.Errorf("users do not have permissions on other users")
		}

		return []perms.Permission{
			{
				Namespace: "global",
				Perm:      PermissionOwner,
			},
		}, nil
	case "bot":
		var teamOwner pgtype.Text
		var owner pgtype.Text
		err := state.Pool.QueryRow(ctx, "SELECT team_owner, owner FROM bots WHERE bot_id = $1", targetId).Scan(&teamOwner, &owner)

		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("bot not found")
		}

		if err != nil {
			return nil, fmt.Errorf("error finding bot: %v", err)
		}

		if owner.Valid {
			// Fast path, we dont even need to perform kittycat
			// permission resolution here
			if owner.String == userId {
				return []perms.Permission{
					{
						Namespace: "global",
						Perm:      PermissionOwner,
					},
				}, nil
			}

			return []perms.Permission{}, nil
		}

		teamId = teamOwner.String
	case "team":
		teamId = targetId
	case "server":
		var teamOwner pgtype.Text
		err := state.Pool.QueryRow(ctx, "SELECT team_owner FROM servers WHERE server_id = $1", targetId).Scan(&teamOwner)

		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("server not found")
		}

		if err != nil {
			return nil, fmt.Errorf("error finding server: %v", err)
		}

		teamId = teamOwner.String
	default:
		return nil, fmt.Errorf("invalid target type")
	}

	// Handle teams
	if _, err := uuid.Parse(teamId); err != nil {
		return nil, fmt.Errorf("invalid team id")
	}

	// Get the team member from the team
	var teamPerms []string
	err := state.Pool.QueryRow(ctx, "SELECT flags FROM team_members WHERE team_id = $1 AND user_id = $2", teamId, userId).Scan(&teamPerms)

	if errors.Is(err, pgx.ErrNoRows) {
		return []perms.Permission{}, nil
	}

	if err != nil {
		return nil, fmt.Errorf("error finding team member: %v", err)
	}

	if len(teamPerms) == 0 {
		// Skip resolution, just return an empty array
		return []perms.Permission{}, nil
	}

	// Right now, team permissions are treated as permission overrides
	var resolvedPerms = perms.StaffPermissions{
		PermOverrides: perms.PFSS(teamPerms),
	}.Resolve()

	return resolvedPerms, nil
}
