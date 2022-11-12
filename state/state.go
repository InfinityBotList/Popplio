package state

import (
	"context"
	"flag"
	"os"

	"github.com/bwmarrin/discordgo"
	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/joho/godotenv"
)

var (
	Pool        *pgxpool.Pool
	BackupsPool *pgxpool.Pool
	Redis       *redis.Client
	Discord     *discordgo.Session
	Context     = context.Background()
)

// This should be the only init function, sets global state
func init() {
	godotenv.Load()

	var connUrl string
	var backupsConnUrl string
	var redisUrl string

	flag.StringVar(&connUrl, "db", "postgresql:///infinity", "Database connection URL")
	flag.StringVar(&backupsConnUrl, "backups-db", "postgresql:///backups", "Database connection URL for backups")
	flag.StringVar(&redisUrl, "redis", "redis://localhost:6379", "Redis connection URL")
	flag.Parse()

	var err error
	Pool, err = pgxpool.Connect(Context, connUrl)

	if err != nil {
		panic(err)
	}

	BackupsPool, err = pgxpool.Connect(Context, backupsConnUrl)

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

}
