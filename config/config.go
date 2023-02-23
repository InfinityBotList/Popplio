package config

type Config struct {
	DiscordAuth     DiscordAuth     `yaml:"discord_auth" validate:"required"`
	HighSecurityCtx HighSecurityCtx `yaml:"high_security_ctx" validate:"required"`
	Sites           Sites           `yaml:"sites" validate:"required"`
	Metro           Metro           `yaml:"metro" validate:"required"`
	Channels        Channels        `yaml:"channels" validate:"required"`
	Roles           Roles           `yaml:"roles" validate:"required"`
	JAPI            JAPI            `yaml:"japi" validate:"required"`
	Notifications   Notifications   `yaml:"notifications" validate:"required"`
	Servers         Servers         `yaml:"servers" validate:"required"`
	Meta            Meta            `yaml:"meta" validate:"required"`
	Hcaptcha        Hcaptcha        `yaml:"hcaptcha" validate:"required"`
}

type Hcaptcha struct {
	Secret string `yaml:"secret" comment:"Hcaptcha Secret" validate:"required"`
}

type DiscordAuth struct {
	Token            string   `yaml:"token" comment:"Discord bot token" validate:"required"`
	ClientID         string   `yaml:"client_id" default:"815553000470478850" comment:"Discord Client ID" validate:"required"`
	ClientSecret     string   `yaml:"client_secret" comment:"Discord Client Secret" validate:"required"`
	AllowedRedirects []string `yaml:"allowed_redirects" default:"http://localhost:3000/sauron,https://reedwhisker.infinitybots.gg/sauron" validate:"required"`
}

type HighSecurityCtx struct {
	Disabled     bool   `yaml:"disabled" default:"false" comment:"Disable high security context"`
	ClientID     string `yaml:"client_id" default:"1046812407743385611" comment:"Client ID to use for high security operations" validate:"required"`
	ClientSecret string `yaml:"client_secret" comment:"Client Secret to use for high security operations" validate:"required"`
	RedirectURL  string `yaml:"redirect_url" default:"https://spider.infinitybotlist.com/cosmog" comment:"Redirect URL to use for high security operations" validate:"required"`
}

type Sites struct {
	Frontend string `yaml:"frontend" default:"https://reedwhisker.infinitybots.gg" comment:"Frontend URL" validate:"required"`
	API      string `yaml:"api" default:"https://spider.infinitybots.gg" comment:"API URL" validate:"required"`
	AppSite  string `yaml:"app_site" default:"https://ptb.botlist.app" comment:"App Site" validate:"required"`
}

type Metro struct {
	ListID    string `yaml:"list_id" default:"02a6a5d1-8cd4-4847-a3ed-a32c88bd805b" comment:"Metro List ID" validate:"required"`
	SecretKey string `yaml:"secret_key" comment:"Metro Secret Key" required:"false"`
}

type Roles struct {
	AwaitingStaff string `yaml:"awaiting_staff" default:"1029058929361174678" comment:"Awaiting Staff Role" validate:"required"`
}

type Channels struct {
	BotLogs    string `yaml:"bot_logs" default:"762077915499593738" comment:"Bot Logs Channel" validate:"required"`
	ModLogs    string `yaml:"mod_logs" default:"911907978926493716" comment:"Mod Logs Channel" validate:"required"`
	Apps       string `yaml:"apps" default:"1034075132030894100" comment:"Apps Channel, should be a staff only channel" validate:"required"`
	VoteLogs   string `yaml:"vote_logs" default:"762077981811146752" comment:"Vote Logs Channel" validate:"required"`
	BanAppeals string `yaml:"ban_appeals" default:"870950610692878337" comment:"Ban Appeals Channel" validate:"required"`
	AuthLogs   string `yaml:"auth_logs" default:"1075091440117498007" comment:"Auth Logs Channel" validate:"required"`
}

type JAPI struct {
	Key string `yaml:"key" comment:"JAPI Key. Get it from https://japi.rest" validate:"required"`
}

type Notifications struct {
	VapidPublicKey  string `yaml:"vapid_public_key" default:"BIdUNSqYzqVjbdJhn8WK6SDYDVj85mKtctrEgj14KkjxIMerxQ9wywvvxECkuP8rL3s8zDgZSE9HSqW1wmhVPM8" comment:"Vapid Public Key (https://www.stephane-quantin.com/en/tools/generators/vapid-keys)" validate:"required"`
	VapidPrivateKey string `yaml:"vapid_private_key" comment:"Vapid Private Key (https://www.stephane-quantin.com/en/tools/generators/vapid-keys)" validate:"required"`
}

type Servers struct {
	Main string `yaml:"main" default:"758641373074423808" comment:"Main Server ID" validate:"required"`
}

type Meta struct {
	PostgresURL     string   `yaml:"postgres_url" default:"postgresql:///infinity" comment:"Postgres URL" validate:"required"`
	RedisURL        string   `yaml:"redis_url" default:"redis://localhost:6379" comment:"Redis URL" validate:"required"`
	Port            string   `yaml:"port" default:":8081" comment:"Port to run the server on" validate:"required"`
	VulgarList      []string `yaml:"vulgar_list" default:"fuck,suck,shit,kill" validate:"required"`
	AllowedHTMLTags []string `yaml:"allowed_html_tags" default:"a,i,button,span,img,video,iframe,style,span,p,br,center,div,h1,h2,h3,h4,h5,section,article,lang,code,pre,strong,em" validate:"required"`
	CliNonce        string   `yaml:"cli_nonce" default:"" comment:"CLI Nonce" validate:"required"`
	UrgentMentions  string   `yaml:"urgent_mentions" default:"<@&1061643797315993701>" comment:"Urgent mentions" validate:"required"`
}
