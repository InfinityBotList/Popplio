// Package config defines Popplio's configuration schema.
//
// The active environment is fixed at build time from the embedded
// current-env file, and Differs[T] carries every value that differs between
// staging and production so both are declared together and neither can be
// left unset. Differs[T] also carries an optional Dev override on top of
// that, for running locally against things like a personal Discord
// application without touching the real staging/prod values — see Parse.
package config

import (
	_ "embed"
	"reflect"
	"strings"

	"github.com/disgoorg/snowflake/v2"
	"github.com/go-playground/validator/v10"
)

const (
	CurrentEnvProd    = "prod"
	CurrentEnvStaging = "staging"
	CurrentEnvBeta    = "beta"
	CurrentEnvDev     = "dev"
)

//go:embed current-env
var CurrentEnv string

func init() {
	CurrentEnv = strings.TrimSpace(CurrentEnv)

	if CurrentEnv != CurrentEnvProd && CurrentEnv != CurrentEnvStaging && CurrentEnv != CurrentEnvBeta && CurrentEnv != CurrentEnvDev {
		panic("invalid environment")
	}
}

// Common struct for values that differ between staging and production
// environments, plus optional overrides for a beta deployment and local dev
// use.
//
// Staging/Prod are not tagged validate:"required" here: whether they're
// actually required depends on CurrentEnv, which a static tag can't express.
// See ValidateDiffers, registered against every instantiation of this type
// used in Config, for the real requirement.
type Differs[T any] struct {
	Staging T `yaml:"staging" comment:"Staging value"`
	Prod    T `yaml:"prod" comment:"Production value"`

	// Beta is only consulted when running with current-env set to "beta",
	// and even then only if it has been set to something other than T's
	// zero value — an unset Beta falls back to Staging, same as Dev below.
	Beta T `yaml:"beta" required:"false" comment:"Beta value, used when current-env is \"beta\"; falls back to staging when unset"`

	// Dev is only consulted when running with current-env set to "dev", and
	// even then only if it has been set to something other than T's zero
	// value — an unset Dev falls back to Staging, so config.yaml files that
	// predate this field, or that simply don't need a dev override for a
	// given key, keep working unchanged.
	Dev T `yaml:"dev" required:"false" comment:"Development value, used when current-env is \"dev\"; falls back to staging when unset"`
}

// ValidateDiffers is a struct-level validator for Differs[T]. Register it
// against every instantiation of Differs[T] actually used in Config (each
// is a distinct type, so RegisterStructValidation needs one call listing
// all of them — see state.Setup).
//
// Only requires whichever value Parse() will actually read for CurrentEnv —
// a box only ever reads one branch of Differs[T], so requiring every other
// branch too just to pass validation forced a single shared config.yaml
// across every environment for no functional reason (a prod-only box was
// rejected for a missing Staging value it would never read, and vice
// versa). "beta" and "dev" both fall back to Staging when their own value
// is unset (see Parse), so either being set satisfies them.
func ValidateDiffers(sl validator.StructLevel) {
	current := sl.Current()

	staging := current.FieldByName("Staging")
	prod := current.FieldByName("Prod")
	beta := current.FieldByName("Beta")
	dev := current.FieldByName("Dev")

	switch CurrentEnv {
	case CurrentEnvProd:
		if prod.IsZero() {
			sl.ReportError(prod.Interface(), "Prod", "Prod", "required", "")
		}
	case CurrentEnvStaging:
		if staging.IsZero() {
			sl.ReportError(staging.Interface(), "Staging", "Staging", "required", "")
		}
	case CurrentEnvBeta:
		if staging.IsZero() && beta.IsZero() {
			sl.ReportError(beta.Interface(), "Beta", "Beta", "required_without", "Staging")
		}
	case CurrentEnvDev:
		if staging.IsZero() && dev.IsZero() {
			sl.ReportError(dev.Interface(), "Dev", "Dev", "required_without", "Staging")
		}
	}
}

