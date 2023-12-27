package get_rss_feed

import (
	"net/http"
	"popplio/seo"
	"popplio/seo/fetchers"
	"popplio/state"
	"strconv"
	"time"

	"encoding/xml"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"go.uber.org/zap"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get RSS Feed",
		Description: "Gets the RSS feed for the site, in XML format",
		Resp:        seo.RssFeed{},
	}
}

const perPage = 10

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

	rssFeed := seo.RssFeed{}

	rssFeed.NS = "http://www.w3.org/2005/Atom"
	rssFeed.Version = "2.0"
	rssFeed.Channel = &seo.RssChannel{
		Title:         "Infinity Bot List",
		Link:          state.Config.Sites.Frontend.Parse(),
		Description:   "Search our vast list of bots for an exciting start to your server.",
		Language:      "en-us",
		LastBuildDate: time.Now().Format(time.RFC822),
		Copyright:     "Copyright " + time.Now().Format("2006") + " Infinity Development",
		Docs:          "https://www.rssboard.org/rss-specification",
		TTL:           120,
		Category:      []string{"Bots", "Servers"},
		AtomLink: []*seo.RssAtomLink{
			{
				Href: func() string {
					d := state.Config.Sites.API.Parse() + r.URL.Path

					if r.URL.RawQuery != "" {
						d += "?" + r.URL.RawQuery
					}

					return d
				}(),
				Rel:  "self",
				Type: "application/rss+xml",
			},
			{
				Href: state.Config.Sites.API.Parse() + "/list/rss.xml",
				Rel:  "first",
				Type: "application/rss+xml",
			},
			{
				Href: state.Config.Sites.API.Parse() + "/list/rss.xml?page=" + strconv.FormatUint(pageNum+1, 10),
				Rel:  "next",
				Type: "application/rss+xml",
			},
		},
		Links: []*seo.RssLink{
			{
				Href: state.Config.Sites.API.Parse() + "/list/rss.xml",
				Rel:  "first",
			},
			{
				Href: state.Config.Sites.API.Parse() + "/list/rss.xml?page=" + strconv.FormatUint(pageNum+1, 10),
				Rel:  "next",
			},
		},
		Generator: "Popplio RSS Generator",
		Image: &seo.RssImage{
			URL:   state.Config.Sites.CDN + "/core/full_logo.webp",
			Title: "Infinity Bot List Logo",
			Link:  state.Config.Sites.Frontend.Parse(),
		},
	}

	if pageNum > 1 {
		rssFeed.Channel.AtomLink = append(rssFeed.Channel.AtomLink, &seo.RssAtomLink{
			Href: state.Config.Sites.API.Parse() + "/list/rss.xml?page=" + strconv.FormatUint(pageNum-1, 10),
			Rel:  "prev",
			Type: "application/rss+xml",
		})
		rssFeed.Channel.Links = append(rssFeed.Channel.Links, &seo.RssLink{
			Href: state.Config.Sites.API.Parse() + "/list/rss.xml?page=" + strconv.FormatUint(pageNum-1, 10),
			Rel:  "prev",
		})

	}

	var collector = seo.IDCollector{}

	// Get new bots
	rows, err := state.Pool.Query(d.Context, "SELECT bot_id FROM bots ORDER BY created_at DESC LIMIT $1 OFFSET $2", limit, offset)

	if err != nil {
		state.Logger.Error("Failed to get bots [row query] for generating RSS feed", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	defer rows.Close()

	newBots, err := collector.Collect(rows)

	if err != nil {
		state.Logger.Error("Failed to collect bot IDs for generating RSS feed", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	for _, id := range newBots {
		err := state.SeoMapGenerator.AddToRss(d.Context, &fetchers.BotFetcher{}, &rssFeed, "New Bots", id)

		if err != nil {
			state.Logger.Error("Failed to add bot to RSS feed", zap.Error(err), zap.String("botId", id))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	}

	// Get certified bots
	rows, err = state.Pool.Query(d.Context, "SELECT bot_id FROM bots WHERE type = 'certified' ORDER BY created_at DESC LIMIT $1 OFFSET $2", limit, offset)

	if err != nil {
		state.Logger.Error("Failed to get bots [row query] for generating RSS feed", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	defer rows.Close()

	certBots, err := collector.Collect(rows)

	if err != nil {
		state.Logger.Error("Failed to collect bot IDs for generating RSS feed", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	for _, id := range certBots {
		err := state.SeoMapGenerator.AddToRss(d.Context, &fetchers.BotFetcher{}, &rssFeed, "Certified Bots", id)

		if err != nil {
			state.Logger.Error("Failed to add bot to RSS feed", zap.Error(err), zap.String("botId", id))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	}

	// Get premium bots
	rows, err = state.Pool.Query(d.Context, "SELECT bot_id FROM bots WHERE premium = true ORDER BY created_at DESC LIMIT $1 OFFSET $2", limit, offset)

	if err != nil {
		state.Logger.Error("Failed to get bots [row query] for generating RSS feed", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	defer rows.Close()

	premiumBots, err := collector.Collect(rows)

	if err != nil {
		state.Logger.Error("Failed to collect bot IDs for generating RSS feed", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	for _, id := range premiumBots {
		err := state.SeoMapGenerator.AddToRss(d.Context, &fetchers.BotFetcher{}, &rssFeed, "Premium Bots", id)

		if err != nil {
			state.Logger.Error("Failed to add bot to RSS feed", zap.Error(err), zap.String("botId", id))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	}

	body, err := xml.Marshal(rssFeed)

	if err != nil {
		state.Logger.Error("Failed to marshal RSS feed", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	_, err = state.Redis.Set(d.Context, "rssfeed-"+strconv.FormatUint(pageNum, 10), string(body), time.Minute*5).Result()

	if err != nil {
		// Log but dont error for this
		state.Logger.Error("Failed to set RSS feed cache", zap.Error(err))
	}

	return uapi.HttpResponse{
		Status: http.StatusOK,
		Bytes:  body,
		Headers: map[string]string{
			"Content-Type": "application/xml",
		},
	}
}
