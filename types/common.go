// Package types defines the request and response bodies of the Popplio API.
//
// These types are the API's contract in both directions: the `json` tags
// shape what callers send and receive, the `db` tags let popplio/db build
// the queries that populate them (see db.GetCols), and the `description`
// tags become the published API documentation. A field's tags are therefore
// part of the public interface, and tygo generates the frontend's TypeScript
// bindings from them.
package types

// A link is any extra link
type Link struct {
	Name  string `json:"name" description:"Name of the link. Links starting with an underscore are 'asset links' and are not visible"`
	Value string `json:"value" description:"Value of the link. Must normally be HTTPS with the exception of 'asset links'"`
}

// A snapshot of one of a server's custom emojis, synced periodically by the
// tracking bot from the guild it's actually a member of (see Infernoplex's
// serversync task) — not fetched live per page view.
type Emoji struct {
	ID       string `json:"id" description:"The emoji's Discord ID"`
	Name     string `json:"name" description:"The emoji's name"`
	Animated bool   `json:"animated" description:"Whether the emoji is animated"`
	URL      string `json:"url" description:"The emoji's CDN URL"`
}

// A snapshot of one of a server's stickers, synced the same way as Emoji.
type Sticker struct {
	ID     string `json:"id" description:"The sticker's Discord ID"`
	Name   string `json:"name" description:"The sticker's name"`
	Format string `json:"format" description:"The sticker's format (png, apng, lottie or gif)"`
	URL    string `json:"url" description:"The sticker's CDN URL"`
}

// SEO object (minified bot/user/server for seo purposes)
type SEO struct {
	Name   string `json:"name" description:"Name of the entity"`
	ID     string `json:"id" description:"ID of the entity"`
	Avatar string `json:"avatar" description:"The entities resolved avatar URL (not just hash)"`
	Short  string `json:"short" description:"Short description of the entity"`
}

// This represents a IBL Popplio API Error
type ApiError struct {
	Context map[string]string `json:"context,omitempty" description:"Context of the error. Usually used for validation error contexts"`
	Message string            `json:"message" description:"Message of the error"`
}

// Paged result common
type PagedResult[T any] struct {
	Count   uint64 `json:"count"`
	PerPage uint64 `json:"per_page"`
	Results T      `json:"results"`
}

// List of items
type ItemList[T any] struct {
	Items []T `json:"items"`
}
