package teams

import (
	"context"
	"errors"
	"fmt"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/exp/slices"
)

// Permission is a set of permissions in a team
//
// # A permission consists of the entity followed by a dot and then a permission
//
// For example `bot.add` is the ability to add a bot
type Permission = string

const (
	// Ability to add new entity to a team
	PermissionAdd Permission = "add"

	// Ability to edit settings for the entity
	PermissionEdit Permission = "edit"

	// Ability to resubmit the entity
	PermissionResubmit Permission = "resubmit"

	// Ability to set a vanity for the entity
	PermissionSetVanity Permission = "set_vanity"

	// Ability to request certification for the entity
	PermissionRequestCertification Permission = "request_cert"

	// Ability to view existing API tokens for the entity
	PermissionViewAPITokens Permission = "view_api_tokens"

	// Ability to reset API tokens for the entity
	PermissionResetAPITokens Permission = "reset_api_tokens"

	// Ability to edit webhooks for the entity
	PermissionEditWebhooks Permission = "edit_webhooks"

	// Ability to test webhooks for the entity
	PermissionTestWebhooks Permission = "test_webhooks"

	// Ability to get the logs of a webhook
	PermissionGetWebhookLogs Permission = "get_webhook_logs"

	// Ability to delete the logs of a webhook
	PermissionDeleteWebhookLogs Permission = "delete_webhook_logs"

	// Ability to delete the entity
	PermissionDelete Permission = "delete"

	// Owner of the entity, this can either be global or entity specific
	PermissionOwner Permission = "*"
)

var PermDetails = []types.PermissionData{
	{
		ID:   PermissionAdd,
		Name: "Add {entity}",
		Desc: "Add new {entity} to the team or allow transferring {entity} to this team",
		SupportedEntities: []string{
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
			"bot",
			"server",
			"team",
			"team_member",
		},
	},
	{
		ID:                PermissionResubmit,
		Name:              "Resubmit {entity_plural}",
		Desc:              "Resubmit {entity_plural} on the team",
		SupportedEntities: []string{"bot", "server"},
	},
	{
		ID:                PermissionSetVanity,
		Name:              "Set {entity} vanity",
		Desc:              "Set vanity URL for {entity_plural} on the team",
		SupportedEntities: []string{"bot", "server"},
	},
	{
		ID:                PermissionRequestCertification,
		Name:              "Request Certification for {entity_plural}",
		Desc:              "Request certification for {entity_plural} on the team",
		SupportedEntities: []string{"bot"},
	},
	{
		ID:                PermissionViewAPITokens,
		Name:              "View Existing {entity} Tokens",
		Desc:              "View existing API tokens of {entity_plural} on the team. *DANGEROUS and a potential security risk*",
		SupportedEntities: []string{"bot", "server"},
	},
	{
		ID:                PermissionResetAPITokens,
		Name:              "Reset {entity} Tokens",
		Desc:              "Reset the API token of {entity_plural} on the team. This is seperate from viewing existing {entity} tokens as that is a much greater security risk",
		SupportedEntities: []string{"bot", "server"},
	},
	{
		ID:                PermissionEditWebhooks,
		Name:              "Edit {entity} Webhooks",
		Desc:              "Edit {entity} webhook settings. Note that 'Test {entity} Webhooks' is a separate permission and is required to test webhooks.",
		SupportedEntities: []string{"bot", "team", "server"},
	},
	{
		ID:                PermissionTestWebhooks,
		Name:              "Test {entity} Webhooks",
		Desc:              "Test {entity} webhooks. Note that this is a separate permission from 'Edit {entity} Webhooks' and is required to test webhooks.",
		SupportedEntities: []string{"bot", "team", "server"},
	},
	{
		ID:                PermissionGetWebhookLogs,
		Name:              "Get {entity} Webhook Logs",
		Desc:              "Get {entity} webhook logs.",
		SupportedEntities: []string{"bot", "team", "server"},
	},
	{
		ID:                PermissionDeleteWebhookLogs,
		Name:              "Delete {entity} Webhook Logs",
		Desc:              "Delete {entity} webhook logs. Usually requires 'Get {entity} Webhook Logs' to be useful.",
		SupportedEntities: []string{"bot", "team", "server"},
	},
	{
		ID:   PermissionDelete,
		Name: "Delete {entity}",
		Desc: "Delete {entity_plural} from the team. This is a very dangerous permission and should usually never be given to anyone.",
		SupportedEntities: []string{
			"bot",
			"server",
			"team_member",
		},
	},
	{
		ID:   PermissionOwner,
		Name: "{entity}: Owner",
		Desc: "Has full control on {entity_plural}. If this is a global permission, it will override all other permissions along with allow for deletions. If this is an entity specific permission, it will override all other permissions for that entity.",
		SupportedEntities: []string{
			"bot",
			"server",
			"team_member",
			"team",
			"global",
		},
	},
}

