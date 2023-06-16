package types

import (
	"time"

	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/jackc/pgx/v5/pgtype"
)

// An announcement
type Announcement struct {
	UserID       string                 `db:"author" json:"-"`
	Author       *dovewing.PlatformUser `json:"author"` // Must be parsed internally
	ID           pgtype.UUID            `db:"id" json:"id"`
	Title        string                 `db:"title" json:"title"`
	Content      string                 `db:"content" json:"content"`
	LastModified time.Time              `db:"modified_date" json:"last_modified"`
	Status       string                 `db:"status" json:"status"`
	Target       pgtype.Text            `db:"target" json:"target"`
}

type AnnouncementList struct {
	Announcements []Announcement `json:"announcements"`
}
