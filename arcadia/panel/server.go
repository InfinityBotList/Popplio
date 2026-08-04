package panel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"
	"strconv"
	"time"

	"popplio/arcadia/types"
	"popplio/config"
	"popplio/state"

	"go.uber.org/zap"
)

// maxBodySize is 1000 MB, the limit upstream set. It is far larger than any
// remaining operation needs now that chunked CDN uploads are gone; see
// CONFORMANCE.md before lowering it, since the panel may still post large
// bodies.
const maxBodySize = 1_048_576_000

// Server is the panel API server.
type Server struct {
	http *http.Server
}

// New builds the panel server. It does not listen yet.
func New() *Server {
	s := &Server{}

	mux := http.NewServeMux()
	mux.HandleFunc("/openapi", s.handleOpenAPI)
	mux.HandleFunc("/", s.handleRoot)

	handler := recoverMiddleware(
		loggingMiddleware(
			corsMiddleware(
				maxBodyMiddleware(mux),
			),
		),
	)

	s.http = &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%d", state.Config.Arcadia.ServerPort.Parse()),
		Handler: handler,

		// TIMEOUTS (§14c): long read/write windows were sized for 1 GB chunked
		// uploads. They are left as-is rather than retuned blindly. ReadHeaderTimeout stays short so a stalled client cannot
		// hold a connection open without sending anything. The Rust version set no
		// timeouts at all.
		ReadHeaderTimeout: 30 * time.Second,
		ReadTimeout:       30 * time.Minute,
		WriteTimeout:      30 * time.Minute,
		IdleTimeout:       120 * time.Second,
	}

	return s
}

// Start runs the startup DDL and begins serving. It blocks until the server
// stops.
func (s *Server) Start(ctx context.Context) error {
	if err := ensureAuthchainTable(ctx); err != nil {
		return fmt.Errorf("failed to create staffpanel__authchain table: %w", err)
	}

	state.Logger.Info("Starting panel server", zap.String("addr", s.http.Addr))

	if err := s.http.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

// Shutdown drains the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}

func ensureAuthchainTable(ctx context.Context) error {
	_, err := state.Pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS staffpanel__authchain (
            itag UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
            user_id TEXT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
            token TEXT NOT NULL,
            popplio_token TEXT NOT NULL,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
            state TEXT NOT NULL DEFAULT 'pending'
        )`)

	return err
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		writeText(http.StatusNotFound, "Not Found").write(w)
		return
	}

	if r.Method != http.MethodPost {
		writeText(http.StatusMethodNotAllowed, "Method Not Allowed").write(w)
		return
	}

	var req types.PanelQuery

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// axum rejects an unparseable body with 400 and the serde message.
		writeText(http.StatusBadRequest, "Failed to parse the request body as JSON: "+err.Error()).write(w)
		return
	}

	resp, err := s.dispatch(r.Context(), &req)

	if err != nil {
		var perr Error

		if errors.As(err, &perr) {
			writeText(perr.Status, perr.Message).write(w)
			return
		}

		writeText(http.StatusInternalServerError, err.Error()).write(w)
		return
	}

	resp.write(w)
}

// recoverMiddleware turns a panic into a 500 plain-text response and logs the
// stack.
func recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				state.Logger.Error("panel: panic serving request",
					zap.Any("panic", rec),
					zap.String("path", r.URL.Path),
					zap.ByteString("stack", debug.Stack()),
				)

				writeText(http.StatusInternalServerError, "Internal Server Error").write(w)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// statusRecorder captures the status code for the request log.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rec, r)

		state.Logger.Info("panel",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Int("status", rec.status),
			zap.Duration("took", time.Since(start)),
		)
	})
}

// corsMiddleware reproduces tower-http's CorsLayer with Any/Any/Any and answers
// preflights itself.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func maxBodyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

		next.ServeHTTP(w, r)
	})
}

// instanceDescription is the panel instance blurb in the Hello response.
func instanceDescription() string {
	switch config.CurrentEnv {
	case config.CurrentEnvStaging:
		return "Arcadia Staging Panel Instance"
	case config.CurrentEnvDev:
		return "Arcadia Development Panel Instance"
	default:
		return "Arcadia Production Panel Instance"
	}
}

// serverIDs renders the configured guild ids as strings for CoreConstants.
func serverIDs() types.PanelServers {
	return types.PanelServers{
		Main:    strconv.FormatUint(uint64(state.Config.Servers.Main), 10),
		Staff:   strconv.FormatUint(uint64(state.Config.Servers.Staff), 10),
		Testing: strconv.FormatUint(uint64(state.Config.Servers.Testing), 10),
	}
}
