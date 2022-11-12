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

const (
	NotFound         = "{\"message\":\"Slow down, bucko! We couldn't find this resource *anywhere*!\",\"error\":true}"
	NotFoundPage     = "{\"message\":\"Slow down, bucko! You got the path wrong or something but this endpoint doesn't exist!\",\"error\":true}"
	BadRequest       = "{\"message\":\"Slow down, bucko! You're doing something illegal!!!\",\"error\":true}"
	BadRequestStats  = "{\"message\":\"Slow down, bucko! You're not posting stats correctly. Hint: try posting stats as integers and not as strings?\",\"error\":true}"
	Unauthorized     = "{\"message\":\"Slow down, bucko! You're not authorized to do this or did you forget a API token somewhere?\",\"error\":true}"
	InternalError    = "{\"message\":\"Slow down, bucko! Something went wrong on our end!\",\"error\":true}"
	MethodNotAllowed = "{\"message\":\"Slow down, bucko! That method is not allowed for this endpoint!!!\",\"error\":true}"
	NotApproved      = "{\"message\":\"Woah there, your bot needs to be approved. Calling the police right now over this infraction!\",\"error\":true}"
	VoteBanned       = "{\"message\":\"Slow down, bucko! Either you or this bot is banned from voting right now!\",\"error\":true}"
	Success          = "{\"message\":\"Success!\",\"error\":false}"
	BackTick         = "`"
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
