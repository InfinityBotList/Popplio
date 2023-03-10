package constants

const (
	NotFound         = "{\"message\":\"Slow down, bucko! We couldn't find this resource *anywhere*!\",\"error\":true}"
	NotFoundPage     = "{\"message\":\"Slow down, bucko! You got the path wrong or something but this endpoint doesn't exist!\",\"error\":true}"
	BadRequest       = "{\"message\":\"Slow down, bucko! You're doing something illegal!!!\",\"error\":true}"
	Forbidden        = "{\"message\":\"Slow down, bucko! You're not allowed to do this!\",\"error\":true}"
	Unauthorized     = "{\"message\":\"Slow down, bucko! You're not authorized to do this or did you forget a API token somewhere?\",\"error\":true}"
	InternalError    = "{\"message\":\"Slow down, bucko! Something went wrong on our end!\",\"error\":true}"
	MethodNotAllowed = "{\"message\":\"Slow down, bucko! That method is not allowed for this endpoint!!!\",\"error\":true}"
	BackTick         = "`"
	DoubleBackTick   = "``"
)
