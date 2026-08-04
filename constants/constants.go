// Package constants holds the canned JSON bodies returned for generic
// failures.
//
// These are raw JSON strings rather than marshalled structs because uapi
// writes them straight to the response body without going through a
// marshaller. They are what a caller sees when a failure carries no specific
// message of its own — see popplio/api/resp for the helpers that return
// them.
package constants

const (
	ResourceNotFound    = "{\"message\":\"The requested resource could not be found.\"}"
	EndpointNotFound    = "{\"message\":\"This endpoint does not exist. Check the request path and try again.\"}"
	BadRequest          = "{\"message\":\"The request was malformed or invalid.\"}"
	Forbidden           = "{\"message\":\"You do not have permission to perform this action.\"}"
	Unauthorized        = "{\"message\":\"Authentication is required. Check that a valid API token was provided.\"}"
	InternalServerError = "{\"message\":\"An unexpected error occurred while processing the request. Please try again later.\"}"
	MethodNotAllowed    = "{\"message\":\"This HTTP method is not supported for this endpoint.\"}"
	BodyRequired        = "{\"message\":\"A request body is required for this endpoint.\"}"
	BackTick            = "`"
	DoubleBackTick      = "``"
)