func (d *Differs[T]) Parse() T {
	switch CurrentEnv {
	case CurrentEnvProd:
		return d.Prod
	case CurrentEnvStaging:
		return d.Staging
	case CurrentEnvBeta:
		if !reflect.ValueOf(d.Beta).IsZero() {
			return d.Beta
		}
		return d.Staging
	case CurrentEnvDev:
		if !reflect.ValueOf(d.Dev).IsZero() {
			return d.Dev
		}
		return d.Staging
	default:
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
	Infernoplex   Infernoplex   `yaml:"infernoplex" validate:"required"`
}

type DiscordAuth struct {
	// Token is Popplio's own bot's Discord token. A dev override lets a
	// local checkout run against a personal Discord application instead of
	// the real staging/prod bot.
	Token            Differs[string] `yaml:"token" comment:"Discord bot token" validate:"required"`
	ClientID         string          `yaml:"client_id" default:"815553000470478850" comment:"Discord Client ID" validate:"required"`
	ClientSecret     string          `yaml:"client_secret" comment:"Discord Client Secret" validate:"required"`
	AllowedRedirects []string        `yaml:"allowed_redirects" default:"http://localhost:3000/auth/sauron,http://localhost:8000/auth/sauron,https://reedwhisker.infinitybots.gg/auth/sauron,https://infinitybots.gg/auth/sauron,https://botlist.site/auth/sauron,https://infinitybots.xyz/auth/sauron" validate:"required"`
}

type Sites struct {
	Frontend    Differs[string] `yaml:"frontend" default:"https://reedwhisker.infinitybots.gg" comment:"Frontend URL" validate:"required"`
	API         Differs[string] `yaml:"api" default:"https://spider.infinitybots.gg" comment:"API URL" validate:"required"`
	Panel       Differs[string] `yaml:"panel" default:"https://panel.infinitybots.gg" comment:"Panel URL" validate:"required"`
	Infernoplex Differs[string] `yaml:"infernoplex" default:"https://infernoplex.infinitybots.gg" comment:"Infernoplex URL" validate:"required"`
	Instatus    string          `yaml:"instatus" default:"https://infinity-bots.instatus.com" comment:"Instatus Status Page URL" validate:"required"`
}

type Roles struct {
	AwaitingStaff snowflake.ID            `yaml:"awaiting_staff" default:"1029058929361174678" comment:"Awaiting Staff Role" validate:"required"`
	Apps          snowflake.ID            `yaml:"apps" default:"907729844605968454" comment:"Apps Role" validate:"required"`
	CertBot       snowflake.ID            `yaml:"cert_bot" default:"759468236999491594" comment:"Certified Bot Role" validate:"required"`
	PremiumRoles  Differs[[]snowflake.ID] `yaml:"premium_roles" default:"759468236999491594" comment:"Premium Roles" validate:"required"`

	// Arcadia (staff panel/bot) roles. Ported from Arcadia's config.roles.
	// Not required in "dev" — these only matter once Arcadia's staff bot is
	// actually pointed at a real staff Discord server, which a local
	// checkout usually isn't. See requirednotdev in state.Setup.
	BotDeveloper       snowflake.ID `yaml:"bot_developer" default:"758756147313246209" comment:"Bot Developer Role" validate:"requirednotdev"`
	CertifiedDeveloper snowflake.ID `yaml:"certified_developer" default:"759468303344992266" comment:"Certified Developer Role" validate:"requirednotdev"`
	BotRole            snowflake.ID `yaml:"bot_role" default:"758652296459976715" comment:"Role given to bots joining the main server" validate:"requirednotdev"`
	BugHunters         snowflake.ID `yaml:"bug_hunters" default:"1042546603795427398" comment:"Bug Hunters Role" validate:"requirednotdev"`
	TopReviewers       snowflake.ID `yaml:"top_reviewers" default:"1239696066350420038" comment:"Top Reviewers Role" validate:"requirednotdev"`
}

type Channels struct {
	BotLogs    snowflake.ID `yaml:"bot_logs" default:"762077915499593738" comment:"Bot Logs Channel" validate:"required"`
	ModLogs    snowflake.ID `yaml:"mod_logs" default:"911907978926493716" comment:"Mod Logs Channel" validate:"required"`
	Apps       snowflake.ID `yaml:"apps" default:"1034075132030894100" comment:"Apps Channel, should be a staff only channel" validate:"required"`
	VoteLogs   snowflake.ID `yaml:"vote_logs" default:"762077981811146752" comment:"Vote Logs Channel" validate:"required"`
	BanAppeals snowflake.ID `yaml:"ban_appeals" default:"870950610692878337" comment:"Ban Appeals Channel" validate:"required"`
	AuthLogs   snowflake.ID `yaml:"auth_logs" default:"1075091440117498007" comment:"Auth Logs Channel" validate:"required"`

	// Arcadia (staff panel/bot) channels. Ported from Arcadia's config.channels.
	// Not required in "dev", see requirednotdev in state.Setup.
	TestingLounge snowflake.ID `yaml:"testing_lounge" default:"891611731699335209" comment:"Testing Lounge Channel, auto-unclaims are announced here" validate:"requirednotdev"`
	System        snowflake.ID `yaml:"system" default:"762958420277067786" comment:"System Channel" validate:"requirednotdev"`
	Uptime        snowflake.ID `yaml:"uptime" default:"1083108330442076292" comment:"Uptime Channel" validate:"requirednotdev"`
	StaffLogs     snowflake.ID `yaml:"staff_logs" default:"1186195848497999912" comment:"Staff Logs Channel" validate:"requirednotdev"`
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
	// Not required in "dev", see requirednotdev in state.Setup.
	Staff   snowflake.ID `yaml:"staff" default:"870950609291972618" comment:"Staff Server ID" validate:"requirednotdev"`
	Testing snowflake.ID `yaml:"testing" default:"870952645811134475" comment:"Testing Server ID" validate:"requirednotdev"`
}

type Meta struct {
	PostgresURL         string          `yaml:"postgres_url" default:"postgresql:///infinity" comment:"Postgres URL" validate:"required"`
	RedisURL            Differs[string] `yaml:"redis_url" default:"redis://localhost:6379" comment:"Redis URL" validate:"required"`
	Port                Differs[string] `yaml:"port" default:":8081" comment:"Port to run the server on" validate:"required"`
	VulgarList          []string        `yaml:"vulgar_list" default:"fuck,suck,shit,kill" validate:"required"`
	UrgentMentions      string          `yaml:"urgent_mentions" default:"<@&1061643797315993701>" comment:"Urgent mentions" validate:"required"`
	PaypalClientID      Differs[string] `yaml:"paypal_client_id" default:"" comment:"Paypal Client ID" validate:"required"`
	PaypalSecret        Differs[string] `yaml:"paypal_secret" default:"" comment:"Paypal Secret" validate:"required"`
	StripePublicKey     Differs[string] `yaml:"stripe_public_key" default:"" comment:"Stripe Public Key" validate:"required"`
	StripeSecretKey     Differs[string] `yaml:"stripe_secret_key" default:"" comment:"Stripe Public Key" validate:"required"`
	UptimeRobotROAPIKey string          `yaml:"uptime_robot_ro_api_key" default:"" comment:"Uptime Robot Read-Only API Key" validate:"required"`
	PopplioProxy        string          `yaml:"popplio_proxy" default:"https://gateway.nodebyte.host/proxy/discord" comment:"Popplio Proxy URL" validate:"required"`
}

// Arcadia holds the configuration keys the staff panel API and staff bot need
// that Popplio did not already carry.
//
// Keys Arcadia had that Popplio already provides are NOT duplicated here; they
// are read from the existing Popplio config instead:
//
//	arcadia database_url    -> meta.postgres_url
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
	// Token is the staff bot's own Discord token. Arcadia runs its own gateway
	// connection under its own bot identity, separate from Popplio's.
	Token      Differs[string] `yaml:"token" comment:"Staff bot Discord token. This is a SEPARATE Discord application from Popplio's" validate:"required"`
	ServerPort Differs[int]    `yaml:"server_port" default:"3010" comment:"Port the staff panel API listens on (staging 3011 / prod 3010)" validate:"required"`

	// PrefixCommands enables the legacy message commands. Slash commands are the
	// primary interface; leaving this off means the staff bot does not need the
	// privileged Message Content intent.
	PrefixCommands bool            `yaml:"prefix_commands" default:"false" comment:"Enable legacy prefix commands. Requires the privileged Message Content intent to be granted"`
	Prefix         Differs[string] `yaml:"prefix" default:"ibs!" comment:"Staff bot prefix, only used when prefix_commands is on (staging ibb! / prod ibs!)" validate:"required"`
	Owners         []snowflake.ID  `yaml:"owners" default:"510065483693817867" comment:"Bot owners, these users always hold the 'owner' staff position" validate:"required"`
	ProtectedBots  []snowflake.ID  `yaml:"protected_bots" default:"1019662370278228028" comment:"Bots that cannot be force-removed with kick enabled" validate:"required"`
	Panel          Panel           `yaml:"panel" validate:"required"`
}

