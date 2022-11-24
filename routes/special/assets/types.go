package assets

import "time"

type InternalOauthUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Disc     string `json:"discriminator"`
	TID      string `json:"-"` // Only set in taskFn
}

type Action struct {
	Action string
	Ctx    string
	TID    int64 // In the case of gettoken, this is the bot ID to reset the token of
	Time   time.Time
}
