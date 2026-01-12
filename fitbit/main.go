package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
)

type SecretClient struct {
	ClientID      string
	Secret        string
	CodeVerifier  string
	CodeChallenge string
	CallbackURL   string
}

type FitbitClient struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresIn    int64        `json:"expires_in"`
	TokenType    string       `json:"token_type"`
	UserID       string       `json:"user_id"`
	SecretClient SecretClient `json:"-"`
	GoalHours    float64
}

var callbackUrl string = "https://fitbit.hackclub.cc/callback"

func main() {

	signal.Ignore(syscall.SIGPIPE)
	// load the .env

	godotenv.Load()

	// get the command line arguments

	args := os.Args[1:]

	if len(args) == 0 {
		// start the program as usual
		startApp(false)
	} else if args[0] == "setup" {
		// start a chi server

		client := newSecretClient()

		// print the url to visit

		var port = os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}

		if os.Getenv("FITBIT_CALLBACK_URL") != "" {
			client.CallbackURL = os.Getenv("FITBIT_CALLBACK_URL")
		} else {
			client.CallbackURL = "https://fitbit.hackclub.cc/callback"
		}

		callbackUrl = client.CallbackURL

		v := url.Values{}
		v.Set("client_id", client.ClientID)
		v.Set("response_type", "code")
		v.Set("code_challenge", client.CodeChallenge)
		v.Set("code_challenge_method", "S256")
		v.Set("scope", "activity heartrate location nutrition oxygen_saturation profile respiratory_rate settings sleep social temperature weight")
		v.Set("callback_uri", callbackUrl)
		v.Set("redirect_uri", callbackUrl)

		loginURL := "https://www.fitbit.com/oauth2/authorize?" + v.Encode()

		fmt.Println(loginURL)

		log.Println("Visit the following URL to authorize the application:")
		log.Println(loginURL)

		r := chi.NewRouter()

		r.Get("/callback", func(w http.ResponseWriter, r *http.Request) {
			HandleFitbitCallback(w, r, client)
		})

		http.ListenAndServe(":"+port, r)
	} else if args[0] == "test" {
		startApp(true)
	} else {
		fmt.Println("Usage: go run . [setup|test|nothing]")
	}
}

func newFitbitClientFromJSON(data []byte) (*FitbitClient, error) {
	var client = &FitbitClient{}

	err := json.Unmarshal(data, client)

	if err != nil {
		return nil, err
	}

	client.GoalHours = 8.0 // default to 8 hours

	return client, nil
}

func newSecretClient() *SecretClient {

	// generate the code verifier and code challenge

	var code, err = GenerateCodeVerifier(43)

	if err != nil {
		panic(err)
	}

	var challenge = GenerateCodeChallenge(code)

	return &SecretClient{
		CodeVerifier:  code,
		CodeChallenge: challenge,
		ClientID:      os.Getenv("FITBIT_CLIENT_ID"),
		Secret:        os.Getenv("FITBIT_CLIENT_SECRET"),
	}
}

func startApp(runTest bool) {

	// check if tokens.json exists

	if _, err := os.Stat("tokens.json"); os.IsNotExist(err) {
		log.Fatal("tokens.json not found, please run 'go run main.go setup' first")
	}

	tokenFile, err := os.ReadFile("tokens.json")

	client, err := newFitbitClientFromJSON(tokenFile)
	client.SecretClient = *newSecretClient()

	if err != nil {
		log.Fatal(err)
	}

	if err != nil {
		log.Fatal(err)
	}

	log.Println("running this janky bot")

	runBot(client, runTest)
}

func checkNewSleepData(client *FitbitClient) bool {
	var dateString string = time.Now().Format("2006-01-02")
	sleep, err := getSleep(client, dateString)
	if err != nil {
		log.Println("Error getting sleep data:", err)
		return false
	}

	if len(sleep.Sleep) == 0 {
		log.Println("No sleep data found for", dateString)
		return false
	} else {
		log.Println("Found sleep data for", dateString)
		return true
	}
}

