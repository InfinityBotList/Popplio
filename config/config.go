package config

import (
	_ "embed"
	"strings"

	"github.com/disgoorg/snowflake/v2"
)

const (
	CurrentEnvProd    = "prod"
	CurrentEnvStaging = "staging"
)

//go:embed current-env
var CurrentEnv string

func init() {
	CurrentEnv = strings.TrimSpace(CurrentEnv)

	if CurrentEnv != CurrentEnvProd && CurrentEnv != CurrentEnvStaging {
		panic("invalid environment")
	}
}

// Common struct for values that differ between staging and production environments
type Differs[T any] struct {
	Staging T `yaml:"staging" comment:"Staging value" validate:"required"`
	Prod    T `yaml:"prod" comment:"Production value" validate:"required"`
}

func (d *Differs[T]) Parse() T {
	if CurrentEnv == CurrentEnvProd {
		return d.Prod
	} else if CurrentEnv == CurrentEnvStaging {
		return d.Staging
	} else {
		panic("invalid environment")
	}
}

func (d *Differs[T]) Production() T {
	return d.Prod
}

type Config struct {
	DiscordAuth   DiscordAuth   `yaml:"discord_auth" validate:"required"`
	Sites         Sites         `yaml:"sites" validate:"required"`
	Channels      Channels      `yaml:"channels" validate:"required"`
	Roles         Roles         `yaml:"roles" validate:"required"`
	JAPI          JAPI          `yaml:"japi" validate:"required"`
	Notifications Notifications `yaml:"notifications" validate:"required"`
	Servers       Servers       `yaml:"servers" validate:"required"`
	Meta          Meta          `yaml:"meta" validate:"required"`
	Arcadia       Arcadia       `yaml:"arcadia" validate:"required"`
}

type DiscordAuth struct {
	Token            string   `yaml:"token" comment:"Discord bot token" validate:"required"`
	ClientID         string   `yaml:"client_id" default:"815553000470478850" comment:"Discord Client ID" validate:"required"`
	ClientSecret     string   `yaml:"client_secret" comment:"Discord Client Secret" validate:"required"`
	AllowedRedirects []string `yaml:"allowed_redirects" default:"http://localhost:3000/auth/sauron,http://localhost:8000/auth/sauron,https://reedwhisker.infinitybots.gg/auth/sauron,https://infinitybots.gg/auth/sauron,https://botlist.site/auth/sauron,https://infinitybots.xyz/auth/sauron" validate:"required"`
}

type Sites struct {
	Frontend    Differs[string] `yaml:"frontend" default:"https://reedwhisker.infinitybots.gg" comment:"Frontend URL" validate:"required"`
	API         Differs[string] `yaml:"api" default:"https://spider.infinitybots.gg" comment:"API URL" validate:"required"`
	Panel       Differs[string] `yaml:"panel" default:"https://panel.infinitybots.gg" comment:"Panel URL" validate:"required"`
	Infernoplex Differs[string] `yaml:"infernoplex" default:"https://infernoplex.infinitybots.gg" comment:"Infernoplex URL" validate:"required"`
	CDN         string          `yaml:"cdn" default:"https://cdn.infinitybots.gg" comment:"CDN URL" validate:"required"`
	Instatus    string          `yaml:"instatus" default:"https://infinity-bots.instatus.com" comment:"Instatus Status Page URL" validate:"required"`
}

type Roles struct {
	AwaitingStaff snowflake.ID            `yaml:"awaiting_staff" default:"1029058929361174678" comment:"Awaiting Staff Role" validate:"required"`
	Apps          snowflake.ID            `yaml:"apps" default:"907729844605968454" comment:"Apps Role" validate:"required"`
	CertBot       snowflake.ID            `yaml:"cert_bot" default:"759468236999491594" comment:"Certified Bot Role" validate:"required"`
	PremiumRoles  Differs[[]snowflake.ID] `yaml:"premium_roles" default:"759468236999491594" comment:"Premium Roles" validate:"required"`

	// Arcadia (staff panel/bot) roles. Ported from Arcadia's config.roles.
	BotDeveloper       snowflake.ID `yaml:"bot_developer" default:"758756147313246209" comment:"Bot Developer Role" validate:"required"`
	CertifiedDeveloper snowflake.ID `yaml:"certified_developer" default:"759468303344992266" comment:"Certified Developer Role" validate:"required"`
	BotRole            snowflake.ID `yaml:"bot_role" default:"758652296459976715" comment:"Role given to bots joining the main server" validate:"required"`
	BugHunters         snowflake.ID `yaml:"bug_hunters" default:"1042546603795427398" comment:"Bug Hunters Role" validate:"required"`
	TopReviewers       snowflake.ID `yaml:"top_reviewers" default:"1239696066350420038" comment:"Top Reviewers Role" validate:"required"`
}

