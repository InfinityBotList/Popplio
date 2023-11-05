package types

import (
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/infinitybotlist/eureka/dovewing/dovetypes"
	"github.com/jackc/pgx/v5/pgtype"
)

type Ticket struct {
	ID            string                  `db:"id" json:"id"`
	ChannelID     string                  `db:"channel_id" json:"channel_id"`
	TopicID       string                  `db:"topic_id" json:"topic_id"`
	Issue         string                  `db:"issue" json:"issue"`
	TicketContext map[string]string       `db:"ticket_context" json:"ticket_context"`
	Messages      []Message               `db:"messages" json:"messages"`
	UserID        string                  `db:"user_id" json:"-"`
	Author        *dovetypes.PlatformUser `db:"-" json:"author"`
	CloseUserID   pgtype.Text             `db:"close_user_id" json:"-"`
	CloseUser     *dovetypes.PlatformUser `db:"-" json:"close_user"`
	Open          bool                    `db:"open" json:"open"`
	CreatedAt     time.Time               `db:"created_at" json:"created_at"`
	EncKey        pgtype.Text             `db:"enc_key" json:"enc_key"`
}

type Message struct {
	ID          string                    `json:"id"`
	Timestamp   time.Time                 `json:"timestamp"` // Not in DB, but generated from snowflake ID
	Content     string                    `json:"content"`
	Embeds      []*discordgo.MessageEmbed `json:"embeds"`
	AuthorID    string                    `json:"author_id"`
	Author      *dovetypes.PlatformUser   `json:"author"`
	Attachments []Attachment              `json:"attachments"`
}

type Attachment struct {
	ID          string   `json:"id"`           // ID of the attachment within the ticket
	URL         string   `json:"url"`          // URL of the attachment
	ProxyURL    string   `json:"proxy_url"`    // URL (cached) of the attachment
	Name        string   `json:"name"`         // Name of the attachment
	ContentType string   `json:"content_type"` // Content type of the attachment
	Size        int      `json:"size"`         // Size of the attachment in bytes
	Errors      []string `json:"errors"`       // Non-fatal errors that occurred while uploading the attachment
}
