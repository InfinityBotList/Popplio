package state

import (
	"context"
	"flag"
	"os"
	"reflect"
	"strings"

	"popplio/cmd/genconfig"
	"popplio/config"

	"github.com/bwmarrin/discordgo"
	"github.com/go-playground/validator/v10"
	"github.com/go-playground/validator/v10/non-standard/validators"
	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v3"
)

var (
	Pool      *pgxpool.Pool
	Redis     *redis.Client
	Discord   *discordgo.Session
	Logger    *zap.SugaredLogger
	Context   = context.Background()
	Validator = validator.New()

	Config *config.Config
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
		panic("not a string")
	}
}

func noSpaces(fl validator.FieldLevel) bool {
	// get the field value
	switch fl.Field().Kind() {
	case reflect.String:
		value := fl.Field().String()

		if strings.Contains(value, " ") {
			return false
		}
		return true
	default:
		panic("not a string")
	}
}

func notpresent(fl validator.FieldLevel) bool {
	// get the field value
	switch fl.Field().Kind() {
	case reflect.String:
		value := fl.Field().String()

		if value == "" {
			return false
		}
		return true
	default:
		panic("not a string")
	}
}

func Setup() {
	Validator.RegisterValidation("nonvulgar", nonVulgar)
	Validator.RegisterValidation("notblank", validators.NotBlank)
	Validator.RegisterValidation("nospaces", noSpaces)
	Validator.RegisterValidation("notpresent", notpresent)

	var connUrl string
	var redisUrl string
	var cmdStr string

	flag.StringVar(&connUrl, "db", "postgresql:///infinity", "Database connection URL")
	flag.StringVar(&redisUrl, "redis", "redis://localhost:6379", "Redis connection URL")
	flag.StringVar(&cmdStr, "cmd", "", "Command to run")
	flag.Parse()

	if cmdStr != "" {
		if cmdStr == "genconfig" {
			genconfig.GenConfig()
			os.Exit(0)
		}
	}

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

	Pool, err = pgxpool.New(Context, connUrl)

	if err != nil {
		panic(err)
	}

	// Create the cache tables in db
	_, err = Pool.Exec(Context, `
		CREATE TABLE IF NOT EXISTS internal_user_cache (
			id TEXT PRIMARY KEY,
			username TEXT NOT NULL,
			discriminator TEXT NOT NULL,
			avatar TEXT NOT NULL,
			bot BOOLEAN NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			last_updated TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)

	if err != nil {
		panic("User cache table creation error: " + err.Error())
	}

	rOptions, err := redis.ParseURL(redisUrl)

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
	}()

	w := zapcore.AddSync(os.Stdout)

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		w,
		zap.DebugLevel,
	)

	Logger = zap.New(core).Sugar()
}