type Channels struct {
	BotLogs    snowflake.ID `yaml:"bot_logs" default:"762077915499593738" comment:"Bot Logs Channel" validate:"required"`
	ModLogs    snowflake.ID `yaml:"mod_logs" default:"911907978926493716" comment:"Mod Logs Channel" validate:"required"`
	Apps       snowflake.ID `yaml:"apps" default:"1034075132030894100" comment:"Apps Channel, should be a staff only channel" validate:"required"`
	VoteLogs   snowflake.ID `yaml:"vote_logs" default:"762077981811146752" comment:"Vote Logs Channel" validate:"required"`
	BanAppeals snowflake.ID `yaml:"ban_appeals" default:"870950610692878337" comment:"Ban Appeals Channel" validate:"required"`
	AuthLogs   snowflake.ID `yaml:"auth_logs" default:"1075091440117498007" comment:"Auth Logs Channel" validate:"required"`

	// Arcadia (staff panel/bot) channels. Ported from Arcadia's config.channels.
	TestingLounge snowflake.ID `yaml:"testing_lounge" default:"891611731699335209" comment:"Testing Lounge Channel, auto-unclaims are announced here" validate:"required"`
	System        snowflake.ID `yaml:"system" default:"762958420277067786" comment:"System Channel" validate:"required"`
	Uptime        snowflake.ID `yaml:"uptime" default:"1083108330442076292" comment:"Uptime Channel" validate:"required"`
	StaffLogs     snowflake.ID `yaml:"staff_logs" default:"1186195848497999912" comment:"Staff Logs Channel" validate:"required"`
}

type JAPI struct {
	Key string `yaml:"key" comment:"JAPI Key. Get it from https://japi.rest" validate:"required"`
}

type Notifications struct {
	VapidPublicKey  string `yaml:"vapid_public_key" default:"BIdUNSqYzqVjbdJhn8WK6SDYDVj85mKtctrEgj14KkjxIMerxQ9wywvvxECkuP8rL3s8zDgZSE9HSqW1wmhVPM8" comment:"Vapid Public Key (https://www.stephane-quantin.com/en/tools/generators/vapid-keys)" validate:"required"`
	VapidPrivateKey string `yaml:"vapid_private_key" comment:"Vapid Private Key (https://www.stephane-quantin.com/en/tools/generators/vapid-keys)" validate:"required"`
}

type Servers struct {
	Main snowflake.ID `yaml:"main" default:"758641373074423808" comment:"Main Server ID" validate:"required"`

	// Arcadia (staff panel/bot) servers. Ported from Arcadia's config.servers.
	Staff   snowflake.ID `yaml:"staff" default:"870950609291972618" comment:"Staff Server ID" validate:"required"`
	Testing snowflake.ID `yaml:"testing" default:"870952645811134475" comment:"Testing Server ID" validate:"required"`
}

