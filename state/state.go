package state

import (
	"context"
	"flag"
	"os"
	"reflect"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/go-playground/validator/v10"
	"github.com/go-playground/validator/v10/non-standard/validators"
	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	Pool        *pgxpool.Pool
	BackupsPool *pgxpool.Pool
	Redis       *redis.Client
	Discord     *discordgo.Session
	Logger      *zap.SugaredLogger
	Context     = context.Background()
	Validator   = validator.New()

	Migration = false
)

var vulgarList = []string{
	"fuck",
	"shit",
	"suck",
	"kill",
}

func nonVulgar(fl validator.FieldLevel) bool {
	// get the field value
	switch fl.Field().Kind() {
	case reflect.String:
		value := fl.Field().String()

		for _, v := range vulgarList {
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

// This should be the only init function, sets global state
func init() {
	godotenv.Load()

	if os.Getenv("IN_POPPLIO") != "true" {
		return
	}

	var connUrl string
	var backupsConnUrl string
	var redisUrl string

	flag.StringVar(&connUrl, "db", "postgresql:///infinity", "Database connection URL")
	flag.StringVar(&backupsConnUrl, "backups-db", "postgresql:///backups", "Database connection URL for backups")
	flag.StringVar(&redisUrl, "redis", "redis://localhost:6379", "Redis connection URL")
	flag.Parse()

	var err error
	Pool, err = pgxpool.New(Context, connUrl)

	if err != nil {
		panic(err)
	}

	BackupsPool, err = pgxpool.New(Context, backupsConnUrl)

	if err != nil {
		panic(err)
	}

	rOptions, err := redis.ParseURL(redisUrl)

	if err != nil {
		panic(err)
	}

	Redis = redis.NewClient(rOptions)

	Discord, err = discordgo.New("Bot " + os.Getenv("DISCORD_TOKEN"))

	if err != nil {
		panic(err)
	}

	Discord.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentGuildPresences | discordgo.IntentsGuildMembers

	err = Discord.Open()
	if err != nil {
		panic(err)
	}

	// lumberjack.Logger is already safe for concurrent use, so we don't need to
	// lock it.
	w := zapcore.AddSync(&lumberjack.Logger{
		Filename:   "/var/log/popplio.log",
		MaxSize:    10, // megabytes
		MaxBackups: 3,
		MaxAge:     28, // days
	})

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		w,
		zap.DebugLevel,
	)

	Logger = zap.New(core).Sugar()

	Validator.RegisterValidation("nonvulgar", nonVulgar)
	Validator.RegisterValidation("notblank", validators.NotBlank)
	Validator.RegisterValidation("nospaces", noSpaces)
	Validator.RegisterValidation("notpresent", notpresent)
}
