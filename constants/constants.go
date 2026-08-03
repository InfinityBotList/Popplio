// Package constants holds the canned JSON bodies returned for generic
// failures.
//
// These are raw JSON strings rather than marshalled structs because uapi
// writes them straight to the response body without going through a
// marshaller. They are what a caller sees when a failure carries no specific
// message of its own see popplio/api/resp for the helpers that return
// them.
package constants

const (
	ResourceNotFound    = "{\"message\":\"Slow down, bucko! We couldn't find this resource *anywhere*!\"}"
	EndpointNotFound    = "{\"message\":\"Slow down, bucko! You got the path wrong or something but this endpoint doesn't exist!\"}"
	BadRequest          = "{\"message\":\"Slow down, bucko! You're doing something illegal!!!\"}"
	Forbidden           = "{\"message\":\"Slow down, bucko! You're not allowed to do this!\"}"
	Unauthorized        = "{\"message\":\"Slow down, bucko! You're not authorized to do this or did you forget a API token somewhere?\"}"
	InternalServerError = "{\"message\":\"Slow down, bucko! Something went wrong on our end!\"}"
	MethodNotAllowed    = "{\"message\":\"Slow down, bucko! That method is not allowed for this endpoint!!!\"}"
	BodyRequired        = "{\"message\":\"Slow down, bucko! A body is required for this endpoint!!!\"}"
	BackTick            = "`"
	DoubleBackTick      = "``"
)
