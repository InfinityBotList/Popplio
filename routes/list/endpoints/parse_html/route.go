package parse_html

import (
	"bytes"
	"io"
	"net/http"
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
	),
	goldmark.WithParserOptions(
		parser.WithAutoHeadingID(),
	),
	goldmark.WithRendererOptions(
		html.WithHardWraps(),
		html.WithUnsafe(),
	),
)

func init() {
	allowedEls := []string{
		"a",
		"button",
		"span",
		"img",
		"video",
		"iframe",
		"style",
		"span",
		"p",
		"br",
		"center",
		"div",
		"h1",
		"h2",
		"h3",
		"h4",
		"h5",
		"section",
		"article",
		"lang",
		"code",
		"pre",
		"strong",
		"em",
	}

	p = bluemonday.NewPolicy()

	// We want to allow custom CSS
	p.AllowUnsafe(true)
	p.AllowStyling()

	p.AllowStandardURLs()
	p.AllowURLSchemes("https")
	p.AllowAttrs("href").OnElements("a")
	p.AddTargetBlankToFullyQualifiedLinks(true)
	p.AllowStandardAttributes()
	p.AllowLists()
	p.AllowImages()
	p.AllowTables()

	p.AllowElements(allowedEls...)
	p.AllowAttrs("style", "class", "src", "href", "code").Globally()

	p.AllowAttrs("src", "height", "width").OnElements("iframe")
	p.AllowAttrs("src", "alt", "width", "height", "crossorigin", "referrerpolicy", "sizes", "srcset").OnElements("img")
	p.AllowAttrs("nowrap").OnElements("td", "th")
}

func Docs() *docs.Doc {
	return docs.Route(&docs.Doc{
		Method:      "POST",
		Path:        "/list/parse-html",
		OpId:        "parse_html",
		Summary:     "Parse HTML",
		Description: "Sanitizes a HTML string for use in previews or on long descriptions",
		Tags:        []string{api.CurrentTag},
		Resp:        "Sanitized HTML",
	})
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	// Read body
	var bodyBytes, err = io.ReadAll(r.Body)

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
