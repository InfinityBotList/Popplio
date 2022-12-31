package state

import (
	"context"
	"flag"
	"io"
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
	"gopkg.in/natefinch/lumberjack.v2"
	"gopkg.in/yaml.v3"
)

var (
	Pool      *pgxpool.Pool
	Redis     *redis.Client
	Discord   *discordgo.Session
	Logger    *zap.SugaredLogger
	Context   = context.Background()
	Validator = validator.New()

	Migration = false
	Config    *config.Config
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

func Setup(cfg []byte) {
	Validator.RegisterValidation("nonvulgar", nonVulgar)
	Validator.RegisterValidation("notblank", validators.NotBlank)
	Validator.RegisterValidation("nospaces", noSpaces)
	Validator.RegisterValidation("notpresent", notpresent)

	err := yaml.Unmarshal(cfg, &Config)

	if err != nil {
		panic(err)
	}

	err = Validator.Struct(Config)

	if err != nil {
		panic("configError: " + err.Error())
	}

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

	Pool, err = pgxpool.New(Context, connUrl)

	if err != nil {
		panic(err)
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

	err = Discord.Open()
	if err != nil {
		panic(err)
	}

	// lumberjack.Logger is already safe for concurrent use, so we don't need to
	// lock it.
	w := zapcore.AddSync(io.MultiWriter(
		&lumberjack.Logger{
			Filename:   "/var/log/popplio.log",
			MaxSize:    10, // megabytes
			MaxBackups: 3,
			MaxAge:     28, // days
		},
		os.Stdout,
	))

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		w,
		zap.DebugLevel,
	)

	Logger = zap.New(core).Sugar()
}