type PermMan struct {
	perms []string
}

// Resolves a permission into an entity and the perm name
func ResolvePerm(perm Permission) (string, string, bool) {
	pSplit := strings.Split(perm, ".")

	if len(pSplit) != 2 {
		return "", "", false
	}

	return pSplit[0], pSplit[1], true
}

// Returns whether a permission is valid or not
func IsValidPerm(perm Permission) bool {
	entity, flag, ok := ResolvePerm(perm)

	if !ok {
		return false
	}

	if flag == PermissionOwner {
		return true
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

// NewPermMan creates a new permission manager from a list of permissions
// It will remove duplicates and undefined permissions
func NewPermMan(perms []string) *PermMan {
	var uniquePerms = []string{}

	for _, perm := range perms {
		if !IsValidPerm(perm) {
			continue
		}

		if !slices.Contains(uniquePerms, perm) {
			uniquePerms = append(uniquePerms, perm)
		}
	}

	return &PermMan{perms: uniquePerms}
}

// Has returns if the user can perform a specific operation on an entity
func (f PermMan) Has(entity string, flag Permission) bool {
	for _, p := range f.perms {
		// From fastest to slowest
		if p == "global."+PermissionOwner || p == entity+"."+PermissionOwner || p == "global."+flag || p == entity+"."+flag {
			return true
		}
	}

	return false
}

// Has raw returns if the user can perform an operation based on a full permission name
func (f PermMan) HasRaw(flag string) bool {
	entity, flag, ok := ResolvePerm(flag)

	if !ok {
		return false
	}

	return f.Has(entity, flag)
}

func (f *PermMan) Add(flag Permission) {
	f.perms = append(f.perms, string(flag))
}
func (f *PermMan) Clear(flag Permission) {
	f.perms = []string{}
}
func (f PermMan) Perms() []string {
	return f.perms
}

// Returns the permission of an entity
//
// - If the entity is a bot, it will return the permissions of the bot's owner (or the permissions a user has on the team)
// - If the entity is a team, it will return the permissions the user has on the team
func GetEntityPerms(ctx context.Context, userId, targetType, targetId string) (*PermMan, error) {
	var teamId string

	switch targetType {
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
			if owner.String == userId {
				return NewPermMan([]string{"global." + PermissionOwner}), nil
			}

			return NewPermMan([]string{}), nil
		}

		teamId = teamOwner.String
	case "team":
		teamId = targetId
	default:
		return nil, fmt.Errorf("invalid target type")
	}

	// Handle teams
	if teamId == "" || !utils.IsValidUUID(teamId) {
		return nil, fmt.Errorf("invalid team id")
	}

	// Get the team member from the team
	var teamPerms []string
	err := state.Pool.QueryRow(ctx, "SELECT flags FROM team_members WHERE team_id = $1 AND user_id = $2", teamId, userId).Scan(&teamPerms)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("user not found in team")
	}

	if err != nil {
		return nil, fmt.Errorf("error finding team member: %v", err)
	}

	return NewPermMan(teamPerms), nil
}
