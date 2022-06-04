package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	integrase "github.com/MetroReviews/metro-integrase/lib"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	mongoUrl   = "mongodb://127.0.0.1:27017/infinity" // Is already public in 10 other places so
	docsSite   = "https://docs.botlist.site"
	mainSite   = "https://infinitybotlist.com"
	statusPage = "https://status.botlist.site"
	apiBot     = "https://discord.com/api/oauth2/authorize?client_id=818419115068751892&permissions=140898593856&scope=bot%20applications.commands"
)

type Bot struct {
	BotID            string   `bson:"botID" json:"bot_id"`
	Name             string   `bson:"botName" json:"name"`
	TagsRaw          string   `bson:"tags" json:"tags_raw"`
	Tags             []string `bson:"-" json:"tags"` // This is created by API
	Prefix           *string  `bson:"prefix" json:"prefix"`
	Owner            string   `bson:"main_owner" json:"owner"`
	AdditionalOwners []string `bson:"additional_owners" json:"additional_owners"` // This field should be removed outside of Fates imports
	StaffBot         bool     `bson:"staff" json:"staff_bot"`
	Short            string   `bson:"short" json:"short"`
	Long             string   `bson:"long" json:"long"`
	Library          *string  `bson:"library" json:"library"`
	Website          *string  `bson:"website" json:"website"`
	Donate           *string  `bson:"donate" json:"donate"`
	Support          *string  `bson:"support" json:"support"`
	NSFW             bool     `bson:"nsfw" json:"nsfw"`
	Premium          bool     `bson:"premium" json:"premium"`
	Certified        bool     `bson:"certified" json:"certified"`
	Servers          int      `bson:"servers" json:"servers"`
	Shards           int      `bson:"shards" json:"shards"`
	Votes            int      `bson:"votes" json:"votes"`
	Views            int      `bson:"clicks" json:"views"`
	InviteClicks     int      `bson:"invite_clicks" json:"invites"`
	Github           *string  `bson:"github" json:"github"`
	Banner           *string  `bson:"background" json:"banner"`
	Invite           *string  `bson:"invite" json:"invite"`
}

func parseBot(bot *Bot) *Bot {
	bot.Tags = strings.Split(bot.TagsRaw, ",")

	if *bot.Donate == "None" {
		bot.Donate = nil
	}

	return bot
}

func main() {
	r := mux.NewRouter()

	// Create base payloads before startup

	// Index
	helloWorldB := map[string]string{
		"message": "Hello world from IBL API v5!",
		"docs":    docsSite,
		"ourSite": mainSite,
		"status":  statusPage,
	}

	helloWorld, err := json.Marshal(helloWorldB)

	if err != nil {
		panic(err)
	}

	// Not Found
	notFoundB := map[string]string{
		"message": "Slow down, bucko! We couldn't find this resource *anywhere*!",
	}

	notFound, err := json.Marshal(notFoundB)

	if err != nil {
		panic(err)
	}

	// Bad request
	badRequestB := map[string]string{
		"message": "Slow down, bucko! You're doing something illegal!!!",
	}

	badRequest, err := json.Marshal(badRequestB)

	if err != nil {
		panic(err)
	}

	ctx := context.Background()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoUrl))

	if err != nil {
		panic(err)
	}

	fmt.Println("Connected to mongoDB?")

	db := client.Database("infinity")

	colNames, err := db.ListCollectionNames(ctx, bson.D{})

	fmt.Println("Collections:", colNames)

	if err != nil {
		panic(err)
	}
	godotenv.Load()

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(helloWorld))
	})

	r.HandleFunc("/fates/bots/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method != "GET" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte(badRequest))
			return
		}

		vars := mux.Vars(r)

		botId := vars["id"]

		if botId == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(badRequest))
			return
		}

		if r.Header.Get("Authorization") == "" || r.Header.Get("Authorization") != os.Getenv("FATES_TOKEN") {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(badRequest))
			return
		}

		botCol := db.Collection("bots")

		var bot Bot

		err := botCol.FindOne(ctx, bson.M{"botID": botId}).Decode(&bot)

		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(notFound))
			return
		}

		bot = *parseBot(&bot)

		bytes, err := json.Marshal(bot)

		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Not found"))
			return
		}

		w.Write(bytes)
	})

	adp := DummyAdapter{}

	integrase.StartServer(adp, r)
}
