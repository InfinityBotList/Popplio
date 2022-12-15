package assets

import "time"

type Action struct {
	Action string    `json:"action"`
	Ctx    string    `json:"ctx"` // For extra context
	TID    string    `json:"tid"` // In the case of rtb/bwebsec, this is the ID to target
	Time   time.Time `json:"time"`
}

type Redirect struct {
	Redirect string `json:"redirect"`
}
