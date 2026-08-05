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
	"github.com/disgoorg/snowflake/v2"
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

	// Like "required", but not enforced in "dev" — for fields (Arcadia's
	// staff server channels/roles/IDs) that only matter once Arcadia is
	// actually pointed at a real staff Discord server, which a local
	// checkout usually isn't.
	Validator.RegisterValidation("requirednotdev", func(fl validator.FieldLevel) bool {
		if config.CurrentEnv == config.CurrentEnvDev {
			return true
		}
		return !fl.Field().IsZero()
	})

	// One call per instantiation of config.Differs[T] actually used in
	// Config, generics make each a distinct type as far as the validator
	// is concerned.
	Validator.RegisterStructValidation(
		config.ValidateDiffers,
		config.Differs[string]{},
		config.Differs[int]{},
		config.Differs[[]snowflake.ID]{},
	)

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

	Discord, err = disgo.New(Config.DiscordAuth.Token.Parse(),
		bot.WithRestClientConfigOpts(ProxyRestOpts(Config.DiscordAuth.Token.Parse())...),
		bot.WithShardManagerConfigOpts(
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
				Logger.Info("All guilds ready", zap.Int("shardID", event.ShardID()))

				// Popplio runs sharded (OpenShardManager), not a single
				// gateway (OpenGateway) — SetPresence only ever checks
				// Discord's single-gateway field and unconditionally
				// returns "no gateway configured" on a sharded bot, no
				// matter how ready the shards are. GuildsReady also fires
				// once per shard, not once globally, so set presence per
				// shard via SetPresenceForShard using the event's shard ID.
				if presenceErr := Discord.SetPresenceForShard(Context, event.ShardID(), gateway.WithWatchingActivity(Config.Sites.Frontend.Parse())); presenceErr != nil {
					Logger.Error("error while setting presence", zap.Error(presenceErr), zap.Int("shardID", event.ShardID()))
				}
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
		// Only real production talks to Paypal's live API — staging and dev
		// both use the sandbox, since both use test keys.
		if config.CurrentEnv == config.CurrentEnvProd {
			return paypal.APIBaseLive
		}
		return paypal.APIBaseSandBox
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
		// A transient failure here (Stripe API hiccup, network blip) should
		// disable Stripe webhook support, not take down the whole process —
		// same graceful-degradation treatment as the Paypal setup above.

		// Get all current webhooks
		i := webhookendpoint.List(&stripe.WebhookEndpointListParams{})

		for i.Next() {
			// Delete it
			_, err := webhookendpoint.Del(i.WebhookEndpoint().ID, nil)

			if err != nil {
				Logger.Error("Stripe webhook setup failed [delete existing], disabling stripe webhook support", zap.Error(err))
				return
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
			Logger.Error("Stripe webhook setup failed [create], disabling stripe webhook support", zap.Error(err))
			return
		}

		StripeWebhSecret = wh.Secret

		// Next fetch the IP list
		resp, err := http.Get("https://stripe.com/files/ips/ips_webhooks.txt")

		if err != nil {
			Logger.Error("Stripe webhook setup failed [fetch IP list], disabling stripe webhook support", zap.Error(err))
			return
		}

		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)

		if err != nil {
			Logger.Error("Stripe webhook setup failed [read IP list], disabling stripe webhook support", zap.Error(err))
			return
		}

		// Split the body into lines, dropping empty ones. Built into a new
		// slice rather than filtered in place — mutating StripeWebhIPList
		// while ranging over its original indices would skip the element
		// that shifts into each removed slot.
		lines := strings.Split(string(body), "\n")
		ipList := make([]string, 0, len(lines))
		for _, v := range lines {
			if v != "" {
				ipList = append(ipList, v)
			}
		}
		StripeWebhIPList = ipList

		Logger.Info("Stripe webhook IP allowlist:", zap.Strings("ipList", StripeWebhIPList))
	}()
}
