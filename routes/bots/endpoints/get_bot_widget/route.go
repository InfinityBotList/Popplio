package get_bot_widget

import (
	"bytes"
	"net/http"
	"popplio/api"
	"popplio/constants"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"strings"

	svg "github.com/ajstarks/svgo"
	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return docs.Route(&docs.Doc{
		Method:  "GET",
		Path:    "/bots/{id}/widget",
		OpId:    "get_bot_widget",
		Summary: "Get Bot Widget",
		Description: `
Creates a bot widget using a SVG. The widget will be cached for one hour.
		`,
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "The bots ID, name or vanity",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.Bot{},
		Tags: []string{api.CurrentTag},
	})
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	name := chi.URLParam(r, "id")

	name = strings.ToLower(name)

	if name == "" {
		return api.DefaultResponse(http.StatusBadRequest)
	}

	// Check cache, this is how we can avoid hefty ratelimits
	cache := state.Redis.Get(d.Context, "bcw-"+name).Val()
	if cache != "" {
		return api.HttpResponse{
			Data: cache,
			Headers: map[string]string{
				"X-Popplio-Cached": "true",
			},
		}
	}

	var botId string
	err := state.Pool.QueryRow(d.Context, "SELECT bot_id FROM bots WHERE "+constants.ResolveBotSQL, name).Scan(&botId)

	if err != nil {
		return api.DefaultResponse(http.StatusNotFound)
	}

	// These variables can be changed in query parameters in the future
	var (
		height     = 400
		logoHeight = 40
		color      = "#000000"
	)

	// Height = width * 1.5
	width := int(float64(height) * 1.5)

	// Using svgo to create the SVG
	svgBuf := bytes.NewBuffer([]byte{})

	canvas := svg.New(svgBuf)

	canvas.Start(width, height)

	canvas.Rect(0, 0, width, height, "fill: "+color)

	canvas.Image(0, height-logoHeight, logoHeight, logoHeight, "https://cdn.infinitybots.xyz/images/core/logo.webp", "preserveAspectRatio: none")
	canvas.Text(logoHeight+10, height-(logoHeight/3), "Infinity Bot List", "text-anchor: start; font-size: 20px; fill: #fff")
	//canvas.Text(300, 300, "Hello World", "text-anchor: middle; font-size: 30px; fill: #fff")

	canvas.End()

	return api.HttpResponse{
		Bytes: svgBuf.Bytes(),
		Headers: map[string]string{
			"Content-Type": "image/svg+xml",
		},
	}
}
