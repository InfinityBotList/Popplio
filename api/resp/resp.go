// Package resp constructs the HTTP responses returned by Popplio's route
// handlers.
//
// Route handlers return a [uapi.HttpResponse]. Building those literals by hand
// is verbose and, for failures, easy to get subtly wrong: the two halves of an
// error response (the server-side log line and the client-side body) have to be
// written separately, so it is possible to return a 500 that was never logged,
// or to log an error and then forget to return at all.
//
// This package pairs the two halves in a single call. Every helper here is a
// pure function returning the response the caller should return:
//
//	rows, err := state.Pool.Query(d.Context, "...")
//	if err != nil {
//		return resp.Err("Error while getting bot [db fetch]", err, zap.String("id", id))
//	}
//
// # Choosing a helper
//
// Failures fall into two groups, and the distinction matters for security:
//
//   - Server faults (5xx) mean Popplio is broken. The cause is logged with full
//     context and the client receives a fixed, generic body — internal detail
//     such as SQL text or driver errors must never reach the caller. Use [Err],
//     or [ErrDetail] when a specific, safe-to-expose sentence genuinely helps
//     the caller.
//
//   - Client faults (4xx) mean the request was wrong. The caller needs to know
//     what to fix, so the message is returned verbatim in the body and nothing
//     is logged — a malformed request is not a Popplio fault and must not add
//     noise to the error logs. Use [BadRequest], [NotFound], [Forbidden] and so
//     on.
//
// # Message conventions
//
// Messages passed to the 5xx helpers are log lines, so they name the operation
// and the stage that failed, without trailing punctuation:
//
//	"Error while getting bot [db fetch]"
//
// Messages passed to the 4xx helpers are shown to API consumers, so they are
// complete sentences addressed to the caller:
//
//	"Too many `team_includes`. Maximum is 16"
package resp

import (
	"net/http"

	"popplio/state"
	"popplio/types"

	"github.com/infinitybotlist/eureka/ratelimit"
	"github.com/infinitybotlist/eureka/uapi"
	"go.uber.org/zap"
)

// Err logs a server-side fault and returns a 500 carrying uapi's generic
// internal-error body.
//
// This is the correct helper for essentially every unexpected error — failed
// queries, failed upstream calls, failed marshalling. The cause and any extra
// fields are recorded for operators; the caller learns only that something
// broke, which is what keeps internal detail out of the API surface.
//
// reason names the operation and stage that failed and is used as the log
// message. err is attached automatically and may be nil for faults that are not
// carried by an error value.
//
//	if err != nil {
//		return resp.Err("Error while getting bot [db fetch]", err, zap.String("id", id))
//	}
func Err(reason string, err error, fields ...zap.Field) uapi.HttpResponse {
	state.Logger.Error(reason, append([]zap.Field{zap.Error(err)}, fields...)...)
	return uapi.DefaultResponse(http.StatusInternalServerError)
}

// ErrDetail logs a server-side fault exactly as [Err] does, but returns reason
// to the caller as the response body instead of the generic internal-error
// text.
//
// Prefer [Err]. Reach for ErrDetail only when the caller can act on knowing
// which stage failed, and only when reason is safe to expose — it must describe
// the operation, never the underlying error, which stays in the log.
func ErrDetail(reason string, err error, fields ...zap.Field) uapi.HttpResponse {
	state.Logger.Error(reason, append([]zap.Field{zap.Error(err)}, fields...)...)
	return Status(http.StatusInternalServerError, reason+".")
}

// ErrBody logs a server-side fault exactly as [Err] does, but returns message
// to the caller as the response body.
//
// It exists for the case where the log line and the client-facing text genuinely
// need to differ: reason carries the internal detail an operator needs ("Failed
// to refund order [jsonimpl.Unmarshal]") while message carries only what is safe
// and useful to expose ("Failed to refund order."). Keep that split — the point
// of the helper is that internal detail stays in reason.
//
// When the two would say the same thing, [ErrDetail] says it once. When the
// caller has no use for either, prefer [Err].
func ErrBody(reason, message string, err error, fields ...zap.Field) uapi.HttpResponse {
	state.Logger.Error(reason, append([]zap.Field{zap.Error(err)}, fields...)...)
	return Status(http.StatusInternalServerError, message)
}

// Status returns an arbitrary status code with message as the JSON body.
//
// It is the escape hatch behind the named 4xx helpers, for the handful of
// statuses that do not warrant one of their own (402, 415, 424, 503 and the
// like). Reach for a named helper when one exists — Status carries no
// indication of whether the condition should have been logged, so it must not
// be used for 5xx faults. Use [Err] or [ErrDetail] for those.
func Status(status int, message string) uapi.HttpResponse {
	return uapi.HttpResponse{
		Status: status,
		Json:   types.ApiError{Message: message},
	}
}

