package types

import (
	"encoding/json"
	"fmt"
)

// PlatformUser is a dovewing-resolved Discord user. Equality is by id only.
type PlatformUser struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	Avatar      string `json:"avatar"`
	DisplayName string `json:"display_name"`
	Bot         bool   `json:"bot"`
	Status      string `json:"status"`
}

func (p PlatformUser) Equals(other PlatformUser) bool {
	return p.ID == other.ID
}

type PartialBot struct {
	BotID        string       `json:"bot_id"`
	User         PlatformUser `json:"user"`
	Short        string       `json:"short"`
	Type         string       `json:"type"`
	Votes        int32        `json:"votes"`
	Shards       int32        `json:"shards"`
	Library      string       `json:"library"`
	InviteClicks int32        `json:"invite_clicks"`
	Clicks       int32        `json:"clicks"`
	Servers      int32        `json:"servers"`
	ClaimedBy    *string      `json:"claimed_by"`
	LastClaimed  *Timestamp   `json:"last_claimed"`
	ApprovalNote string       `json:"approval_note"`
	Mentionable  []string     `json:"mentionable"`
	Invite       string       `json:"invite"`
	ClientID     string       `json:"client_id"`
}

type PartialServer struct {
	ServerID      string     `json:"server_id"`
	Name          string     `json:"name"`
	Avatar        string     `json:"avatar"`
	TotalMembers  int32      `json:"total_members"`
	OnlineMembers int32      `json:"online_members"`
	Short         string     `json:"short"`
	Type          string     `json:"type"`
	Votes         int32      `json:"votes"`
	InviteClicks  int32      `json:"invite_clicks"`
	Clicks        int32      `json:"clicks"`
	NSFW          bool       `json:"nsfw"`
	Tags          []string   `json:"tags"`
	Premium       bool       `json:"premium"`
	ClaimedBy     *string    `json:"claimed_by"`
	LastClaimed   *Timestamp `json:"last_claimed"`
	Mentionable   []string   `json:"mentionable"`
}

// PartialEntity is a newtype union: {"Bot":{..}} / {"Server":{..}}.
type PartialEntity struct {
	Bot    *PartialBot
	Server *PartialServer
}

func (e *PartialEntity) UnmarshalJSON(data []byte) error {
	*e = PartialEntity{}

	return unmarshalUnion("PartialEntity", data, map[string]func() any{
		"Bot": func() any {
			e.Bot = &PartialBot{}
			return e.Bot
		},
		"Server": func() any {
			e.Server = &PartialServer{}
			return e.Server
		},
	})
}

func (e PartialEntity) MarshalJSON() ([]byte, error) {
	switch {
	case e.Bot != nil:
		return encodeVariant("Bot", e.Bot)
	case e.Server != nil:
		return encodeVariant("Server", e.Server)
	default:
		return nil, fmt.Errorf("PartialEntity: no variant set")
	}
}

// RPCLogEntry is one row of the rpc_logs table as the panel sees it. `data` is
// raw JSON passthrough.
type RPCLogEntry struct {
	ID        string          `json:"id"`
	UserID    string          `json:"user_id"`
	Method    string          `json:"method"`
	State     string          `json:"state"`
	Data      json.RawMessage `json:"data"`
	CreatedAt Timestamp       `json:"created_at"`
}

// BaseAnalytics is the BaseAnalytics response. changelogs_count is hardcoded 0
// upstream.
type BaseAnalytics struct {
	BotCounts       map[string]int64 `json:"bot_counts"`
	ServerCounts    map[string]int64 `json:"server_counts"`
	TicketCounts    map[string]int64 `json:"ticket_counts"`
	TotalUsers      int64            `json:"total_users"`
	ChangelogsCount int64            `json:"changelogs_count"`
}
