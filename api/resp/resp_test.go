package resp

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"popplio/constants"
	"popplio/state"
	"popplio/types"

	"github.com/infinitybotlist/eureka/ratelimit"
	"github.com/infinitybotlist/eureka/uapi"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// setup wires up the two globals the helpers reach into — state.Logger, which
// the 5xx helpers write to, and uapi.State.Constants, which backs
// uapi.DefaultResponse — and returns a sink holding everything logged.
//
// Both globals are restored when the test ends so ordering never matters.
func setup(t *testing.T) *observer.ObservedLogs {
	t.Helper()

	core, logs := observer.New(zapcore.DebugLevel)

	prevLogger := state.Logger
	state.Logger = zap.New(core)
	t.Cleanup(func() { state.Logger = prevLogger })

	prevState := uapi.State
	uapi.SetupState(uapi.UAPIState{
		Constants: &uapi.UAPIConstants{
			ResourceNotFound:    constants.ResourceNotFound,
			BadRequest:          constants.BadRequest,
			Forbidden:           constants.Forbidden,
			Unauthorized:        constants.Unauthorized,
			InternalServerError: constants.InternalServerError,
			MethodNotAllowed:    constants.MethodNotAllowed,
			BodyRequired:        constants.BodyRequired,
		},
	})
	t.Cleanup(func() { uapi.State = prevState })

	return logs
}

// apiErrorMessage asserts the response body is a types.ApiError and returns its
// message.
func apiErrorMessage(t *testing.T, res uapi.HttpResponse) string {
	t.Helper()

	apiErr, ok := res.Json.(types.ApiError)
	if !ok {
		t.Fatalf("body = %#v, want types.ApiError", res.Json)
	}

	return apiErr.Message
}

// Err must log the cause but must not leak it into the response body, which is
// the whole reason handlers are expected to prefer it over ErrDetail.
func TestErrLogsCauseAndReturnsGenericBody(t *testing.T) {
	logs := setup(t)

	cause := errors.New("pq: relation \"bots\" does not exist")
	res := Err("Error while getting bot [db fetch]", cause, zap.String("id", "1234"))

	if res.Status != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", res.Status, http.StatusInternalServerError)
	}

	if res.Data != constants.InternalServerError {
		t.Errorf("body = %q, want the generic internal-error body", res.Data)
	}

	if res.Json != nil {
		t.Errorf("Json = %#v, want nil so the generic Data body is what gets sent", res.Json)
	}

	entries := logs.All()
	if len(entries) != 1 {
		t.Fatalf("logged %d entries, want 1", len(entries))
	}

	entry := entries[0]
	if entry.Level != zapcore.ErrorLevel {
		t.Errorf("level = %v, want error", entry.Level)
	}

	if entry.Message != "Error while getting bot [db fetch]" {
		t.Errorf("message = %q, want the reason passed in", entry.Message)
	}

	fields := entry.ContextMap()
	if got := fields["error"]; got != cause.Error() {
		t.Errorf("error field = %v, want %q", got, cause)
	}

	if got := fields["id"]; got != "1234" {
		t.Errorf("id field = %v, want \"1234\"", got)
	}
}

// A nil cause is legal — some faults are conditions rather than error values —
// and must not panic.
func TestErrAcceptsNilCause(t *testing.T) {
	logs := setup(t)

	res := Err("Error while getting bot [no rows]", nil)

	if res.Status != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", res.Status, http.StatusInternalServerError)
	}

	if len(logs.All()) != 1 {
		t.Fatalf("logged %d entries, want 1", len(logs.All()))
	}
}

// ErrDetail logs exactly as Err does, but surfaces the reason to the caller as a
// sentence.
func TestErrDetailReturnsReasonAsBody(t *testing.T) {
	logs := setup(t)

	res := ErrDetail("Error while getting bot [db fetch]", errors.New("boom"))

	if res.Status != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", res.Status, http.StatusInternalServerError)
	}

	if got := apiErrorMessage(t, res); got != "Error while getting bot [db fetch]." {
		t.Errorf("body message = %q, want the reason with a trailing period", got)
	}

	if len(logs.All()) != 1 {
		t.Fatalf("logged %d entries, want 1", len(logs.All()))
	}
}

