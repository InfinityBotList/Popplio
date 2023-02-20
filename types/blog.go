package types

import "time"

type BlogPost struct {
	Slug        string       `db:"slug" json:"slug"`
	Title       string       `db:"title" json:"title"`
	Description string       `db:"description" json:"description"`
	UserID      string       `db:"user_id" json:"-"` // Must be parsed internally
	Author      *DiscordUser `db:"-" json:"author"`
	CreatedAt   time.Time    `db:"created_at" json:"created_at"`
	Content     string       `db:"content" json:"content"`
	Draft       bool         `db:"draft" json:"draft"`
	Tags        []string     `db:"tags" json:"tags"`
}

type BlogListPost struct {
	Slug        string       `db:"slug" json:"slug"`
	Title       string       `db:"title" json:"title"`
	Description string       `db:"description" json:"description"`
	UserID      string       `db:"user_id" json:"-"` // Must be parsed internally
	Author      *DiscordUser `db:"-" json:"author"`
	CreatedAt   time.Time    `db:"created_at" json:"created_at"`
	Draft       bool         `db:"draft" json:"draft"`
	Tags        []string     `db:"tags" json:"tags"`
}

type Blog struct {
	Posts []BlogListPost `json:"posts"`
}
