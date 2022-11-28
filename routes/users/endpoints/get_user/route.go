package get_user

import (
	"net/http"
	"popplio/api"
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

type User struct {
	ITag  pgtype.UUID        `db:"itag" json:"itag"`
	ID    string             `db:"user_id" json:"user_id"`
	User  *types.DiscordUser `db:"-" json:"user"` // Must be handled internally
	Staff bool               `db:"staff" json:"staff"`
	About pgtype.Text        `db:"about" json:"about"`

	VoteBanned                bool               `db:"vote_banned" json:"vote_banned"`
	Admin                     bool               `db:"admin" json:"admin"`
	HAdmin                    bool               `db:"hadmin" json:"hadmin"`
	Dev                       bool               `db:"ibldev" json:"ibldev"`
	HDev                      bool               `db:"iblhdev" json:"iblhdev"`
	StaffOnboarded            bool               `db:"staff_onboarded" json:"staff_onboarded"`
	StaffOnboardState         string             `db:"staff_onboard_state" json:"staff_onboard_state"`
	StaffOnboardLastStartTime pgtype.Timestamptz `db:"staff_onboard_last_start_time" json:"staff_onboard_last_start_time"`
	StaffOnboardMacroTime     pgtype.Timestamptz `db:"staff_onboard_macro_time" json:"staff_onboard_macro_time"`
	StaffOnboardGuild         pgtype.Text        `db:"staff_onboard_guild" json:"staff_onboard_guild"`
	Certified                 bool               `db:"certified" json:"certified"`
	Developer                 bool               `db:"developer" json:"developer"`
	UserBots                  []UserBot          `json:"user_bots"` // Must be handled internally

	ExtraLinks []types.Link `db:"extra_links" json:"extra_links"`
}

type UserBot struct {
	BotID              string             `db:"bot_id" json:"bot_id"`
	User               *types.DiscordUser `db:"-" json:"user"`
	Short              string             `db:"short" json:"short"`
	Type               string             `db:"type" json:"type"`
	Vanity             string             `db:"vanity" json:"vanity"`
	Votes              int                `db:"votes" json:"votes"`
	Shards             int                `db:"shards" json:"shards"`
	Library            string             `db:"library" json:"library"`
	InviteClick        int                `db:"invite_clicks" json:"invite_clicks"`
	Views              int                `db:"clicks" json:"clicks"`
	Servers            int                `db:"servers" json:"servers"`
	NSFW               bool               `db:"nsfw" json:"nsfw"`
	Tags               []string           `db:"tags" json:"tags"`
	OwnerID            string             `db:"owner" json:"owner_id"`
	Certified          bool               `db:"certified" json:"certified"`
	Premium            bool               `db:"premium" json:"premium"`
	AdditionalOwnerIDS []string           `db:"additional_owners" json:"additional_owner_ids"`
}

var (
	userColsArr = utils.GetCols(User{})
	userCols    = strings.Join(userColsArr, ",")

	userBotColsArr = utils.GetCols(UserBot{})
	// These are the columns of a userbot object
	userBotCols = strings.Join(userBotColsArr, ",")
)

func Docs() *docs.Doc {
	return docs.Route(&docs.Doc{
		Method:      "GET",
		Path:        "/users/{id}",
		OpId:        "get_user",
		Summary:     "Get User",
		Description: "Gets a user by id or username",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: User{},
		Tags: []string{api.CurrentTag},
	})
}

func Route(d api.RouteData, r *http.Request) {
	name := chi.URLParam(r, "id")

	if name == "" {
		d.Resp <- api.DefaultResponse(http.StatusBadRequest)
		return
	}

	if name == "undefined" {
		d.Resp <- api.HttpResponse{
			Status: http.StatusOK,
			Data:   `{"error":"false","message":"Handling known issue"}`,
		}
		return
	}

	// Check cache, this is how we can avoid hefty ratelimits
	cache := state.Redis.Get(d.Context, "uc-"+name).Val()
	if cache != "" {
		d.Resp <- api.HttpResponse{
			Data: cache,
			Headers: map[string]string{
				"X-Popplio-Cached": "true",
			},
		}
		return
	}

	var user User

	var err error

	row, err := state.Pool.Query(d.Context, "SELECT "+userCols+" FROM users WHERE user_id = $1 OR username = $1", name)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusNotFound)
		return
	}

	err = pgxscan.ScanOne(&user, row)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusNotFound)
		return
	}

	if utils.IsNone(user.About.String) {
		user.About.Valid = false
		user.About.String = ""
	}

	userObj, err := utils.GetDiscordUser(user.ID)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
		return
	}

	user.User = userObj

	userBotsRows, err := state.Pool.Query(d.Context, "SELECT "+userBotCols+" FROM bots WHERE owner = $1 OR additional_owners && $2", user.ID, []string{user.ID})

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
		return
	}

	var userBots []UserBot = []UserBot{}

	err = pgxscan.ScanAll(&userBots, userBotsRows)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
		return
	}

	parsedUserBots := []UserBot{}
	for _, bot := range userBots {
		userObj, err := utils.GetDiscordUser(bot.BotID)

		if err != nil {
			state.Logger.Error(err)
			continue
		}

		bot.User = userObj
		parsedUserBots = append(parsedUserBots, bot)
	}

	user.UserBots = parsedUserBots

	d.Resp <- api.HttpResponse{
		Json:      user,
		CacheKey:  "uc-" + name,
		CacheTime: 3 * time.Minute,
	}
}
