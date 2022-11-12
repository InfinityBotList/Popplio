// Modified version of zapchi for Popplio
package zapchi

import (
	"net/http"
	"popplio/utils"
	"time"

	"github.com/go-chi/chi/middleware"
	"go.uber.org/zap"
)

// Logger is a Chi middleware that logs each request recived using
// the provided Zap logger, sugared or not.
// Provide a name if you want to set the caller (`.Named()`)
// otherwise leave blank.
func Logger(l interface{}, name string) func(next http.Handler) http.Handler {
	switch logger := l.(type) {
	case *zap.SugaredLogger:
		logger = zap.New(logger.Desugar().Core(), zap.AddCallerSkip(1)).Sugar().Named(name)
		return func(next http.Handler) http.Handler {
			fn := func(w http.ResponseWriter, r *http.Request) {
				ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
				t1 := time.Now()
				next.ServeHTTP(ww, r)

				logger.With(
					zap.Int("status", ww.Status()),
					zap.String("statusText", http.StatusText(ww.Status())),
					zap.String("method", r.Method),
					zap.String("url", r.URL.String()),
					zap.String("reqIp", r.RemoteAddr),
					zap.String("protocol", r.Proto),
					zap.Int("size", ww.BytesWritten()),
					zap.String("latency", time.Since(t1).String()),
					zap.String("userAgent", r.UserAgent()),
					zap.String("reqId", utils.RandString(12)),
				).Info("Got Request")
			}
			return http.HandlerFunc(fn)
		}
	default:
		// Log error and exit
		panic("Unknown logger passed in. Please provide *Zap.SugaredLogger")
	}
}
