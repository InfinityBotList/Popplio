package types

import (
	"time"

	"github.com/bwmarrin/discordgo"
)

// A link is any extra link
type Link struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Interval struct {
	Duration time.Duration `json:"duration"`
	String   string        `json:"string"`
	Seconds  int           `json:"secs"`
}

func NewInterval(d time.Duration) Interval {
	return Interval{
		Duration: d,
		String:   d.String(),
		Seconds:  int(d.Seconds()),
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
	Route          *CacheRoute      `json:"route"`
}

type CacheRoute struct {
	State bool `json:"state"`
	Redis bool `json:"redis"`
	DB    bool `json:"db"`
}

type AuthUser struct {
	Token  string `json:"token"`
	UserID string `json:"user_id"`
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
	Premium       []IndexBot     `json:"premium"`
	MostViewed    []IndexBot     `json:"most_viewed"`
	Packs         []IndexBotPack `json:"packs"`
	RecentlyAdded []IndexBot     `json:"recently_added"`
	TopVoted      []IndexBot     `json:"top_voted"`
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
