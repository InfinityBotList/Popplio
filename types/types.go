package types

import (
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v5/pgtype"
)

// A link is any extra link
type Link struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Interval struct {
	Duration time.Duration `json:"duration"`
	String   string        `json:"string"`
}

func NewInterval(d time.Duration) Interval {
	return Interval{
		Duration: d,
		String:   d.String(),
	}
}

// SEO object (minified bot/user/server for seo purposes)
type SEO struct {
	Username string `json:"username"`
	ID       string `json:"id"`
	Avatar   string `json:"avatar"`
	Short    string `json:"short"`
}

// This represents a IBL Popplio API Error
type ApiError struct {
	Context any    `json:"context,omitempty"`
	Message string `json:"message"`
	Error   bool   `json:"error"`
}

// A discord user
type DiscordUser struct {
	ID             string           `json:"id"`
	Username       string           `json:"username"`
	Discriminator  string           `json:"discriminator"`
	Avatar         string           `json:"avatar"`
	Bot            bool             `json:"bot"`
	Mention        string           `json:"mention"`
	Status         discordgo.Status `json:"status"`
	System         bool             `json:"system"`
	Nickname       string           `json:"nickname"`
	Guild          string           `json:"in_guild"`
	Flags          int              `json:"flags"`
	Tag            string           `json:"tag"`
	IsServerMember bool             `json:"is_member"`
}

type ResolvedReminderBot struct {
	Name   string `db:"-" json:"name"`
	Avatar string `db:"-" json:"avatar"`
}

type Reminder struct {
	UserID      string              `db:"user_id" json:"user_id"`
	BotID       string              `db:"bot_id" json:"bot_id"`
	ResolvedBot ResolvedReminderBot `db:"-" json:"resolved"`
	CreatedAt   time.Time           `db:"created_at" json:"created_at"`
	LastAcked   time.Time           `db:"last_acked" json:"last_acked"`
}

type AuthUser struct {
	Token  string `json:"token"`
	UserID string `json:"user_id"`
}

type Message struct {
	ID          string                         `json:"id"`
	Timestamp   time.Time                      `json:"timestamp"` // Not in DB, but generated from snowflake ID
	Content     string                         `json:"content"`
	Embeds      []*discordgo.MessageEmbed      `json:"embeds"`
	AuthorID    string                         `json:"author_id"`
	Author      *DiscordUser                   `json:"author"`
	Attachments []*discordgo.MessageAttachment `json:"attachments"`
}

type Ticket struct {
	ID            string            `db:"id" json:"id"`
	ChannelID     string            `db:"channel_id" json:"channel_id"`
	TopicID       string            `db:"topic_id" json:"topic_id"`
	Issue         string            `db:"issue" json:"issue"`
	TicketContext map[string]string `db:"ticket_context" json:"ticket_context"`
	Messages      []Message         `db:"messages" json:"messages"`
	UserID        string            `db:"user_id" json:"-"`
	Author        *DiscordUser      `db:"-" json:"author"`
	CloseUserID   pgtype.Text       `db:"close_user_id" json:"-"`
	CloseUser     *DiscordUser      `db:"-" json:"close_user"`
	Open          bool              `db:"open" json:"open"`
	CreatedAt     time.Time         `db:"created_at" json:"created_at"`
}

type UserSubscription struct {
	Auth     string `json:"auth"`
	P256dh   string `json:"p256dh"`
	Endpoint string `json:"endpoint"`
}

type ReminderList struct {
	Reminders []Reminder `json:"reminders"`
}

type NotificationInfo struct {
	PublicKey string `json:"public_key"`
}

type TargetType int

const (
	TargetTypeUser TargetType = iota
	TargetTypeBot
	TargetTypeServer
)

// Notification
type NotifGet struct {
	Endpoint    string           `json:"endpoint"`
	NotifID     string           `json:"notif_id"`
	CreatedAt   time.Time        `json:"created_at"`
	BrowserInfo NotifBrowserInfo `json:"browser_info"`
}

type NotifBrowserInfo struct {
	// The OS of the browser
	OS         string
	Browser    string
	BrowserVer string
	Mobile     bool
}

type NotifGetList struct {
	Notifications []NotifGet `json:"notifications"`
}

// List Index
type ListIndex struct {
	Certified     []IndexBot     `json:"certified"`
	MostViewed    []IndexBot     `json:"most_viewed"`
	Packs         []IndexBotPack `json:"packs"`
	RecentlyAdded []IndexBot     `json:"recently_added"`
	TopVoted      []IndexBot     `json:"top_voted"`
}

// List Stats
type ListStatsBot struct {
	BotID              string   `json:"bot_id"`
	Vanity             string   `json:"vanity"`
	Short              string   `json:"short"`
	Type               string   `json:"type"`
	MainOwnerID        string   `json:"main_owner_id"`
	AdditionalOwnerIDS []string `json:"additional_owners_ids"`
	QueueName          string   `json:"queue_name"`
}

type ListStats struct {
	Bots         []ListStatsBot `json:"bots"`
	TotalStaff   int64          `json:"total_staff"`
	TotalUsers   int64          `json:"total_users"`
	TotalVotes   int64          `json:"total_votes"`
	TotalPacks   int64          `json:"total_packs"`
	TotalTickets int64          `json:"total_tickets"`
}

// OauthInfo struct, internally used
type OauthUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Disc     string `json:"discriminator"`
}

type TestAuth struct {
	AuthType TargetType `json:"auth_type"`
	TargetID string     `json:"target_id"`
	Token    string     `json:"token"`
}
