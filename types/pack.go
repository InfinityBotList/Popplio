package types

import (
	"time"

	"github.com/infinitybotlist/eureka/dovewing/dovetypes"
	"github.com/jackc/pgx/v5/pgtype"
)

// Represents a Bot Pack
type BotPack struct {
	Owner         string                  `db:"owner" json:"-" ci:"internal"`
	ResolvedOwner *dovetypes.PlatformUser `db:"-" json:"owner"`
	Name          string                  `db:"name" json:"name"`
	Short         string                  `db:"short" json:"short"`
	Votes         int                     `db:"votes" json:"votes" description:"The pack's vote count"`
	Tags          []string                `db:"tags" json:"tags"`
	URL           string                  `db:"url" json:"url"`
	CreatedAt     time.Time               `db:"created_at" json:"created_at"`
	Bots          []string                `db:"bots" json:"bot_ids"`
	ResolvedBots  []ResolvedPackBot       `db:"-" json:"bots"`
}

type ResolvedPackBot struct {
	User         *dovetypes.PlatformUser `json:"user"`
	Short        string                  `json:"short"`
	Type         pgtype.Text             `json:"type"`
	Vanity       string                  `json:"vanity"`
	Banner       pgtype.Text             `json:"banner"`
	NSFW         bool                    `json:"nsfw"`
	Premium      bool                    `json:"premium"`
	Shards       int                     `json:"shards"`
	Votes        int                     `json:"votes"`
	InviteClicks int                     `json:"invite_clicks"`
	Servers      int                     `json:"servers"`
	Tags         []string                `json:"tags"`
}

type IndexBotPack struct {
	Owner     string    `db:"owner" json:"owner_id"`
	Name      string    `db:"name" json:"name"`
	Short     string    `db:"short" json:"short"`
	Votes     int       `db:"votes" json:"votes" description:"The pack's vote count"`
	Tags      []string  `db:"tags" json:"tags"`
	URL       string    `db:"url" json:"url"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	Bots      []string  `db:"bots" json:"bot_ids"`
}
