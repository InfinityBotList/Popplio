package parse_html

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"popplio/api"
	"popplio/docs"
	"popplio/state"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"mvdan.cc/xurls/v2"
)

var p *bluemonday.Policy
var gm = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,
		extension.NewLinkify(
			extension.WithLinkifyAllowedProtocols([][]byte{
				[]byte("http:"),
				[]byte("https:"),
			}),
			extension.WithLinkifyURLRegexp(
				xurls.Strict(),
			),
		),
		extension.Typographer,
	),
	goldmark.WithParserOptions(
		parser.WithAutoHeadingID(),
	),
	goldmark.WithRendererOptions(
		html.WithHardWraps(),
		html.WithUnsafe(),
	),
)

func Setup() {
	allowedEls := state.Config.Meta.AllowedHTMLTags

	p = bluemonday.NewPolicy()

	// We want to allow custom CSS
	p.AllowUnsafe(true)
	p.AllowStyling()

	p.AllowStandardURLs()
	p.AllowURLSchemes("https")
	p.AddTargetBlankToFullyQualifiedLinks(true)
	p.AllowStandardAttributes()
	p.AllowLists()
	p.AllowComments()
	p.AllowImages()
	p.AllowTables()

	p.AllowElements(allowedEls...)
	p.AllowAttrs("href").OnElements("a")
	p.AllowAttrs("style", "class", "src", "href", "code", "id").Globally()

	p.AllowAttrs("src", "height", "width").OnElements("iframe")
	p.AllowAttrs("src", "alt", "width", "height", "crossorigin", "referrerpolicy", "sizes", "srcset").OnElements("img")
	p.AllowAttrs("nowrap").OnElements("td", "th")
}

func Docs() *docs.Doc {
	return &docs.Doc{
		Method:      "POST",
		Path:        "/list/parse-html",
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
		var sanitized = p.SanitizeBytes(bodyBytes)

		return api.HttpResponse{
			Bytes: sanitized,
		}
	}

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	var buf bytes.Buffer

	err = gm.Convert(bodyBytes, &buf)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	bytes := buf.Bytes()

	// Now sanitize the HTML with bluemonday
	var sanitized = p.SanitizeBytes(bytes)

	return api.HttpResponse{
		Bytes: sanitized,
	}
}
