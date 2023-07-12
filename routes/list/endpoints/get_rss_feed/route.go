package get_rss_feed

import (
	"net/http"
	"popplio/state"
	"strconv"
	"time"

	"encoding/xml"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5/pgtype"
)

type RssFeed struct {
	XMLName xml.Name `xml:"rss"`
	// NS
	NS string `xml:"xmlns:atom,attr"`
	// Version
	Version string `xml:"version,attr"`
	// Channel
	Channel *RssChannel `xml:"channel"`
}

type RssChannel struct {
	Title         string         `xml:"title"`
	Link          string         `xml:"link"`
	Description   string         `xml:"description"`
	Language      string         `xml:"language"`
	LastBuildDate string         `xml:"lastBuildDate"`
	Copyright     string         `xml:"copyright"`
	Docs          string         `xml:"docs"`
	TTL           int            `xml:"ttl"`
	Category      []string       `xml:"category"`
	AtomLink      []*RssAtomLink `xml:"atom:link"`
	Links         []*RssLink
	Generator     string     `xml:"generator"`
	Image         *RssImage  `xml:"image"`
	Items         []*RssItem `xml:"item"`
}

type RssAtomLink struct {
	XMLName xml.Name `xml:"atom:link"`
	Href    string   `xml:"href,attr"`
	Rel     string   `xml:"rel,attr"`
	Type    string   `xml:"type,attr"`
}

type RssImage struct {
	XMLName xml.Name `xml:"image"`
	// URL
	URL string `xml:"url"`
	// Title
	Title string `xml:"title"`
	// Link
	Link string `xml:"link"`
}

type RssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	Author      string `xml:"author"`
	Category    string `xml:"category"`
	Guid        string `xml:"guid"`
	PubDate     string `xml:"pubDate"`
}

type RssLink struct {
	XMLName xml.Name `xml:"link"`
	Rel     string   `xml:"rel,attr"`
	Href    string   `xml:"href,attr"`
}

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get RSS Feed",
		Description: "Gets the RSS feed for the site, in XML format",
		Resp:        RssFeed{},
	}
}

const perPage = 20

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	page := r.URL.Query().Get("page")

	if page == "" {
		page = "1"
	}

	pageNum, err := strconv.ParseUint(page, 10, 32)

	if err != nil {
		return uapi.DefaultResponse(http.StatusBadRequest)
	}

	// Check cache, this is how we can avoid hefty ratelimits
	cache := state.Redis.Get(d.Context, "rssfeed-"+strconv.FormatUint(pageNum, 10)).Val()
	if cache != "" {
		return uapi.HttpResponse{
			Data: cache,
			Headers: map[string]string{
				"X-Popplio-Cached": "true",
			},
		}
	}

	limit := perPage
	offset := (pageNum - 1) * perPage

	rssFeed := RssFeed{}

	rssFeed.NS = "http://www.w3.org/2005/Atom"
	rssFeed.Version = "2.0"
	rssFeed.Channel = &RssChannel{
		Title:         "Infinity Bot List",
		Link:          state.Config.Sites.Frontend.Parse(),
		Description:   "Search our vast list of bots for an exciting start to your server.",
		Language:      "en-us",
		LastBuildDate: time.Now().Format(time.RFC822),
		Copyright:     "Copyright " + time.Now().Format("2006") + " Infinity Development",
		Docs:          "https://www.rssboard.org/rss-specification",
		TTL:           120,
		Category:      []string{"Bots", "Servers"},
		AtomLink: []*RssAtomLink{
			{
				Href: state.Config.Sites.Frontend.Parse() + "/list/rss.xml?page=" + strconv.FormatUint(pageNum, 10),
				Rel:  "self",
				Type: "application/rss+xml",
			},
			{
				Href: state.Config.Sites.Frontend.Parse() + "/list/rss.xml",
				Rel:  "first",
				Type: "application/rss+xml",
			},
			{
				Href: state.Config.Sites.Frontend.Parse() + "/list/rss.xml?page=" + strconv.FormatUint(pageNum+1, 10),
				Rel:  "next",
				Type: "application/rss+xml",
			},
		},
		Links: []*RssLink{
			{
				Href: state.Config.Sites.Frontend.Parse() + "/list/rss.xml",
				Rel:  "first",
			},
			{
				Href: state.Config.Sites.Frontend.Parse() + "/list/rss.xml?page=" + strconv.FormatUint(pageNum+1, 10),
				Rel:  "next",
			},
		},
		Generator: "Popplio RSS Generator",
		Image: &RssImage{
			URL:   state.Config.Sites.CDN + "/logos/Infinity.png",
			Title: "Infinity Bot List Logo",
			Link:  state.Config.Sites.Frontend.Parse(),
		},
	}

	if pageNum > 1 {
		rssFeed.Channel.AtomLink = append(rssFeed.Channel.AtomLink, &RssAtomLink{
			Href: state.Config.Sites.Frontend.Parse() + "/list/rss.xml?page=" + strconv.FormatUint(pageNum-1, 10),
			Rel:  "prev",
			Type: "application/rss+xml",
		})
		rssFeed.Channel.Links = append(rssFeed.Channel.Links, &RssLink{
			Href: state.Config.Sites.Frontend.Parse() + "/list/rss.xml?page=" + strconv.FormatUint(pageNum-1, 10),
			Rel:  "prev",
		})

	}

	// Get all bots
	rows, err := state.Pool.Query(d.Context, "SELECT bot_id, short, owner, team_owner, created_at FROM bots ORDER BY created_at DESC LIMIT $1 OFFSET $2", limit, offset)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	defer rows.Close()

	for rows.Next() {
		var botID string
		var short string
		var owner pgtype.Text
		var teamOwner pgtype.Text
		var createdAt time.Time

		err = rows.Scan(&botID, &short, &owner, &teamOwner, &createdAt)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		botUser, err := dovewing.GetUser(d.Context, botID, state.DovewingPlatformDiscord)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		rssFeed.Channel.Items = append(rssFeed.Channel.Items, &RssItem{
			Title:       botUser.Username,
			Link:        state.Config.Sites.Frontend.Parse() + "/bots/" + botID,
			Description: short,
			Author: func() string {
				if teamOwner.Valid {
					return teamOwner.String
				}

				return owner.String
			}(),
			Category: "Bots",
			Guid:     botID,
			PubDate:  createdAt.Format(time.RFC822),
		})
	}

	body, err := xml.Marshal(rssFeed)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	return uapi.HttpResponse{
		Status: http.StatusOK,
		Bytes:  body,
		Headers: map[string]string{
			"Content-Type": "application/xml",
		},
	}
}
