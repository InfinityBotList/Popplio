package state

import (
	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"mvdan.cc/xurls/v2"
)

var BlueMonday *bluemonday.Policy
var GoldMark = goldmark.New(
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

func setupPolicy() {
	allowedEls := Config.Meta.AllowedHTMLTags

	BlueMonday = bluemonday.NewPolicy()

	// We want to allow custom CSS
	BlueMonday.AllowUnsafe(true)
	BlueMonday.AllowStyling()

	BlueMonday.AllowStandardURLs()
	BlueMonday.AllowURLSchemes("https")
	BlueMonday.AddTargetBlankToFullyQualifiedLinks(true)
	BlueMonday.AllowStandardAttributes()
	BlueMonday.AllowLists()
	BlueMonday.AllowComments()
	BlueMonday.AllowImages()
	BlueMonday.AllowTables()

	BlueMonday.AllowElements(allowedEls...)
	BlueMonday.AllowAttrs("href").OnElements("a")
	BlueMonday.AllowAttrs("style", "class", "src", "href", "code", "id").Globally()

	BlueMonday.AllowAttrs("src", "height", "width").OnElements("iframe")
	BlueMonday.AllowAttrs("src", "alt", "width", "height", "crossorigin", "referrerpolicy", "sizes", "srcset").OnElements("img")
	BlueMonday.AllowAttrs("nowrap").OnElements("td", "th")
}