func runBot(c *FitbitClient, runTest bool) {

	refreshTicker := time.NewTicker(6 * time.Hour)
	defer refreshTicker.Stop()

	go func() {
		for range refreshTicker.C {
			if err := refreshToken(c); err != nil {
				log.Println("Error refreshing token:", err)
			}
		}
	}()

	var lastSentDate string

	for {
		now := time.Now()
		next5am := time.Date(now.Year(), now.Month(), now.Day(), 5, 0, 0, 0, now.Location())
		if now.After(next5am) {
			next5am = next5am.Add(24 * time.Hour)
		}
		timeUntilNext5am := next5am.Sub(now)
		log.Println("Sleeping until 5 AM:", next5am)

		if !runTest {
			time.Sleep(timeUntilNext5am)
		} else {
			log.Println("Test mode: skipping sleep until 5 AM")
		}

		for {
			today := time.Now().Format("2006-01-02")
			if todayHour := time.Now().Hour(); todayHour > 22 {
				break
			}

			sleepData, err := getSleep(c, today)
			if err != nil {
				log.Println("Error getting sleep data:", err)

				lastSentDate = today
				continue
			}

			if len(sleepData.Sleep) == 0 {
				log.Println("No sleep data yet, retrying in 1 hour")
				time.Sleep(1 * time.Hour)
				continue
			}

			if today != lastSentDate {
				var totalSleepMillis int64
				start := sleepData.Sleep[0].StartTime
				end := sleepData.Sleep[0].EndTime
				for _, s := range sleepData.Sleep {
					totalSleepMillis += s.Duration
					if s.StartTime < start {
						start = s.StartTime
					}
					if s.EndTime > end {
						end = s.EndTime
					}
				}

				const fitbitTimeLayout = "2006-01-02T15:04:05.000"

				startTime, err := time.Parse(fitbitTimeLayout, sleepData.Sleep[0].StartTime)
				if err != nil {
					log.Println("Error parsing start time:", err)
					startTime = time.Now()
				}

				endTime, err := time.Parse(fitbitTimeLayout, sleepData.Sleep[0].EndTime)
				if err != nil {
					log.Println("Error parsing end time:", err)
					endTime = time.Now()
				}

				startStr := startTime.Format("3:04 PM")
				endStr := endTime.Format("3:04 PM")
				hours := float64(totalSleepMillis) / (1000 * 60 * 60)
				hoursStr := fmt.Sprintf("%.1f", hours)

				messageText := fmt.Sprintf("I slept from %s -> %s for a total of %s hours!", startStr, endStr, hoursStr)
				msg := SlackMessage{
					Channel: os.Getenv("SLACK_CHANNEL_ID"),
					Text:    messageText,
				}
				if err := sendSlackMessage(msg); err != nil {
					log.Println("Error sending Slack message:", err)
				}

				bar := generateSleepBar(totalSleepMillis, c.GoalHours)
				barMessage := SlackMessage{
					Channel: os.Getenv("SLACK_CHANNEL_ID"),
					Text:    fmt.Sprintf("`%s` (%.1fh/%.1fh)", bar, hours, c.GoalHours),
				}
				if err := sendSlackMessage(barMessage); err != nil {
					log.Println("Error sending Slack message:", err)
				}

				lastSentDate = today
			} else {
				log.Println("Already sent sleep data for today")
			}

			time.Sleep(1 * time.Hour)
		}
	}
}

func generateSleepBar(sleptMillis int64, goalHours float64) string {
	sleptHours := float64(sleptMillis) / (1000 * 60 * 60)
	percent := (sleptHours / goalHours) * 100
	if percent > 100 {
		percent = 100
	}
	totalBlocks := 10
	filled := int(percent / 10)
	if filled > totalBlocks {
		filled = totalBlocks
	}

	return strings.Repeat("█", filled) + strings.Repeat("░", totalBlocks-filled)
}
