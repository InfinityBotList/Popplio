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
}

type Channels struct {
	BotLogs    snowflake.ID `yaml:"bot_logs" default:"762077915499593738" comment:"Bot Logs Channel" validate:"required"`
	ModLogs    snowflake.ID `yaml:"mod_logs" default:"911907978926493716" comment:"Mod Logs Channel" validate:"required"`
	Apps       snowflake.ID `yaml:"apps" default:"1034075132030894100" comment:"Apps Channel, should be a staff only channel" validate:"required"`
	VoteLogs   snowflake.ID `yaml:"vote_logs" default:"762077981811146752" comment:"Vote Logs Channel" validate:"required"`
	BanAppeals snowflake.ID `yaml:"ban_appeals" default:"870950610692878337" comment:"Ban Appeals Channel" validate:"required"`
	AuthLogs   snowflake.ID `yaml:"auth_logs" default:"1075091440117498007" comment:"Auth Logs Channel" validate:"required"`
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
