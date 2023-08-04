package utils

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"popplio/state"
	"popplio/types"

	"github.com/google/uuid"
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
