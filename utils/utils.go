package utils

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"popplio/config"
	"popplio/state"
	"popplio/teams"
	"popplio/types"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/google/uuid"
	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/exp/slices"
)

var (
	indexBotColsArr = GetCols(types.IndexBot{})
	indexBotCols    = strings.Join(indexBotColsArr, ",")
)

// Returns if a string is empty/null or not. Used throughout the codebase
func IsNone(s string) bool {
	if s == "None" || s == "none" || s == "" || s == "null" {
		return true
	}
	return false
}

func ResolveTeam(ctx context.Context, teamId string) (*types.Team, error) {
	var name string
	var avatar string

	err := state.Pool.QueryRow(ctx, "SELECT name, avatar FROM teams WHERE id = $1", teamId).Scan(&name, &avatar)

	if err != nil {
		return nil, err
	}

	// Next handle members
	var members = []types.TeamMember{}

	rows, err := state.Pool.Query(ctx, "SELECT user_id, perms, created_at FROM team_members WHERE team_id = $1 ORDER BY created_at ASC", teamId)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var userId string
		var perms []types.TeamPermission
		var createdAt time.Time

		err = rows.Scan(&userId, &perms, &createdAt)

		if err != nil {
			return nil, err
		}

		user, err := dovewing.GetUser(ctx, userId, state.DovewingPlatformDiscord)

		if err != nil {
			return nil, err
		}

		members = append(members, types.TeamMember{
			User:      user,
			Perms:     teams.NewPermissionManager(perms).Perms(),
			CreatedAt: createdAt,
		})
	}

	// Bots
	var teamBotIds []string
	var bots = []types.IndexBot{}

	teamBotRows, err := state.Pool.Query(ctx, "SELECT bot_id FROM bots WHERE team_owner = $1", teamId)

	if err != nil {
		return nil, err
	}

	err = pgxscan.ScanAll(&teamBotIds, teamBotRows)

	if err != nil {
		return nil, err
	}

	// Loop over all bot IDs and create user bots from them
	for _, botId := range teamBotIds {
		indexBotsRows, err := state.Pool.Query(ctx, "SELECT "+indexBotCols+" FROM bots WHERE bot_id = $1", botId)

		if err != nil {
			return nil, err
		}

		var indexBot = types.IndexBot{}

		err = pgxscan.ScanOne(&indexBot, indexBotsRows)

		if err != nil {
			return nil, err
		}

		userObj, err := dovewing.GetUser(ctx, indexBot.BotID, state.DovewingPlatformDiscord)

		if err != nil {
			state.Logger.Error(err)
			continue
		}

		indexBot.User = userObj

		bots = append(bots, indexBot)
	}

	if err != nil {
		state.Logger.Error(err)
		return nil, err
	}

	return &types.Team{
		ID:       teamId,
		Name:     name,
		Avatar:   avatar,
		Members:  members,
		UserBots: bots,
	}, nil
}

func GetCols(s any) []string {
	refType := reflect.TypeOf(s)

	var cols []string

	for _, f := range reflect.VisibleFields(refType) {
		db := f.Tag.Get("db")
		reflectOpts := f.Tag.Get("reflect")

		if db == "-" || db == "" || reflectOpts == "ignore" {
			continue
		}

		// Do not allow even accidental fetches of tokens
		if db == "api_token" || db == "webhook_secret" {
			continue
		}

		cols = append(cols, db)
	}

	return cols
}

// Returns a permission manager of the permissions the user has on the bot
// Also takes teams into account if the bot is in a team
func GetUserBotPerms(ctx context.Context, userID string, botID string) (*teams.PermissionManager, error) {
	var teamOwner pgtype.Text
	var owner pgtype.Text
	err := state.Pool.QueryRow(ctx, "SELECT team_owner, owner FROM bots WHERE bot_id = $1", botID).Scan(&teamOwner, &owner)

	if err != nil {
		return &teams.PermissionManager{}, fmt.Errorf("error finding bot: %v", err)
	}

	// Handle teams
	if teamOwner.Valid && teamOwner.String != "" {
		// Get the team member from the team
		var teamPerms []types.TeamPermission

		err = state.Pool.QueryRow(ctx, "SELECT perms FROM team_members WHERE team_id = $1 AND user_id = $2", teamOwner, userID).Scan(&teamPerms)

		if err != nil {
			return &teams.PermissionManager{}, fmt.Errorf("error finding team member: %v", err)
		}

		return teams.NewPermissionManager(teamPerms), nil
	}

	if owner.String == userID {
		return teams.NewPermissionManager([]types.TeamPermission{teams.TeamPermissionOwner}), nil
	}

	return teams.NewPermissionManager([]types.TeamPermission{}), nil
}

func ClearUserCache(ctx context.Context, userId string) error {
	// Delete from cache
	state.Redis.Del(ctx, "uc-"+userId)

	return nil
}

