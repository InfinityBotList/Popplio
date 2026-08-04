package panel

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"popplio/config"
	"popplio/state"

	"go.uber.org/zap"
)

// TestMain sets up just enough state for the HTTP layer. Nothing here touches
// Postgres or Discord: these tests cover the custom server itself (§4) and the
// response envelope rules (§3.4), which is the part of the port that owes
// nothing to the database.
func TestMain(m *testing.M) {
	state.Logger = zap.NewNop()
	state.Config = &config.Config{
		Arcadia: config.Arcadia{
			ServerPort: config.Differs[int]{Staging: 3011, Prod: 3010},
		},
	}

	os.Exit(m.Run())
}

// handler returns the full middleware chain the real server serves.
func handler(t *testing.T) http.Handler {
	t.Helper()

	return New().http.Handler
}

// CORS is Any/Any/Any and preflights are answered with 204.
func TestCORSPreflight(t *testing.T) {
	rec := httptest.NewRecorder()
	handler(t).ServeHTTP(rec, httptest.NewRequest(http.MethodOptions, "/", nil))

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", rec.Code)
	}

	want := map[string]string{
		"Access-Control-Allow-Origin":  "*",
		"Access-Control-Allow-Methods": "*",
		"Access-Control-Allow-Headers": "*",
	}

	for header, value := range want {
		if got := rec.Header().Get(header); got != value {
			t.Errorf("%s = %q, want %q", header, got, value)
		}
	}

	if rec.Body.Len() != 0 {
		t.Errorf("preflight body = %q, want empty", rec.Body.String())
	}
}

// CORS headers are present on ordinary responses too.
func TestCORSOnNormalResponse(t *testing.T) {
	rec := httptest.NewRecorder()
	handler(t).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/openapi", nil))

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin = %q, want *", got)
	}
}

// GET /openapi serves a valid JSON document as application/json.
func TestOpenAPIRoute(t *testing.T) {
	rec := httptest.NewRecorder()
	handler(t).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/openapi", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}

	var doc map[string]any

	if err := json.Unmarshal(rec.Body.Bytes(), &doc); err != nil {
		t.Fatalf("openapi document is not valid JSON: %v", err)
	}

	paths, ok := doc["paths"].(map[string]any)

	if !ok {
		t.Fatal("openapi document has no paths")
	}

	for _, route := range []string{"/", "/openapi"} {
		if _, ok := paths[route]; !ok {
			t.Errorf("openapi document does not describe %q", route)
		}
	}
}

// The API surface is exactly two routes: POST / and GET /openapi.
func TestRouting(t *testing.T) {
	tests := []struct {
		method string
		path   string
		want   int
	}{
		{http.MethodGet, "/", http.StatusMethodNotAllowed},
		{http.MethodPut, "/", http.StatusMethodNotAllowed},
		{http.MethodPost, "/nope", http.StatusNotFound},
		{http.MethodPost, "/openapi", http.StatusMethodNotAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			handler(t).ServeHTTP(rec, httptest.NewRequest(tt.method, tt.path, nil))

			if rec.Code != tt.want {
				t.Errorf("status = %d, want %d", rec.Code, tt.want)
			}

			if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/plain") {
				t.Errorf("Content-Type = %q, want text/plain", got)
			}
		})
	}
}

// A body that is not a valid PanelQuery is rejected as 400 plain text, not as a
// JSON envelope.
func TestMalformedBodyIsPlainText(t *testing.T) {
	bodies := []string{
		`not json`,
		`{}`,
		`{"NotAVariant":{}}`,
		`{"Hello":{},"BotQueue":{}}`,
	}

	for _, body := range bodies {
		t.Run(body, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))

			handler(t).ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want 400", rec.Code)
			}

			if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/plain") {
				t.Errorf("Content-Type = %q, want text/plain", got)
			}
		})
	}
}

// §3.4: responses are not uniform. JSON bodies are application/json, string
// bodies are text/plain; charset=utf-8, and a 204 carries no body at all.
func TestResponseEnvelopes(t *testing.T) {
	t.Run("json", func(t *testing.T) {
		rec := httptest.NewRecorder()
		writeJSON(http.StatusOK, map[string]string{"a": "b"}).write(rec)

		if got := rec.Header().Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q", got)
		}

		if got := rec.Body.String(); got != `{"a":"b"}` {
			t.Errorf("body = %q", got)
		}
	})

	t.Run("text", func(t *testing.T) {
		rec := httptest.NewRecorder()
		writeText(http.StatusOK, "raw token").write(rec)

		if got := rec.Header().Get("Content-Type"); got != "text/plain; charset=utf-8" {
			t.Errorf("Content-Type = %q", got)
		}

		if got := rec.Body.String(); got != "raw token" {
			t.Errorf("body = %q", got)
		}
	})

	t.Run("no content", func(t *testing.T) {
		rec := httptest.NewRecorder()
		writeNoContent().write(rec)

		if rec.Code != http.StatusNoContent {
			t.Errorf("status = %d, want 204", rec.Code)
		}

		if rec.Body.Len() != 0 {
			t.Errorf("204 body = %q, want empty", rec.Body.String())
		}
	})
}

// A panic becomes a 500 plain-text response rather than a dropped connection.
func TestPanicRecovery(t *testing.T) {
	panicking := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	})

	rec := httptest.NewRecorder()
	recoverMiddleware(panicking).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}

	if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/plain") {
		t.Errorf("Content-Type = %q, want text/plain", got)
	}
}

// The body cap is 1000 MB. CDN chunk uploads ride through this endpoint as JSON
// byte arrays, so lowering it silently breaks uploads.
func TestMaxBodySize(t *testing.T) {
	if maxBodySize != 1_048_576_000 {
		t.Errorf("maxBodySize = %d, want 1048576000", maxBodySize)
	}
}

// The server binds the configured port for the current environment.
func TestListenAddress(t *testing.T) {
	s := New()

	want := "0.0.0.0:3011" // current-env is staging

	if config.CurrentEnv == config.CurrentEnvProd {
		want = "0.0.0.0:3010"
	}

	if s.http.Addr != want {
		t.Errorf("Addr = %q, want %q", s.http.Addr, want)
	}
}
