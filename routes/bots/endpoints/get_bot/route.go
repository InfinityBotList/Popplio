package get_bot

import (
	"net/http"
	"popplio/api"
	"popplio/constants"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"strings"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// A bot is a Discord bot that is on the infinitybotlist.
type Bot struct {
	ITag                     pgtype.UUID          `db:"itag" json:"itag"`
	BotID                    string               `db:"bot_id" json:"bot_id"`
	ClientID                 string               `db:"client_id" json:"client_id"`
	QueueName                string               `db:"queue_name" json:"queue_name"` // Used purely by the queue system
	ExtraLinks               []types.Link         `db:"extra_links" json:"extra_links"`
	Tags                     []string             `db:"tags" json:"tags"`
	Prefix                   pgtype.Text          `db:"prefix" json:"prefix"`
	User                     *types.DiscordUser   `json:"user"` // Must be parsed internally
	Owner                    string               `db:"owner" json:"-"`
	MainOwner                *types.DiscordUser   `json:"owner"` // Must be parsed internally
	AdditionalOwners         []string             `db:"additional_owners" json:"-"`
	ResolvedAdditionalOwners []*types.DiscordUser `json:"additional_owners"` // Must be parsed internally
	StaffBot                 bool                 `db:"staff_bot" json:"staff_bot"`
	Short                    string               `db:"short" json:"short"`
	Long                     string               `db:"long" json:"long"`
	Library                  string               `db:"library" json:"library"`
	NSFW                     pgtype.Bool          `db:"nsfw" json:"nsfw"`
	Premium                  pgtype.Bool          `db:"premium" json:"premium"`
	PendingCert              pgtype.Bool          `db:"pending_cert" json:"pending_cert"`
	Servers                  int                  `db:"servers" json:"servers"`
	Shards                   int                  `db:"shards" json:"shards"`
	ShardList                []int                `db:"shard_list" json:"shard_list"`
	Users                    int                  `db:"users" json:"users"`
	Votes                    int                  `db:"votes" json:"votes"`
	Views                    int                  `db:"clicks" json:"clicks"`
	UniqueClicks             int64                `json:"unique_clicks"` // Must be parsed internally
	InviteClicks             int                  `db:"invite_clicks" json:"invites"`
	Banner                   pgtype.Text          `db:"banner" json:"banner"`
	Invite                   pgtype.Text          `db:"invite" json:"invite"`
	Type                     string               `db:"type" json:"type"` // For auditing reasons, we do not filter out denied/banned bots in API
	Vanity                   string               `db:"vanity" json:"vanity"`
	ExternalSource           pgtype.Text          `db:"external_source" json:"external_source"`
	ListSource               pgtype.UUID          `db:"list_source" json:"list_source"`
	VoteBanned               bool                 `db:"vote_banned" json:"vote_banned"`
	CrossAdd                 bool                 `db:"cross_add" json:"cross_add"`
	StartPeriod              pgtype.Timestamptz   `db:"start_premium_period" json:"start_premium_period"`
	SubPeriod                time.Duration        `db:"premium_period_length" json:"-"`
	SubPeriodParsed          types.Interval       `db:"-" json:"premium_period_length"` // Must be parsed internally
	CertReason               pgtype.Text          `db:"cert_reason" json:"cert_reason"`
	Announce                 bool                 `db:"announce" json:"announce"`
	AnnounceMessage          pgtype.Text          `db:"announce_message" json:"announce_message"`
	Uptime                   int                  `db:"uptime" json:"uptime"`
	TotalUptime              int                  `db:"total_uptime" json:"total_uptime"`
	ClaimedBy                pgtype.Text          `db:"claimed_by" json:"claimed_by"`
	Note                     pgtype.Text          `db:"approval_note" json:"approval_note"`
	CreatedAt                pgtype.Timestamptz   `db:"created_at" json:"created_at"`
	LastClaimed              pgtype.Timestamptz   `db:"last_claimed" json:"last_claimed"`
}

var (
	botColsArr = utils.GetCols(Bot{})
	botCols    = strings.Join(botColsArr, ",")
)

func Docs() *docs.Doc {
	return docs.Route(&docs.Doc{
		Method:  "GET",
		Path:    "/bots/{id}",
		OpId:    "get_bot",
		Summary: "Get Bot",
		Description: `
Gets a bot by id or name

**Some things to note:**

-` + constants.BackTick + constants.BackTick + `external_source` + constants.BackTick + constants.BackTick + ` shows the source of where a bot came from (Metro Reviews etc etr.). If this is set to ` + constants.BackTick + constants.BackTick + `metro` + constants.BackTick + constants.BackTick + `, then ` + constants.BackTick + constants.BackTick + `list_source` + constants.BackTick + constants.BackTick + ` will be set to the metro list ID where it came from` + `
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
		Resp: Bot{},
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
	cache := state.Redis.Get(d.Context, "bc-"+name).Val()
	if cache != "" {
		return api.HttpResponse{
			Data: cache,
			Headers: map[string]string{
				"X-Popplio-Cached": "true",
			},
		}
	}

	// First check count so we can avoid expensive DB calls
	var count int64

	err := state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM bots WHERE (lower(vanity) = $1 OR bot_id = $1)", name).Scan(&count)

	if err != nil {
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if count == 0 {
		return api.DefaultResponse(http.StatusNotFound)
	}

	var bot Bot

	row, err := state.Pool.Query(d.Context, "SELECT "+botCols+" FROM bots WHERE (lower(vanity) = $1 OR bot_id = $1)", name)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	err = pgxscan.ScanOne(&bot, row)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusNotFound)
	}

	if utils.IsNone(bot.Banner.String) || !strings.HasPrefix(bot.Banner.String, "https://") {
		bot.Banner.Valid = false
		bot.Banner.String = ""
	}

	if utils.IsNone(bot.Invite.String) || !strings.HasPrefix(bot.Invite.String, "https://") {
		bot.Invite.Valid = false
		bot.Invite.String = ""
	}

	ownerUser, err := utils.GetDiscordUser(bot.Owner)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusNotFound)
	}

	bot.SubPeriodParsed = types.NewInterval(bot.SubPeriod)

	bot.MainOwner = ownerUser

	botUser, err := utils.GetDiscordUser(bot.BotID)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusNotFound)
	}

	bot.User = botUser

	bot.ResolvedAdditionalOwners = []*types.DiscordUser{}

	for _, owner := range bot.AdditionalOwners {
		ownerUser, err := utils.GetDiscordUser(owner)

		if err != nil {
			state.Logger.Error(err)
			continue
		}

		bot.ResolvedAdditionalOwners = append(bot.ResolvedAdditionalOwners, ownerUser)
	}

	var uniqueClicks int64
	err = state.Pool.QueryRow(d.Context, "SELECT cardinality(unique_clicks) AS unique_clicks FROM bots WHERE bot_id = $1", bot.BotID).Scan(&uniqueClicks)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusNotFound)
	}

	bot.UniqueClicks = uniqueClicks

	return api.HttpResponse{
		Json:      bot,
		CacheKey:  "bc-" + name,
		CacheTime: time.Minute * 3,
	}
}