// The 4xx helpers describe a caller mistake, so they return the message
// verbatim and must stay out of the error log.
func TestClientErrorsReturnMessageAndDoNotLog(t *testing.T) {
	tests := []struct {
		name       string
		call       func(string) uapi.HttpResponse
		wantStatus int
	}{
		{"BadRequest", BadRequest, http.StatusBadRequest},
		{"NotFound", NotFound, http.StatusNotFound},
		{"Forbidden", Forbidden, http.StatusForbidden},
		{"Unauthorized", Unauthorized, http.StatusUnauthorized},
		{"Conflict", Conflict, http.StatusConflict},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logs := setup(t)

			const message = "Too many `team_includes`. Maximum is 16"
			res := tt.call(message)

			if res.Status != tt.wantStatus {
				t.Errorf("status = %d, want %d", res.Status, tt.wantStatus)
			}

			if got := apiErrorMessage(t, res); got != message {
				t.Errorf("body message = %q, want %q", got, message)
			}

			if n := len(logs.All()); n != 0 {
				t.Errorf("logged %d entries, want 0 — client mistakes are not Popplio faults", n)
			}
		})
	}
}

// The Retry-After header is the reason RateLimited exists, so it is the thing
// most worth pinning down.
func TestRateLimitedSetsRetryAfterHeader(t *testing.T) {
	logs := setup(t)

	res := RateLimited(ratelimit.Limit{
		Exceeded:    true,
		TimeToReset: 90 * time.Second,
	})

	if res.Status != http.StatusTooManyRequests {
		t.Errorf("status = %d, want %d", res.Status, http.StatusTooManyRequests)
	}

	if got := res.Headers["Retry-After"]; got != "90" {
		t.Errorf("Retry-After = %q, want \"90\"", got)
	}

	if got := apiErrorMessage(t, res); got != "You are being ratelimited. Please try again in 1m30s" {
		t.Errorf("body message = %q, want the time remaining", got)
	}

	if n := len(logs.All()); n != 0 {
		t.Errorf("logged %d entries, want 0 — a ratelimited caller is not a Popplio fault", n)
	}
}

func TestStatusUsesGivenCode(t *testing.T) {
	setup(t)

	res := Status(http.StatusServiceUnavailable, "Paypal is not available")

	if res.Status != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", res.Status, http.StatusServiceUnavailable)
	}

	if got := apiErrorMessage(t, res); got != "Paypal is not available" {
		t.Errorf("body message = %q, want the message passed in", got)
	}
}

func TestSuccessHelpers(t *testing.T) {
	setup(t)

	body := types.ItemList[string]{Items: []string{"a"}}

	t.Run("OK", func(t *testing.T) {
		res := OK(body)

		// uapi treats a zero Status as 200.
		if res.Status != 0 && res.Status != http.StatusOK {
			t.Errorf("status = %d, want 0 or %d", res.Status, http.StatusOK)
		}

		if _, ok := res.Json.(types.ItemList[string]); !ok {
			t.Errorf("Json = %#v, want the body passed in", res.Json)
		}
	})

	t.Run("Created", func(t *testing.T) {
		res := Created(body)

		if res.Status != http.StatusCreated {
			t.Errorf("status = %d, want %d", res.Status, http.StatusCreated)
		}

		if _, ok := res.Json.(types.ItemList[string]); !ok {
			t.Errorf("Json = %#v, want the body passed in", res.Json)
		}
	})

	t.Run("NoContent", func(t *testing.T) {
		res := NoContent()

		if res.Status != http.StatusNoContent {
			t.Errorf("status = %d, want %d", res.Status, http.StatusNoContent)
		}

		if res.Json != nil {
			t.Errorf("Json = %#v, want nil", res.Json)
		}
	})
}
