package assets

import "time"

type Action struct {
	Action string    `json:"a"`
	Nonce  string    `json:"n"`
	Ctx    string    `json:"c"` // For extra context
	TID    string    `json:"t"` // In the case of rtb/bwebsec, this is the ID to target
	Time   time.Time `json:"ts"`
}

type Redirect struct {
	Redirect string `json:"redirect"`
}

type ConfirmTemplate struct {
	Action Action `json:"action"`
}
