package types

import (
	"time"

	"github.com/infinitybotlist/dovewing"
	"github.com/jackc/pgx/v5/pgtype"
)

// A review is a review on ibl
type Review struct {
	ID        pgtype.UUID           `db:"id" json:"id"`
	BotID     string                `db:"bot_id" json:"bot_id"`
	AuthorID  string                `db:"author" json:"-"`
	Author    *dovewing.DiscordUser `db:"-" json:"author"`
	Content   string                `db:"content" json:"content"`
	Stars     pgtype.Int4           `db:"stars" json:"stars"`
	CreatedAt time.Time             `db:"created_at" json:"created_at"`
	ParentID  pgtype.UUID           `db:"parent_id" json:"parent_id"`
}

type ReviewList struct {
	Reviews []Review `json:"reviews"`
}