type Panel struct {
	ClientID           string   `yaml:"client_id" comment:"Discord client ID of the panel login app" validate:"required"`
	ClientSecret       string   `yaml:"client_secret" comment:"Discord client secret of the panel login app" validate:"required"`
	RedirectURL        []string `yaml:"redirect_url" comment:"Allow-list of panel login redirect URLs" validate:"required"`
	PanelScope         string   `yaml:"panel_scope" comment:"Static handshake value the frontend sends" validate:"required"`
	PanelResponseScope string   `yaml:"panel_response_scope" comment:"Static handshake value the frontend expects back" validate:"required"`
}

type Infernoplex struct {
	ClientID     string          `yaml:"client_id" comment:"Infernoplex bot Discord client ID" validate:"required"`
	ClientSecret string          `yaml:"client_secret" comment:"Infernoplex bot Discord client secret" validate:"required"`
	Prefix       Differs[string] `yaml:"prefix" default:"inf!" comment:"Infernoplex bot prefix" validate:"required"`
	ServerPort   Differs[int]    `yaml:"server_port" default:"3012" comment:"Port the Infernoplex bot API listens on" validate:"required"`
	Token        Differs[string] `yaml:"token" comment:"Infernoplex bot Discord token" validate:"required"`
}

type Naevis struct {
	ClientID string          `yaml:"client_id" comment:"Naevis bot Discord client ID" validate:"required"`
	Token    Differs[string] `yaml:"token" comment:"Naevis bot Discord token" validate:"required"`
	Prefix   Differs[string] `yaml:"prefix" default:"nae!" comment:"Naevis bot prefix" validate:"required"`
}
