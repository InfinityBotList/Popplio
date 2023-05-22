package types

import (
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/jackc/pgx/v5/pgtype"
)

type Ticket struct {
	ID            string                `db:"id" json:"id"`
	ChannelID     string                `db:"channel_id" json:"channel_id"`
	TopicID       string                `db:"topic_id" json:"topic_id"`
	Issue         string                `db:"issue" json:"issue"`
	TicketContext map[string]string     `db:"ticket_context" json:"ticket_context"`
	Messages      []Message             `db:"messages" json:"messages"`
	UserID        string                `db:"user_id" json:"-"`
	Author        *dovewing.DiscordUser `db:"-" json:"author"`
	CloseUserID   pgtype.Text           `db:"close_user_id" json:"-"`
	CloseUser     *dovewing.DiscordUser `db:"-" json:"close_user"`
	Open          bool                  `db:"open" json:"open"`
	CreatedAt     time.Time             `db:"created_at" json:"created_at"`
}

type Message struct {
	ID          string                         `json:"id"`
	Timestamp   time.Time                      `json:"timestamp"` // Not in DB, but generated from snowflake ID
	Content     string                         `json:"content"`
	Embeds      []*discordgo.MessageEmbed      `json:"embeds"`
	AuthorID    string                         `json:"author_id"`
	Author      *dovewing.DiscordUser          `json:"author"`
	Attachments []*discordgo.MessageAttachment `json:"attachments"`
}
