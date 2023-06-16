package state

import (
	"context"
	"io"
	"net/http"
	"os"
	"reflect"
	"strings"

	"popplio/config"

	"github.com/bwmarrin/discordgo"
	"github.com/go-playground/validator/v10"
	"github.com/go-playground/validator/v10/non-standard/validators"
	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/infinitybotlist/eureka/genconfig"
	"github.com/infinitybotlist/eureka/snippets"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/plutov/paypal/v4"
	"github.com/redis/go-redis/v9"
	"github.com/stripe/stripe-go/v74"
	"github.com/stripe/stripe-go/v74/webhookendpoint"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

var (
	Pool      *pgxpool.Pool
	Paypal    *paypal.Client
	Redis     *redis.Client
	Discord   *discordgo.Session
	Logger    *zap.SugaredLogger
	Context   = context.Background()
	Validator = validator.New()

	Config           *config.Config
	StripeWebhSecret string
	StripeWebhIPList []string
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

func updateDb(u *dovewing.PlatformUser) error {
	if u.Bot {
		_, err := Pool.Exec(Context, "UPDATE bots SET queue_name = $1, queue_avatar = $2 WHERE bot_id = $3", u.Username, u.Avatar, u.ID)

		if err != nil {
			return err
		}
	}

	return nil
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

	rOptions, err := redis.ParseURL(Config.Meta.RedisURL)

	if err != nil {
		panic(err)
	}

	Redis = redis.NewClient(rOptions)

	Discord, err = discordgo.New("Bot " + Config.DiscordAuth.Token)

	if err != nil {
		panic(err)
	}

	Discord.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentGuildPresences | discordgo.IntentsGuildMembers

	go func() {
		err = Discord.Open()
		if err != nil {
			panic(err)
		}

		err = Discord.UpdateWatchStatus(0, Config.Sites.Frontend)

		if err != nil {
			panic(err)
		}
	}()

	Logger = snippets.CreateZap()

	// Load dovewing state
	dovewing.SetState(&dovewing.State{
		Discord: &dovewing.DiscordState{
			Session:     Discord,
			UpdateCache: updateDb,
		},
		Pool:           Pool,
		Logger:         Logger,
		PreferredGuild: Config.Servers.Main,
		Context:        Context,
		Redis:          Redis,
	})

	c, err := paypal.NewClient(Config.Meta.PaypalClientID, Config.Meta.PaypalSecret, func() string {
		if Config.Meta.PaypalUseSandbox {
			return paypal.APIBaseSandBox
		} else {
			return paypal.APIBaseLive
		}
	}())

	if err != nil {
		panic(err)
	}

	_, err = c.GetAccessToken(Context)

	if err != nil {
		panic(err)
	}

	Paypal = c

	stripe.Key = Config.Meta.StripeSecretKey

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
			URL: stripe.String(Config.Sites.API + "/payments/stripe/webhook"),
			EnabledEvents: stripe.StringSlice([]string{
				"checkout.session.completed",
				"checkout.session.async_payment_succeeded",
				"checkout.session.async_payment_failed",
			}),
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

		Logger.Info("Stripe webhook IP allowlist:", StripeWebhIPList)
	}()
}
