package types

import (
	"time"
)

// A link is any extra link
type Link struct {
	Name  string `json:"name" description:"Name of the link. Links starting with an underscore are 'asset links' and are not visible"`
	Value string `json:"value" description:"Value of the link. Must normally be HTTPS with the exception of 'asset links'"`
}

type Interval struct {
	Duration time.Duration `json:"duration" description:"Duration of the interval (in ns)"`
	String   string        `json:"string" description:"String representation of the interval"`
	Seconds  int           `json:"secs" description:"Duration of the interval (in secs)"`
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
	Username string `json:"username" description:"Username of the entity"`
	ID       string `json:"id" description:"ID of the entity"`
	Avatar   string `json:"avatar" description:"The entities resolved avatar URL (not just hash)"`
	Short    string `json:"short" description:"Short description of the entity"`
}

// This represents a IBL Popplio API Error
type ApiError struct {
	Context map[string]string `json:"context,omitempty" description:"Context of the error. Usually used for validation error contexts"`
	Message string            `json:"message" description:"Message of the error"`
	Error   bool              `json:"error" description:"Whether or not this is an error"`
}

// OauthInfo struct for oauth2 info
type OauthUser struct {
	ID       string `json:"id" description:"The user's ID"`
	Username string `json:"username" description:"The user's username"`
	Disc     string `json:"discriminator" description:"The user's discriminator"`
}
