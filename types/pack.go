package types

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// Represents a Bot Pack
type BotPack struct {
	Owner         string            `db:"owner" json:"owner_id"`
	ResolvedOwner *DiscordUser      `db:"-" json:"owner"`
	Name          string            `db:"name" json:"name"`
	Short         string            `db:"short" json:"short"`
	Votes         []PackVote        `db:"-" json:"votes"`
	Tags          []string          `db:"tags" json:"tags"`
	URL           string            `db:"url" json:"url"`
	CreatedAt     time.Time         `db:"created_at" json:"created_at"`
	Bots          []string          `db:"bots" json:"bot_ids"`
	ResolvedBots  []ResolvedPackBot `db:"-" json:"bots"`
}

type ResolvedPackBot struct {
	User         *DiscordUser `json:"user"`
	Short        string       `json:"short"`
	Type         pgtype.Text  `json:"type"`
	Vanity       pgtype.Text  `json:"vanity"`
	Banner       pgtype.Text  `json:"banner"`
	NSFW         bool         `json:"nsfw"`
	Premium      bool         `json:"premium"`
	Shards       int          `json:"shards"`
	Votes        int          `json:"votes"`
	InviteClicks int          `json:"invite_clicks"`
	Servers      int          `json:"servers"`
	Tags         []string     `json:"tags"`
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

// All packs
type AllPacks struct {
	Count    uint64         `json:"count"`
	PerPage  uint64         `json:"per_page"`
	Next     string         `json:"next"`
	Previous string         `json:"previous"`
	Results  []IndexBotPack `json:"packs"`
}