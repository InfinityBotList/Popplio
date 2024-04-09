package get_sitemap

import (
	"encoding/xml"
	"net/http"
	"popplio/seo"
	"popplio/seo/fetchers"
	"popplio/state"
	"strconv"
	"time"

	docs "github.com/infinitybotlist/eureka/doclib"

	"github.com/infinitybotlist/eureka/uapi"
	"go.uber.org/zap"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Sitemap",
		Description: "Gets the sitemap for the site, in XML format",
		Resp:        seo.Sitemap{},
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
	cache := state.Redis.Get(d.Context, "sitemap-"+strconv.FormatUint(pageNum, 10)).Val()
	if cache != "" {
		return uapi.HttpResponse{
			Data: cache,
			Headers: map[string]string{
				"X-Popplio-Cached": "true",
				"Content-Type":     "application/xml",
			},
		}
	}

	limit := perPage
	offset := (pageNum - 1) * perPage

	sitemap := seo.Sitemap{}
	sitemap.XMLNS = "http://www.sitemaps.org/schemas/sitemap/0.9"
	sitemap.Urls = make([]*seo.SitemapURL, 0)

	var collector = seo.IDCollector{}

	// Get new bots
	rows, err := state.Pool.Query(d.Context, "SELECT bot_id FROM bots WHERE (type = 'approved' OR type = 'certified') ORDER BY created_at DESC LIMIT $1 OFFSET $2", limit, offset)

	if err != nil {
		state.Logger.Error("Failed to get bots [row query] for generating sitemap", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	defer rows.Close()

	newBots, err := collector.Collect(rows)

	if err != nil {
		state.Logger.Error("Failed to collect bot IDs for generating sitemap", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	for _, id := range newBots {
		err := state.SeoMapGenerator.AddToSitemap(d.Context, &fetchers.BotFetcher{}, &sitemap, "New Bots", id)

		if err != nil {
			state.Logger.Error("Failed to add bot to sitemap", zap.Error(err), zap.String("botId", id))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	}

	// Get certified bots
	rows, err = state.Pool.Query(d.Context, "SELECT bot_id FROM bots WHERE type = 'certified' ORDER BY created_at DESC LIMIT $1 OFFSET $2", limit, offset)

	if err != nil {
		state.Logger.Error("Failed to get bots [row query] for generating sitemap", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	defer rows.Close()

	certBots, err := collector.Collect(rows)

	if err != nil {
		state.Logger.Error("Failed to collect bot IDs for generating sitemap", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	for _, id := range certBots {
		err := state.SeoMapGenerator.AddToSitemap(d.Context, &fetchers.BotFetcher{}, &sitemap, "Certified Bots", id)

		if err != nil {
			state.Logger.Error("Failed to add bot to sitemap", zap.Error(err), zap.String("botId", id))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	}

	// Get premium bots
	rows, err = state.Pool.Query(d.Context, "SELECT bot_id FROM bots WHERE premium = true AND (type = 'approved' OR type = 'certified') ORDER BY created_at DESC LIMIT $1 OFFSET $2", limit, offset)

	if err != nil {
		state.Logger.Error("Failed to get bots [row query] for generating sitemap", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	defer rows.Close()

	premiumBots, err := collector.Collect(rows)

	if err != nil {
		state.Logger.Error("Failed to collect bot IDs for generating sitemap", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	for _, id := range premiumBots {
		err := state.SeoMapGenerator.AddToSitemap(d.Context, &fetchers.BotFetcher{}, &sitemap, "Premium Bots", id)

		if err != nil {
			state.Logger.Error("Failed to add bot to sitemap", zap.Error(err), zap.String("botId", id))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	}

	// Add the next page as a sitemap
	sitemap.Urls = append(sitemap.Urls, &seo.SitemapURL{
		Loc:     state.Config.Sites.API.Parse() + "/list/sitemap.xml?page=" + strconv.FormatUint(pageNum+1, 10),
		LastMod: time.Now().Format(time.RFC3339),
	})

	body, err := xml.Marshal(sitemap)

	if err != nil {
		state.Logger.Error("Failed to marshal sitemap", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	_, err = state.Redis.Set(d.Context, "sitemap-"+strconv.FormatUint(pageNum, 10), string(body), time.Minute*5).Result()

	if err != nil {
		// Log but dont error for this
		state.Logger.Error("Failed to set sitemap cache", zap.Error(err))
	}

	return uapi.HttpResponse{
		Status: http.StatusOK,
		Bytes:  body,
		Headers: map[string]string{
			"Content-Type": "application/xml",
		},
	}
}
