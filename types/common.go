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