type Meta struct {
	PostgresURL         string          `yaml:"postgres_url" default:"postgresql:///infinity" comment:"Postgres URL" validate:"required"`
	RedisURL            Differs[string] `yaml:"redis_url" default:"redis://localhost:6379" comment:"Redis URL" validate:"required"`
	Port                Differs[string] `yaml:"port" default:":8081" comment:"Port to run the server on" validate:"required"`
	CDNPath             string          `yaml:"cdn_path" default:"/silverpelt/cdn/ibl" comment:"CDN Path" validate:"required"`
	VulgarList          []string        `yaml:"vulgar_list" default:"fuck,suck,shit,kill" validate:"required"`
	UrgentMentions      string          `yaml:"urgent_mentions" default:"<@&1061643797315993701>" comment:"Urgent mentions" validate:"required"`
	PaypalClientID      Differs[string] `yaml:"paypal_client_id" default:"" comment:"Paypal Client ID" validate:"required"`
	PaypalSecret        Differs[string] `yaml:"paypal_secret" default:"" comment:"Paypal Secret" validate:"required"`
	StripePublicKey     Differs[string] `yaml:"stripe_public_key" default:"" comment:"Stripe Public Key" validate:"required"`
	StripeSecretKey     Differs[string] `yaml:"stripe_secret_key" default:"" comment:"Stripe Public Key" validate:"required"`
	UptimeRobotROAPIKey string          `yaml:"uptime_robot_ro_api_key" default:"" comment:"Uptime Robot Read-Only API Key" validate:"required"`
	PopplioProxy        string          `yaml:"popplio_proxy" default:"http://127.0.0.1:3219" comment:"Popplio Proxy URL" validate:"required"`
}

// Arcadia holds the configuration keys the staff panel API and staff bot need
// that Popplio did not already carry.
//
// Keys Arcadia had that Popplio already provides are NOT duplicated here; they
// are read from the existing Popplio config instead:
//
//	arcadia database_url    -> meta.postgres_url
//	arcadia token           -> discord_auth.token
//	arcadia frontend_url    -> sites.frontend
//	arcadia infernoplex_url -> sites.infernoplex
//	arcadia popplio_url     -> sites.api
//	arcadia cdn_url         -> sites.cdn
//	arcadia proxy_url       -> meta.popplio_proxy
//	arcadia japi_key        -> japi.key
//	arcadia servers.*       -> servers.{main,staff,testing}
//	arcadia roles.*         -> roles.*
//	arcadia channels.*      -> channels.*
type Arcadia struct {
	ServerPort      Differs[int]    `yaml:"server_port" default:"3010" comment:"Port the staff panel API listens on (staging 3011 / prod 3010)" validate:"required"`
	Prefix          Differs[string] `yaml:"prefix" default:"ibs!" comment:"Staff bot prefix (staging ibb! / prod ibs!)" validate:"required"`
	HTMLSanitizeURL string          `yaml:"htmlsanitize_url" default:"https://hs.infinitybots.gg/" comment:"HTML Sanitizer URL" validate:"required"`
	BorealisURL     string          `yaml:"borealis_url" default:"http://localhost:2837" comment:"Borealis URL, used to add approved bots to a cache server" validate:"required"`
	Owners          []snowflake.ID  `yaml:"owners" default:"510065483693817867" comment:"Bot owners, these users always hold the 'owner' staff position" validate:"required"`
	ProtectedBots   []snowflake.ID  `yaml:"protected_bots" default:"1019662370278228028" comment:"Bots that cannot be force-removed with kick enabled" validate:"required"`
	AssetCleanerDry bool            `yaml:"asset_cleaner_dry_run" default:"false" comment:"Log what the asset_cleaner task would delete without deleting it"`
	Panel           Panel           `yaml:"panel" validate:"required"`
}

type Panel struct {
	ClientID           string                           `yaml:"client_id" comment:"Discord client ID of the panel login app" validate:"required"`
	ClientSecret       string                           `yaml:"client_secret" comment:"Discord client secret of the panel login app" validate:"required"`
	RedirectURL        []string                         `yaml:"redirect_url" comment:"Allow-list of panel login redirect URLs" validate:"required"`
	CdnScopes          Differs[map[string]CdnScopeData] `yaml:"cdn_scopes" comment:"CDN scopes, keyed by scope name" validate:"required"`
	MainScope          string                           `yaml:"main_scope" default:"ibl@main" comment:"The main CDN scope" validate:"required"`
	PanelScope         string                           `yaml:"panel_scope" comment:"Static handshake value the frontend sends" validate:"required"`
	PanelResponseScope string                           `yaml:"panel_response_scope" comment:"Static handshake value the frontend expects back" validate:"required"`
}

type CdnScopeData struct {
	Path       string `yaml:"path" json:"path" comment:"Path on local disk"`
	ExposedURL string `yaml:"exposed_url" json:"exposed_url" comment:"Publicly exposed URL for this scope"`
}