// BadRequest returns a 400 with message as the JSON body.
//
// Use it when the request itself is malformed or fails validation: a bad query
// parameter, an out-of-range value, a body that cannot be parsed. message is
// shown to the caller, so it should say what was wrong and what is acceptable.
//
//	if len(includes) > 16 {
//		return resp.BadRequest("Too many `team_includes`. Maximum is 16")
//	}
func BadRequest(message string) uapi.HttpResponse {
	return Status(http.StatusBadRequest, message)
}

// NotFound returns a 404 with message as the JSON body.
//
// Use it when the addressed entity does not exist. Note that "exists but is not
// visible to this caller" is often better served by NotFound than [Forbidden],
// since a 403 confirms the entity is real.
func NotFound(message string) uapi.HttpResponse {
	return Status(http.StatusNotFound, message)
}

// Forbidden returns a 403 with message as the JSON body.
//
// Use it when the caller is authenticated but lacks permission for this action.
// message should name the missing permission where that is not sensitive, so
// the caller can tell a permissions problem from a broken request.
func Forbidden(message string) uapi.HttpResponse {
	return Status(http.StatusForbidden, message)
}

// Unauthorized returns a 401 with message as the JSON body.
//
// Use it when credentials are absent, malformed or expired — that is, when
// retrying with valid credentials could succeed. When the caller is properly
// authenticated but simply not allowed, use [Forbidden].
func Unauthorized(message string) uapi.HttpResponse {
	return Status(http.StatusUnauthorized, message)
}

// Conflict returns a 409 with message as the JSON body.
//
// Use it when the request is well-formed but clashes with current state, such
// as creating an entity that already exists.
func Conflict(message string) uapi.HttpResponse {
	return Status(http.StatusConflict, message)
}

// RateLimited returns a 429 for an exceeded rate limit, carrying both the
// human-readable retry time in the body and limit's Retry-After header.
//
// The header is the part that is easy to forget and the part clients actually
// need, which is why this takes the limit rather than a message: callers cannot
// accidentally return a 429 that gives no machine-readable way to back off.
//
//	limit, err := ratelimit.Ratelimit{...}.Limit(d.Context, r)
//	if err != nil {
//		return resp.Err("Error calculating ratelimits", err)
//	}
//	if limit.Exceeded {
//		return resp.RateLimited(limit)
//	}
func RateLimited(limit ratelimit.Limit) uapi.HttpResponse {
	return uapi.HttpResponse{
		Status:  http.StatusTooManyRequests,
		Json:    types.ApiError{Message: "You are being ratelimited. Please try again in " + limit.TimeToReset.String()},
		Headers: limit.Headers(),
	}
}

// WithHeaders returns res with headers merged into its header map, leaving res
// itself untouched.
//
// It exists for endpoints where every response — success and failure alike —
// has to carry the same headers, most commonly a rate limit's. Wrapping at the
// point of return keeps the header from being forgotten on the one error path
// nobody thought about:
//
//	return resp.WithHeaders(resp.BadRequest("Malformed redirect_uri"), limit.Headers())
//
// Existing keys in res win, so a response that deliberately sets a header keeps
// its own value.
func WithHeaders(res uapi.HttpResponse, headers map[string]string) uapi.HttpResponse {
	if len(headers) == 0 {
		return res
	}

	merged := make(map[string]string, len(headers)+len(res.Headers))
	for k, v := range headers {
		merged[k] = v
	}
	for k, v := range res.Headers {
		merged[k] = v
	}

	res.Headers = merged
	return res
}

// OK returns a 200 with body marshalled as the JSON response.
//
// uapi treats a zero Status as 200, so returning uapi.HttpResponse{Json: body}
// is equivalent; OK simply states the intent.
func OK(body any) uapi.HttpResponse {
	return uapi.HttpResponse{Json: body}
}

// Created returns a 201 with body marshalled as the JSON response.
//
// Use it for handlers that bring a new entity into existence and return its
// representation. When there is nothing meaningful to return, [NoContent] is
// the better answer.
func Created(body any) uapi.HttpResponse {
	return uapi.HttpResponse{
		Status: http.StatusCreated,
		Json:   body,
	}
}

// NoContent returns a 204 with an empty body.
//
// It is the usual success response for handlers that mutate or delete state and
// have nothing to report back.
func NoContent() uapi.HttpResponse {
	return uapi.DefaultResponse(http.StatusNoContent)
}
