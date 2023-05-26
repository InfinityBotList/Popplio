package types

import (
	"time"

	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/jackc/pgx/v5/pgtype"
)

// Represents a Bot Pack
type BotPack struct {
	Owner         string                `db:"owner" json:"-" ci:"internal"`
	ResolvedOwner *dovewing.DiscordUser `db:"-" json:"owner"`
	Name          string                `db:"name" json:"name"`
	Short         string                `db:"short" json:"short"`
	Votes         []PackVote            `db:"-" json:"votes"`
	Tags          []string              `db:"tags" json:"tags"`
	URL           string                `db:"url" json:"url"`
	CreatedAt     time.Time             `db:"created_at" json:"created_at"`
	Bots          []string              `db:"bots" json:"bot_ids"`
	ResolvedBots  []ResolvedPackBot     `db:"-" json:"bots"`
}

type ResolvedPackBot struct {
	User         *dovewing.DiscordUser `json:"user"`
	Short        string                `json:"short"`
	Type         pgtype.Text           `json:"type"`
	Vanity       pgtype.Text           `json:"vanity"`
	Banner       pgtype.Text           `json:"banner"`
	NSFW         bool                  `json:"nsfw"`
	Premium      bool                  `json:"premium"`
	Shards       int                   `json:"shards"`
	Votes        int                   `json:"votes"`
	InviteClicks int                   `json:"invite_clicks"`
	Servers      int                   `json:"servers"`
	Tags         []string              `json:"tags"`
}

type IndexBotPack struct {
	Owner     string     `db:"owner" json:"owner_id"`
	Name      string     `db:"name" json:"name"`
	Short     string     `db:"short" json:"short"`
	Votes     []PackVote `db:"-" json:"votes"`
	Tags      []string   `db:"tags" json:"tags"`
	URL       string     `db:"url" json:"url"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	Bots      []string   `db:"bots" json:"bot_ids"`
}

// Pack vote
type PackVote struct {
	UserID    string    `json:"user_id"`
	Upvote    bool      `json:"upvote"`
	CreatedAt time.Time `json:"created_at"`
}
