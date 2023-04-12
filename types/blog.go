package types

import (
	"time"

	"github.com/infinitybotlist/eureka/dovewing"
)

type BlogPost struct {
	Slug        string                `db:"slug" json:"slug" description:"The slug/vanity of the blog post"`
	Title       string                `db:"title" json:"title" description:"The title of the blog post"`
	Description string                `db:"description" json:"description" description:"The summary/short description of the blog post"`
	UserID      string                `db:"user_id" json:"-"` // Must be parsed internally
	Author      *dovewing.DiscordUser `db:"-" json:"author" description:"The author of the blog post"`
	CreatedAt   time.Time             `db:"created_at" json:"created_at" description:"The time the blog post was created at"`
	Content     string                `db:"content" json:"content" description:"The content of the blog post in markdown format"`
	Draft       bool                  `db:"draft" json:"draft" description:"Whether the blog post is a draft or not (hidden or public)"`
	Tags        []string              `db:"tags" json:"tags" description:"The tags of the blog post for filtering purposes"`
}

type BlogListPost struct {
	Slug        string                `db:"slug" json:"slug" description:"The slug/vanity of the blog post"`
	Title       string                `db:"title" json:"title" description:"The title of the blog post"`
	Description string                `db:"description" json:"description" description:"The summary/short description of the blog post"`
	UserID      string                `db:"user_id" json:"-"` // Must be parsed internally
	Author      *dovewing.DiscordUser `db:"-" json:"author" description:"The author of the blog post"`
	CreatedAt   time.Time             `db:"created_at" json:"created_at" description:"The time the blog post was created at"`
	Draft       bool                  `db:"draft" json:"draft" description:"Whether the blog post is a draft or not (hidden or public)"`
	Tags        []string              `db:"tags" json:"tags" description:"The tags of the blog post for filtering purposes"`
}

type Blog struct {
	Posts []BlogListPost `json:"posts" description:"The list of blog posts on the blog"`
}

type PublishBlogPost struct {
	Draft bool `db:"draft" json:"draft" description:"Whether or not the blog post is a draft or not."`
}

type CreateBlogPost struct {
	Slug        string   `db:"slug" json:"slug" validate:"required" description:"Slug must not contain spaces and should be alphanumeric where possible."`
	Title       string   `db:"title" json:"title" validate:"required" description:"The title of the blog post"`
	Description string   `db:"description" json:"description" validate:"required" description:"The summary/short description of the blog post"`
	Content     string   `db:"content" json:"content" validate:"required" description:"The content of the blog post in markdown format"`
	Tags        []string `db:"tags" json:"tags" validate:"required,dive,required" description:"The tags of the blog post for filtering purposes"`
}

type EditBlogPost struct {
	Title       string   `db:"title" json:"title" validate:"required" description:"The title of the blog post"`
	Description string   `db:"description" json:"description" validate:"required" description:"The summary/short description of the blog post"`
	Content     string   `db:"content" json:"content" validate:"required" description:"The content of the blog post in markdown format"`
	Tags        []string `db:"tags" json:"tags" validate:"required,dive,required" description:"The tags of the blog post for filtering purposes"`
}
