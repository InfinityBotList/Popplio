package parse_html

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"popplio/api"
	"popplio/state"

	docs "github.com/infinitybotlist/doclib"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Parse HTML",
		Description: "Sanitizes a HTML string for use in previews or on long descriptions",
		Resp:        "Sanitized HTML",
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	// Read body
	var bodyBytes, err = io.ReadAll(r.Body)

	if strings.HasPrefix(string(bodyBytes), "<html>") {
		bodyBytes = []byte(string(bodyBytes)[6:])

		// Now sanitize the HTML with bluemonday
		var sanitized = state.BlueMonday.SanitizeBytes(bodyBytes)

		return api.HttpResponse{
			Bytes: sanitized,
		}
	}

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	var buf bytes.Buffer

	err = state.GoldMark.Convert(bodyBytes, &buf)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	bytes := buf.Bytes()

	// Now sanitize the HTML with bluemonday
	var sanitized = state.BlueMonday.SanitizeBytes(bytes)

	return api.HttpResponse{
		Bytes: sanitized,
	}
}
