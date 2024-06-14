package types

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type Vanity struct {
	ITag       pgtype.UUID `db:"itag" json:"itag" description:"The vanities internal ID."`
	TargetID   string      `db:"target_id" json:"target_id" description:"The ID of the entity"`
	TargetType string      `db:"target_type" json:"target_type" description:"The type of the entity"`
	Code       string      `db:"code" json:"code" description:"The code of the vanity"`
	CreatedAt  time.Time   `db:"created_at" json:"created_at" description:"The time the vanity was created"`
}

type PatchVanity struct {
	Code string `json:"code" description:"The new vanity code to use" validate:"required"`
}
