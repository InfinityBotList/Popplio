package types

import (
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/infinitybotlist/eureka/dovewing/dovetypes"
)

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
	ID          string   `json:"id"`                  // ID of the attachment within the ticket
	URL         string   `json:"url,omitempty"`       // URL of the attachment
	ProxyURL    string   `json:"proxy_url,omitempty"` // URL (cached) of the attachment
	Filename    string   `json:"filename,omitempty"`  // Name of the file attached (temporary used for migration)
	Name        string   `json:"name"`                // Name of the attachment
	ContentType string   `json:"content_type"`        // Content type of the attachment
	Size        int      `json:"size"`                // Size of the attachment in bytes
	Errors      []string `json:"errors"`              // Non-fatal errors that occurred while uploading the attachment
}

type TableMigrationType struct {
	ID       string     `db:"id" json:"id"`
	Messages []*Message `db:"messages" json:"messages"`
}
