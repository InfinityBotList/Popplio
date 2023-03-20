package utils

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"popplio/state"
	"popplio/teams"
	"popplio/types"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/google/uuid"
	"github.com/infinitybotlist/dovewing"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

var (
	userBotColsArr = GetCols(types.UserBot{})
	userBotCols    = strings.Join(userBotColsArr, ",")
)

// Returns if a string is empty/null or not. Used throughout the codebase
func IsNone(s string) bool {
	if s == "None" || s == "none" || s == "" || s == "null" {
		return true
	}
	return false
}

// Returns the votes of a pack, Used throughout the codebase
func ResolvePackVotes(ctx context.Context, url string) ([]types.PackVote, error) {
	rows, err := state.Pool.Query(ctx, "SELECT user_id, upvote, created_at FROM pack_votes WHERE url = $1", url)

	if err != nil {
		return []types.PackVote{}, err
	}

	defer rows.Close()

	votes := []types.PackVote{}

	for rows.Next() {
		// Fetch votes for the pack
		var userId string
		var upvote bool
		var createdAt time.Time

		err := rows.Scan(&userId, &upvote, &createdAt)

		if err != nil {
			return nil, err
		}

		votes = append(votes, types.PackVote{
			UserID:    userId,
			Upvote:    upvote,
			CreatedAt: createdAt,
		})
	}

	return votes, nil
}

func ResolveTeamBots(ctx context.Context, teamId string) ([]types.UserBot, error) {
	// Gets the bots of the team so we can add it to UserBots
	var teamBotIds []string
	var bots = []types.UserBot{}

	teamBotRows, err := state.Pool.Query(ctx, "SELECT bot_id FROM bots WHERE team_owner = $1", teamId)

	if err != nil {
		return nil, err
	}

	err = pgxscan.ScanAll(&teamBotIds, teamBotRows)

	if err != nil {
		return nil, err
	}

	for _, botId := range teamBotIds {
		userBotsRows, err := state.Pool.Query(ctx, "SELECT "+userBotCols+" FROM bots WHERE bot_id = $1", botId)

		if err != nil {
			return nil, err
		}

		var userBot = types.UserBot{}

		err = pgxscan.ScanOne(&userBot, userBotsRows)

		if err != nil {
			return nil, err
		}

		userObj, err := dovewing.GetDiscordUser(ctx, userBot.BotID)

		if err != nil {
			state.Logger.Error(err)
			continue
		}

		userBot.User = userObj

		bots = append(bots, userBot)
	}

	return bots, nil
}

func GetDoubleVote() bool {
	return time.Now().Weekday() == time.Friday || time.Now().Weekday() == time.Saturday || time.Now().Weekday() == time.Sunday
}

func GetVoteTime() uint16 {
	if GetDoubleVote() {
		return 6
	} else {
		return 12
	}
}

func GetVoteData(ctx context.Context, userID, botID string, log bool) (*types.UserVote, error) {
	var premium bool
	err := state.Pool.QueryRow(ctx, "SELECT premium FROM bots WHERE bot_id = $1", botID).Scan(&premium)

	if err != nil {
		return nil, err
	}

	var votes []int64

	var voteDates []*struct {
		Date pgtype.Timestamptz `db:"created_at"`
	}

	rows, err := state.Pool.Query(ctx, "SELECT created_at FROM votes WHERE user_id = $1 AND bot_id = $2 ORDER BY created_at DESC", userID, botID)

	if err != nil {
		return nil, err
	}

	err = pgxscan.ScanAll(&voteDates, rows)

	for _, vote := range voteDates {
		if vote.Date.Valid {
			votes = append(votes, vote.Date.Time.UnixMilli())
		}
	}

	voteParsed := types.UserVote{
		UserID: userID,
		VoteInfo: types.VoteInfo{
			Weekend:  GetDoubleVote(),
			VoteTime: GetVoteTime(),
		},
		PremiumBot: premium,
	}

	if premium {
		voteParsed.VoteInfo.VoteTime = 4
	}

	if log {
		state.Logger.With(
			zap.String("user_id", userID),
			zap.String("bot_id", botID),
			zap.Int64s("votes", votes),
			zap.Error(err),
		).Info("Got vote data")
	}

	voteParsed.Timestamps = votes

	// In most cases, will be one but not always
	if len(votes) > 0 {
		if time.Now().UnixMilli() < votes[0] {
			state.Logger.Error("detected illegal vote time", votes[0])
			votes[0] = time.Now().UnixMilli()
		}

		if time.Now().UnixMilli()-votes[0] < int64(voteParsed.VoteInfo.VoteTime)*60*60*1000 {
			voteParsed.HasVoted = true
			voteParsed.LastVoteTime = votes[0]
		}
	}

	if voteParsed.LastVoteTime == 0 && len(votes) > 0 {
		voteParsed.LastVoteTime = votes[0]
	}
	return &voteParsed, nil
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
		var teamPerms []teams.TeamPermission

		err = state.Pool.QueryRow(ctx, "SELECT perms FROM team_members WHERE team_id = $1 AND user_id = $2", teamOwner, userID).Scan(&teamPerms)

		if err != nil {
			return &teams.PermissionManager{}, fmt.Errorf("error finding team member: %v", err)
		}

		return teams.NewPermissionManager(teamPerms), nil
	}

	if owner.String == userID {
		return teams.NewPermissionManager([]teams.TeamPermission{teams.TeamPermissionOwner}), nil
	}

	return teams.NewPermissionManager([]teams.TeamPermission{}), nil
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
	state.Redis.Del(ctx, "bc-"+vanity)
	state.Redis.Del(ctx, "bc-"+botId)
	state.Redis.Del(ctx, "bc-"+clientId)

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

func retArrInt[T comparable](t []T) []string {
	var arr []string
	for _, v := range t {
		arr = append(arr, fmt.Sprint(v))
	}
	return arr
}

// Casts a array of type any to []string
func ArrayCast(v any) []string {
	switch t := v.(type) {
	// String type
	case []string:
		return t
	// Any type
	case []any:
		var arr []string
		for _, v := range t {
			arr = append(arr, v.(string))
		}
		return arr
	// All the int types
	case []int:
		return retArrInt(t)
	case []int8:
		return retArrInt(t)
	case []int16:
		return retArrInt(t)
	case []int32:
		return retArrInt(t)
	case []int64:
		return retArrInt(t)
	// All the uint types
	case []uint:
		return retArrInt(t)
	case []uint8:
		return retArrInt(t)
	case []uint16:
		return retArrInt(t)
	case []uint32:
		return retArrInt(t)
	case []uint64:
		return retArrInt(t)
	// All the float types
	case []float32:
		return retArrInt(t)
	case []float64:
		return retArrInt(t)
	}
	return []string{}
}
