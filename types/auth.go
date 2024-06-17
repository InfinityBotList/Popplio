package types

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type AuthorizeRequest struct {
	ClientID    string `json:"client_id" validate:"required"`
	Code        string `json:"code" validate:"required,min=5"`
	RedirectURI string `json:"redirect_uri" validate:"required"`
	Protocol    string `json:"protocol" validate:"required" description:"Should be 'persepolis'. This is to identify and block older clients that don't support newer protocols"`
	Scope       string `json:"scope" validate:"required,oneof=normal ban_exempt external_auth"`
}

type UserLogin struct {
	Token  string `json:"token" description:"The users token"`
	UserID string `json:"user_id" description:"The users ID"`
}

type OauthMeta struct {
	ClientID string `json:"client_id" description:"The client ID"`
	URL      string `json:"url" description:"The URL to redirect the user to for discord oauth2"`
}

type TestAuth struct {
	AuthType string `json:"auth_type"`
	TargetID string `json:"target_id"`
	Token    string `json:"token"`
}

// @ci table=api_sessions ignore_fields=token
//
// Represents a session that can be used to authorize/identify a user
type Session struct {
	ID         string      `db:"id" json:"id" description:"The ID of the session"`
	Name       pgtype.Text `db:"name" json:"name,omitempty" description:"The name of the session. Login sessions do not have any names by default"`
	CreatedAt  time.Time   `db:"created_at" json:"created_at" description:"The time the session was created"`
	Type       string      `db:"type" json:"type" description:"The type of session token"`
	TargetType string      `db:"target_type" json:"target_type" description:"The target (entities) type"`
	TargetID   string      `db:"target_id" json:"target_id" description:"The target (entities) ID"`
	PermLimits []string    `db:"perm_limits" json:"perm_limits" description:"The permissions the session has"`
	Expiry     time.Time   `db:"expiry" json:"expiry" description:"The time the session expires"`
}

// A list of sessions.
type SessionList struct {
	Sessions []*Session `json:"sessions" description:"The list of sessions"`
}

type CreateSession struct {
	Name       string   `json:"name" validate:"required" description:"The name of the session"`
	Type       string   `json:"type" validate:"oneof=api" description:"The type of session token. Must be 'api'. Login sessions cannot be created using the Create Session API for obvious reasons"`
	PermLimits []string `json:"perm_limits" description:"The permissions the session will have"`
	Expiry     int64    `json:"expiry" validate:"required" description:"The time in seconds the session will last"`
}

type CreateSessionResponse struct {
	Token     string `json:"token" description:"The token of the session"`
	SessionID string `json:"session_id" description:"The ID of the session"`
}
