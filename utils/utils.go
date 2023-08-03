package utils

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"popplio/state"
	"popplio/types"

	"github.com/google/uuid"
	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	indexBotColsArr = GetCols(types.IndexBot{})
	indexBotCols    = strings.Join(indexBotColsArr, ",")
)

type userTeamBotId struct {
	BotID string `db:"bot_id"`
}

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

	rows, err := state.Pool.Query(ctx, "SELECT itag, user_id, flags, created_at, mentionable FROM team_members WHERE team_id = $1 ORDER BY created_at ASC", teamId)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var itag pgtype.UUID
		var userId string
		var flags []string
		var createdAt time.Time
		var mentionable bool

		err = rows.Scan(&itag, &userId, &flags, &createdAt, &mentionable)

		if err != nil {
			return nil, err
		}

		user, err := dovewing.GetUser(ctx, userId, state.DovewingPlatformDiscord)

		if err != nil {
			return nil, err
		}

		members = append(members, types.TeamMember{
			ITag:        itag,
			User:        user,
			Flags:       flags,
			CreatedAt:   createdAt,
			Mentionable: mentionable,
		})
	}

	// Bots
	var bots = []types.IndexBot{}

	teamBotRows, err := state.Pool.Query(ctx, "SELECT bot_id FROM bots WHERE team_owner = $1", teamId)

	if err != nil {
		return nil, err
	}

	teamBotIds, err := pgx.CollectRows(teamBotRows, pgx.RowToStructByName[userTeamBotId])

	// Loop over all bot IDs and create user bots from them
	for _, botId := range teamBotIds {
		indexBotsRows, err := state.Pool.Query(ctx, "SELECT "+indexBotCols+" FROM bots WHERE bot_id = $1", botId.BotID)

		if err != nil {
			return nil, err
		}

		indexBot, err := pgx.CollectOneRow(indexBotsRows, pgx.RowToStructByName[types.IndexBot])

		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}

		if err != nil {
			return nil, err
		}

		indexBot.User, err = dovewing.GetUser(ctx, indexBot.BotID, state.DovewingPlatformDiscord)

		if err != nil {
			state.Logger.Error(err)
			return nil, err
		}

		var code string

		err = state.Pool.QueryRow(ctx, "SELECT code FROM vanity WHERE itag = $1", indexBot.VanityRef).Scan(&code)

		if err != nil {
			state.Logger.Error(err)
			return nil, err
		}

		indexBot.Vanity = code

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

func ClearUserCache(ctx context.Context, userId string) error {
	// Delete from cache
	state.Redis.Del(ctx, "uc-"+userId)

	return nil
}

func ClearBotCache(ctx context.Context, botId string) error {
	// Delete from cache
	for _, k := range []string{"bc-", "seob:"} {
		state.Redis.Del(ctx, k+botId)
	}
	return nil
}

func IsValidUUID(u string) bool {
	_, err := uuid.Parse(u)
	return err == nil
}

func UUIDString(myUUID pgtype.UUID) string {
	return fmt.Sprintf("%x-%x-%x-%x-%x", myUUID.Bytes[0:4], myUUID.Bytes[4:6], myUUID.Bytes[6:8], myUUID.Bytes[8:10], myUUID.Bytes[10:16])
}
