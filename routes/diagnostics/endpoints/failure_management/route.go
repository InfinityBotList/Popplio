package failure_management

import (
	"bytes"
	"io"
	"net/http"
	"popplio/state"
	"popplio/types"
	"time"

	"github.com/getsentry/sentry-go"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/jsonimpl"
	"github.com/infinitybotlist/eureka/uapi"
	"go.uber.org/zap"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Trace Frontend",
		Description: "Traces an issue in the frontend. This can either be to our self-hosted Sentry instance or to any other (self-hosted) tracing service",
		Req:         []byte{},
		Resp:        types.ApiError{},
		Params: []docs.Parameter{
			{
				Name:        "to",
				In:          "query",
				Description: "This can be 'br0' for self-hosted sentry (trace.infinitybots.gg). Other values are not supported yet.",
				Required:    true,
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	to := r.URL.Query().Get("to")

	switch to {
	case "br0":
		if r.Body == nil {
			return uapi.HttpResponse{
				Status: http.StatusBadRequest,
				Json:   types.ApiError{Message: "No body sent"},
			}
		}

		bodyBytes, err := io.ReadAll(r.Body)

		if err != nil {
			state.Logger.Error("Failed to read body", zap.Error(err), zap.Int("size", len(bodyBytes)))
			return uapi.HttpResponse{
				Status: http.StatusBadRequest,
				Json:   types.ApiError{Message: "Failed to read body: " + err.Error()},
			}
		}

		if len(bodyBytes) == 0 {
			return uapi.HttpResponse{
				Status: http.StatusBadRequest,
				Json:   types.ApiError{Message: "Empty body"},
			}
		}

		bodySplit := bytes.Split(bodyBytes, []byte("\n"))

		var dsn string

		for _, line := range bodySplit {
			var dsnData struct {
				Dsn string `json:"dsn"`
			}

			err = jsonimpl.Unmarshal(line, &dsnData)

			if err != nil {
				return uapi.HttpResponse{
					Status: http.StatusBadRequest,
					Json:   types.ApiError{Message: "Invalid sentry envelope"},
				}
			}

			if dsnData.Dsn != "" {
				dsn = dsnData.Dsn
				break
			}
		}

		if dsn == "" {
			return uapi.HttpResponse{
				Status: http.StatusBadRequest,
				Json:   types.ApiError{Message: "No dsn found in envelope"},
			}
		}

		// Parse dsn
		dsnParsed, err := sentry.NewDsn(dsn)

		if err != nil {
			return uapi.HttpResponse{
				Status: http.StatusBadRequest,
				Json:   types.ApiError{Message: "Invalid dsn"},
			}
		}

		if dsnParsed.GetHost() != "trace.infinitybots.gg" {
			return uapi.HttpResponse{
				Status: http.StatusBadRequest,
				Json:   types.ApiError{Message: "Untrusted dsn"},
			}
		}

		// Send to sentry
		req, err := http.NewRequest("POST", dsnParsed.GetAPIURL().String(), bytes.NewReader(bodyBytes))

		if err != nil {
			return uapi.HttpResponse{
				Status: http.StatusInternalServerError,
				Json:   types.ApiError{Message: "Failed to create request: " + err.Error()},
			}
		}

		for k, v := range dsnParsed.RequestHeaders() {
			req.Header.Set(k, v)
		}

		req.Header.Set("Content-Type", "application/text; charset=utf-8")
		//req.Header.Set("User-Agent", r.Header.Get("User-Agent"))

		client := http.Client{
			Timeout: 30 * time.Second,
		}

		resp, err := client.Do(req)

		if err != nil {
			return uapi.HttpResponse{
				Status: http.StatusInternalServerError,
				Json:   types.ApiError{Message: "Failed to send request: " + err.Error()},
			}
		}

		defer resp.Body.Close()

		respBytes, err := io.ReadAll(resp.Body)

		if err != nil {
			return uapi.HttpResponse{
				Status: http.StatusInternalServerError,
				Json:   types.ApiError{Message: "Failed to read response: " + err.Error()},
			}
		}

		return uapi.HttpResponse{
			Status: resp.StatusCode,
			Bytes:  respBytes,
		}
	default:
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "invalid 'to' value"},
		}
	}
}
