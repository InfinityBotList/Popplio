package main

import (
	"encoding/json"
	"net/http"
	"os"

	integrase "github.com/MetroReviews/metro-integrase/lib"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

const (
	mongoUrl   = "mongodb://127.0.0.1:27017/infinity" // Is already public in 10 other places so
	docsSite   = "https://docs.botlist.site"
	mainSite   = "https://infinitybotlist.com"
	statusPage = "https://status.botlist.site"
	apiBot     = "https://discord.com/api/oauth2/authorize?client_id=818419115068751892&permissions=140898593856&scope=bot%20applications.commands"
)

func main() {
	r := mux.NewRouter()

	// Create the hello world payload before startup
	helloWorld := map[string]string{
		"message": "Hello world from IBL API v5!",
		"docs":    docsSite,
		"ourSite": mainSite,
		"status":  statusPage,
	}

	bytes, err := json.Marshal(helloWorld)

	if err != nil {
		panic(err)
	}

	godotenv.Load()

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(bytes))
	})

	r.HandleFunc("/fates/bots/{id}", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte("Method not allowed"))
			return
		}

		vars := mux.Vars(r)

		botId := vars["id"]

		if botId == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		if r.Header.Get("Authorization") == "" || r.Header.Get("Authorization") != os.Getenv("FATES_TOKEN") {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorized"))
			return
		}

	})

	adp := DummyAdapter{}

	integrase.StartServer(adp, r)
}
