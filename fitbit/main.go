package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
)

type SetupClient struct {
	ClientID      string
	Secret        string
	CodeVerifier  string
	CodeChallenge string
}

func main() {

	// load the .env

	godotenv.Load()

	// get the command line arguments

	args := os.Args[1:]

	if len(args) == 0 {
		// start the program as usual
		runBot()
	} else if args[0] == "setup" {
		// start a chi server

		client := newSetupClient()

		// print the url to visit

		var port = os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}

		loginURL := "https://www.fitbit.com/oauth2/authorize?client_id=" + client.ClientID +
			"&response_type=code" +
			"&code_challenge=" + client.CodeChallenge +
			"&code_challenge_method=S256" +
			"&scope=activity%20heartrate%20location%20nutrition%20oxygen_saturation%20profile" +
			"%20respiratory_rate%20settings%20sleep%20social%20temperature%20weight" +
			"&redirect_uri=https://fitbit.alice.hackclub.app/callback"

		log.Println("Visit the following URL to authorize the application:")
		log.Println(loginURL)

		r := chi.NewRouter()

		r.Get("/callback", func(w http.ResponseWriter, r *http.Request) {
			HandleFitbitCallback(w, r, client)
		})

		http.ListenAndServe(":"+port, r)
	}
}

func runBot() {
	println("Running bot...")
}

func newSetupClient() *SetupClient {

	// generate the code verifier and code challenge

	var code, err = GenerateCodeVerifier(43)

	if err != nil {
		panic(err)
	}

	var challenge = GenerateCodeChallenge(code)

	return &SetupClient{
		CodeVerifier:  code,
		CodeChallenge: challenge,
		ClientID:      os.Getenv("FITBIT_CLIENT_ID"),
		Secret:        os.Getenv("FITBIT_CLIENT_SECRET"),
	}
}