func ClearBotCache(ctx context.Context, botId string) error {
	// Get name and vanity, delete from cache
	var vanity string
	var clientId string

	err := state.Pool.QueryRow(ctx, "SELECT lower(vanity), client_id FROM bots WHERE bot_id = $1", botId).Scan(&vanity, &clientId)

	if err != nil {
		return err
	}

	// Delete from cache
	for _, k := range []string{"bc-", "seob:"} {
		state.Redis.Del(ctx, k+vanity)
		state.Redis.Del(ctx, k+botId)
		state.Redis.Del(ctx, k+clientId)
	}
	return nil
}

func ValidateExtraLinks(links []types.Link) error {
	var public, private int

	if len(links) > 20 {
		return errors.New("you have too many links")
	}

	for _, link := range links {
		if strings.HasPrefix(link.Name, "_") {
			private++

			if len(link.Name) > 512 || len(link.Value) > 8192 {
				return errors.New("one of your private links has a name/value that is too long")
			}

			if strings.ReplaceAll(link.Name, " ", "") == "" || strings.ReplaceAll(link.Value, " ", "") == "" {
				return errors.New("one of your private links has a name/value that is empty")
			}
		} else {
			public++

			if len(link.Name) > 64 || len(link.Value) > 512 {
				return errors.New("one of your public links has a name/value that is too long")
			}

			if strings.ReplaceAll(link.Name, " ", "") == "" || strings.ReplaceAll(link.Value, " ", "") == "" {
				return errors.New("one of your public links has a name/value that is empty")
			}

			if !strings.HasPrefix(link.Value, "https://") {
				return errors.New("extra link '" + link.Name + "' must be HTTPS")
			}
		}

		for _, ch := range link.Name {
			allowedChars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_ "

			if !strings.ContainsRune(allowedChars, ch) {
				return errors.New("extra link '" + link.Name + "' has an invalid character: " + string(ch))
			}
		}
	}

	if public > 10 {
		return errors.New("you have too many public links")
	}

	if private > 10 {
		return errors.New("you have too many private links")
	}

	return nil
}

func ResolveBot(ctx context.Context, name string) (string, error) {
	resolveBotSQL := "(lower(vanity) = $1 OR bot_id = $1 OR client_id = $1)"

	// First check count so we can avoid expensive DB calls
	var count int64

	// We need to take the lower of the name because lower(vanity) = $1
	name = strings.ToLower(name)

	err := state.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM bots WHERE "+resolveBotSQL, name).Scan(&count)

	if err != nil {
		return "", err
	}

	if count == 0 {
		return "", nil
	}

	if count > 1 {
		// Delete one of the bots
		_, err := state.Pool.Exec(ctx, "DELETE FROM bots WHERE "+resolveBotSQL+" LIMIT 1", name)

		if err != nil {
			return "", err
		}
	}

	var id string
	err = state.Pool.QueryRow(ctx, "SELECT bot_id FROM bots WHERE "+resolveBotSQL, name).Scan(&id)

	if err != nil {
		return "", err
	}

	return id, nil
}

func IsValidUUID(u string) bool {
	_, err := uuid.Parse(u)
	return err == nil
}

func UUIDString(myUUID pgtype.UUID) string {
	return fmt.Sprintf("%x-%x-%x-%x-%x", myUUID.Bytes[0:4], myUUID.Bytes[4:6], myUUID.Bytes[6:8], myUUID.Bytes[8:10], myUUID.Bytes[10:16])
}

// For staging, ensure user is a hdev or owner
//
// This is because staging uses test keys
func StagingCheckSensitive(ctx context.Context, userId string) error {
	// For staging, ensure user is a hdev or owner
	//
	// This is because staging uses test keys
	if config.CurrentEnv == config.CurrentEnvStaging {
		var hdev bool
		var owner bool

		err := state.Pool.QueryRow(ctx, "SELECT iblhdev, owner FROM users WHERE user_id = $1", userId).Scan(&hdev, &owner)

		if err != nil {
			state.Logger.Error(err)
			return errors.New("unable to determine if user is staff")
		}

		if !hdev && !owner {
			return errors.New("user is not a hdev/owner while being in a staging/test environment")
		}
	}

	return nil
}

func CheckUserBoosterStatus(id string) types.BoosterStatus {
	// Check member is a booster
	m, err := state.Discord.State.Member(state.Config.Servers.Main, id)

	if err != nil {
		return types.BoosterStatus{
			Remark:    "Member not found on server:" + err.Error(),
			IsBooster: false,
		}
	}

	// Check if member has booster role
	roles := state.Config.Roles.PremiumRoles.Parse()
	for _, role := range m.Roles {
		if slices.Contains(roles, role) {
			// Member has booster role
			return types.BoosterStatus{
				IsBooster: true,
			}
		}
	}

	return types.BoosterStatus{
		Remark:    "Member does not have booster role",
		IsBooster: false,
	}
}
