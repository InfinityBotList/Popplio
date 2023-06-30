package types

import (
	"time"

	"github.com/infinitybotlist/eureka/dovewing/dovetypes"
	"github.com/jackc/pgx/v5/pgtype"
)

// A review is a review on ibl
type Review struct {
	ID         pgtype.UUID             `db:"id" json:"id"`
	TargetType string                  `db:"target_type" json:"target_type"`
	TargetID   string                  `db:"target_id" json:"target_id"`
	AuthorID   string                  `db:"author" json:"-"`
	Author     *dovetypes.PlatformUser `db:"-" json:"author"`
	Content    string                  `db:"content" json:"content"`
	Stars      int32                   `db:"stars" json:"stars"`
	CreatedAt  time.Time               `db:"created_at" json:"created_at"`
	ParentID   pgtype.UUID             `db:"parent_id" json:"parent_id"`
}

type ReviewList struct {
	Reviews []Review `json:"reviews"`
}
