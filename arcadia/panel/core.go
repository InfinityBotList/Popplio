// Package panel is the staff panel API: one POST / endpoint dispatching a
// tagged-union request over ~25 operations, plus GET /openapi.
//
// It deliberately does NOT reuse Popplio's chi router or uapi envelope. The
// panel protocol has no JSON envelope, returns bare text bodies and 204s, and
// accepts 1 GB request bodies, none of which survive Popplio's global middleware
// (which pins Content-Type: application/json, caps bodies at 50 MB and applies a
// 30s timeout). It is a second http.Server on its own port, built on net/http
// only - no additional HTTP framework.
package panel

import (
	"encoding/json"
	"io"
	"net/http"

	"popplio/state"

	"go.uber.org/zap"
)

// Error is the panel error type: a status and a message rendered as PLAIN TEXT.
//
// The default constructor uses 500, which is why auth failures surface as 500
// with the bare strings "identityExpired" / "sessionNotActive" - the frontend
// matches on those exact strings.
type Error struct {
	Status  int
	Message string
}

func (e Error) Error() string {
	return e.Message
}

// newError is Error::new: status 500, message from the underlying error.
func newError(err error) Error {
	return Error{Status: http.StatusInternalServerError, Message: err.Error()}
}

// errStatus builds an explicit-status error.
func errStatus(status int, message string) Error {
	return Error{Status: status, Message: message}
}

// response is what a handler returns. Exactly one body form is used.
type response struct {
	status int

	// json is marshalled and served as application/json.
	json any
	// text is served as text/plain; charset=utf-8.
	text *string
	// stream is served as application/octet-stream and closed by the writer.
	stream io.ReadCloser
	// noBody sends no body at all (204).
	noBody bool
}

func writeJSON(status int, v any) response {
	return response{status: status, json: v}
}

func writeText(status int, s string) response {
	return response{status: status, text: &s}
}

func writeNoContent() response {
	return response{status: http.StatusNoContent, noBody: true}
}

func writeStream(status int, body io.ReadCloser) response {
	return response{status: status, stream: body}
}

// write renders a response. Content types follow axum's IntoResponse: JSON
// bodies are application/json, string bodies are text/plain; charset=utf-8.
func (r response) write(w http.ResponseWriter) {
	switch {
	case r.stream != nil:
		defer r.stream.Close()

		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(r.status)

		if _, err := io.Copy(w, r.stream); err != nil {
			state.Logger.Error("panel: failed streaming response body", zap.Error(err))
		}
	case r.json != nil:
		body, err := json.Marshal(r.json)

		if err != nil {
			state.Logger.Error("panel: failed marshalling response", zap.Error(err))
			writeText(http.StatusInternalServerError, err.Error()).write(w)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(r.status)
		w.Write(body)
	case r.text != nil:
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(r.status)
		w.Write([]byte(*r.text))
	default:
		w.WriteHeader(r.status)
	}
}
