package types

import (
	"time"

	"github.com/infinitybotlist/eureka/dovewing/dovetypes"
	"github.com/jackc/pgx/v5/pgtype"
)

// A review is a review on ibl
type Review struct {
	ID          pgtype.UUID             `db:"id" json:"id" description:"The review ID"`
	TargetType  string                  `db:"target_type" json:"target_type" description:"The target type (bot/server) the review is for"`
	TargetID    string                  `db:"target_id" json:"target_id" description:"The target ID the review is for"`
	AuthorID    string                  `db:"author" json:"-" description:"The author ID of the review"`
	Author      *dovetypes.PlatformUser `db:"-" json:"author" description:"The author of the review"`
	OwnerReview bool                    `db:"owner_review" json:"owner_review" description:"Whether or not the review is an owner review"`
	Content     string                  `db:"content" json:"content"`
	Stars       int32                   `db:"stars" json:"stars"`
	CreatedAt   time.Time               `db:"created_at" json:"created_at"`
	ParentID    pgtype.UUID             `db:"parent_id" json:"parent_id"`
}

type CreateReview struct {
	Content     string `db:"content" json:"content" validate:"required,min=5,max=4000" msg:"Content must be between 5 and 4000 characters"`
	Stars       int32  `db:"stars" json:"stars" validate:"required,min=1,max=5" msg:"Stars must be between 1 and 5 stars"`
	ParentID    string `db:"parent_id" json:"parent_id" validate:"omitempty,uuid" msg:"Parent ID must be a valid UUID if provided"`
	OwnerReview bool   `db:"owner_review" json:"owner_review" description:"Whether or not the review is an owner review"`
}

type EditReview struct {
	Content string `db:"content" json:"content" validate:"required,min=5,max=4000" msg:"Content must be between 5 and 4000 characters"`
	Stars   int32  `db:"stars" json:"stars" validate:"required,min=1,max=5" msg:"Stars must be between 1 and 5 stars"`
}

type ReviewList struct {
	Reviews []Review `json:"reviews"`
}
