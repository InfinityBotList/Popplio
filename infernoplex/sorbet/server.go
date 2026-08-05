package sorbet

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"popplio/state"

	"go.uber.org/zap"
)

const maxBodySize = 1_048_576_000

type Server struct {
	http *http.Server
}

func New() *Server {
	s := &Server{}

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleRoot)

	handler := recoverMiddleware(loggingMiddleware(corsMiddleware(maxBodyMiddleware(mux))))

	s.http = &http.Server{
		Addr:              fmt.Sprintf("127.0.0.1:%d", state.Config.Infernoplex.ServerPort.Parse()),
		Handler:           handler,
		ReadHeaderTimeout: 30 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	return s
}

func (s *Server) Start() error {
	state.Logger.Info("Starting Sorbet server", zap.String("addr", s.http.Addr))

	if err := s.http.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "Not Found"}, nil)
		return
	}

	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"message": "Method Not Allowed"}, nil)
		return
	}

	var req InfernoplexQuery

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Failed to parse the request body as JSON: " + err.Error()}, nil)
		return
	}

	res := dispatch(r.Context(), req)

	writeJSON(w, res.status, res.body, res.headers)
}

func writeJSON(w http.ResponseWriter, status int, body any, headers map[string]string) {
	for k, v := range headers {
		w.Header().Set(k, v)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if body != nil {
		_ = json.NewEncoder(w).Encode(body)
	}
}

func recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				state.Logger.Error("sorbet: panic serving request",
					zap.Any("panic", rec),
					zap.String("path", r.URL.Path),
					zap.ByteString("stack", debug.Stack()),
				)

				writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "Internal Server Error"}, nil)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

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

		state.Logger.Info("sorbet",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Int("status", rec.status),
			zap.Duration("took", time.Since(start)),
		)
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.Header().Set("Access-Control-Expose-Headers", "X-Session-Invalid")

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
