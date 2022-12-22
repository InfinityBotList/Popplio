package types

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// An announcement
type Announcement struct {
	ITag         pgtype.UUID `db:"itag" json:"itag"`
	Author       string      `db:"user_id" json:"author"`
	ID           string      `db:"id" json:"id"`
	Title        string      `db:"title" json:"title"`
	Content      string      `db:"content" json:"content"`
	LastModified time.Time   `db:"modified_date" json:"last_modified"`
	Status       string      `db:"status" json:"status"`
	Targetted    bool        `db:"targetted" json:"targetted"`
	Target       pgtype.Text `db:"target" json:"target"`
}

type AnnouncementList struct {
	Announcements []Announcement `json:"announcements"`
}
