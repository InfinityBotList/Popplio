package state

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"reflect"
	"strings"
	"time"

	"popplio/config"
	"popplio/seo"
	"popplio/state/discord_dovewing"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/cache"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/disgo/sharding"
	"github.com/infinitybotlist/eureka/dovewing/dovetypes"
	hredis "github.com/infinitybotlist/eureka/hotcache/redis"
	"github.com/infinitybotlist/eureka/ratelimit"

	"github.com/go-playground/validator/v10"
	"github.com/go-playground/validator/v10/non-standard/validators"
	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/infinitybotlist/eureka/genconfig"
	"github.com/infinitybotlist/eureka/snippets"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/plutov/paypal/v4"
	"github.com/redis/go-redis/v9"
	"github.com/stripe/stripe-go/v75"
	"github.com/stripe/stripe-go/v75/webhookendpoint"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

var (
	Pool      *pgxpool.Pool
	Paypal    *paypal.Client
	Redis     *redis.Client
	Discord   bot.Client
	Logger    *zap.Logger
	Context   = context.Background()
	Validator = validator.New()

	Config           *config.Config
	StripeWebhSecret string
	StripeWebhIPList []string
	SeoMapGenerator  = &seo.MapGenerator{}

	BaseDovewingState       dovewing.BaseState
	DovewingPlatformDiscord dovewing.Platform
)

func nonVulgar(fl validator.FieldLevel) bool {
	// get the field value
	switch fl.Field().Kind() {
	case reflect.String:
		value := fl.Field().String()

		for _, v := range Config.Meta.VulgarList {
			if strings.Contains(value, v) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func Setup() {
	Validator.RegisterValidation("nonvulgar", nonVulgar)
	Validator.RegisterValidation("notblank", validators.NotBlank)
	Validator.RegisterValidation("nospaces", snippets.ValidatorNoSpaces)
	Validator.RegisterValidation("https", snippets.ValidatorIsHttps)
	Validator.RegisterValidation("httporhttps", snippets.ValidatorIsHttpOrHttps)

	genconfig.GenConfig(config.Config{})

	cfg, err := os.ReadFile("config.yaml")

	if err != nil {
		panic(err)
	}

	err = yaml.Unmarshal(cfg, &Config)

	if err != nil {
		panic(err)
	}

	err = Validator.Struct(Config)

	if err != nil {
		panic("configError: " + err.Error())
	}

	Pool, err = pgxpool.New(Context, Config.Meta.PostgresURL)

	if err != nil {
		panic(err)
	}

	rOptions, err := redis.ParseURL(Config.Meta.RedisURL.Parse())

	if err != nil {
		panic(err)
	}

	Redis = redis.NewClient(rOptions)

	Discord, err = disgo.New(Config.DiscordAuth.Token, bot.WithShardManagerConfigOpts(
		sharding.WithShardIDs(0, 1),
		sharding.WithShardCount(2),
		sharding.WithAutoScaling(true),
		sharding.WithGatewayConfigOpts(
			gateway.WithIntents(gateway.IntentGuilds, gateway.IntentGuildPresences, gateway.IntentGuildMembers),
			gateway.WithCompress(true),
		),
	),
		bot.WithCacheConfigOpts(
			cache.WithCaches(cache.FlagGuilds|cache.FlagMembers|cache.FlagPresences),
		),
		bot.WithEventListeners(&events.ListenerAdapter{
			OnGuildReady: func(event *events.GuildReady) {
				Logger.Info("Guild ready", zap.String("guildID", event.Guild.ID.String()))
			},
			OnGuildsReady: func(event *events.GuildsReady) {
				Logger.Info("All guilds ready")
			},
		}),
	)

	if err != nil {
		panic(err)
	}

	go func() {
		if err = Discord.OpenShardManager(Context); err != nil {
			slog.Error("error while connecting to gateway", slog.Any("err", err))
			return
		}

		if config.CurrentEnv == config.CurrentEnvProd {
			Discord.SetPresence(Context, gateway.WithWatchingActivity(Config.Sites.Frontend.Parse()))

			if err != nil {
				panic(err)
			}
		}
	}()

	Logger = snippets.CreateZap()

	// Load dovewing state
	BaseDovewingState = dovewing.BaseState{
		Pool:    Pool,
		Logger:  Logger,
		Context: Context,
		PlatformUserCache: hredis.RedisHotCache[dovetypes.PlatformUser]{
			Redis:  Redis,
			Prefix: "rl:",
		},
		UserExpiryTime: 8 * time.Hour,
	}

	DovewingPlatformDiscord, err = discord_dovewing.DisgoStateConfig{
		Client:         Discord,
		PreferredGuild: &Config.Servers.Main,
		BaseState:      &BaseDovewingState,
	}.New()

	if err != nil {
		panic(err)
	}

	ratelimit.SetupState(&ratelimit.RLState{
		HotCache: hredis.RedisHotCache[int]{
			Redis:  Redis,
			Prefix: "rl:",
		},
	})

	c, err := paypal.NewClient(Config.Meta.PaypalClientID.Parse(), Config.Meta.PaypalSecret.Parse(), func() string {
		if config.CurrentEnv == config.CurrentEnvStaging {
			return paypal.APIBaseSandBox
		} else {
			return paypal.APIBaseLive
		}
	}())

	if err != nil {
		Logger.Error("Paypal setup failed, disabling paypal support", zap.Error(err))
	} else {
		_, err = c.GetAccessToken(Context)

		if err != nil {
			Logger.Error("Paypal setup [oauth2] failed, disabling paypal support", zap.Error(err))
		} else {
			Paypal = c
		}
	}

	stripe.Key = Config.Meta.StripeSecretKey.Parse()

	go func() {
		// Get all current webhooks
		i := webhookendpoint.List(&stripe.WebhookEndpointListParams{})

		for i.Next() {
			// Delete it
			_, err := webhookendpoint.Del(i.WebhookEndpoint().ID, nil)

			if err != nil {
				panic(err)
			}
		}

		// Add/update stripe webhook
		params := &stripe.WebhookEndpointParams{
			URL: stripe.String(Config.Sites.API.Parse() + "/payments/stripe/webhook"),
			EnabledEvents: stripe.StringSlice([]string{
				"checkout.session.completed",
				"checkout.session.async_payment_succeeded",
				"checkout.session.async_payment_failed",
			}),
			APIVersion: stripe.String("2023-08-16"),
		}
		wh, err := webhookendpoint.New(params)

		if err != nil {
			panic(err)
		}

		StripeWebhSecret = wh.Secret

		// Next fetch the IP list
		resp, err := http.Get("https://stripe.com/files/ips/ips_webhooks.txt")

		if err != nil {
			panic(err)
		}

		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)

		if err != nil {
			panic(err)
		}

		// Split the body into lines
		StripeWebhIPList = strings.Split(string(body), "\n")

		// Remove empty lines
		for i, v := range StripeWebhIPList {
			if v == "" {
				StripeWebhIPList = append(StripeWebhIPList[:i], StripeWebhIPList[i+1:]...)
			}
		}

		Logger.Info("Stripe webhook IP allowlist:", zap.Strings("ipList", StripeWebhIPList))
	}()
}
