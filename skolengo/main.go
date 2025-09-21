package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	skolengo "github.com/espcaa/skolen-go"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	file, err := os.Open("tokens.json")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		log.Fatal(err)
	}

	client, err := skolengo.NewClientFromJSON(data)
	if err != nil {
		log.Fatal(err)
	}

	for {
		now := time.Now()
		timetable, err := client.GetTimetable(client.UserInfo.UserID, client.UserInfo.SchoolID, client.UserInfo.EMSCode, now, now.AddDate(0, 0, 1), 0)
		if err != nil {
			log.Println("Error fetching timetable:", err)
			scheduleMessage(now.Add(10*time.Second), "there was an error fetching the timetable, there is probably a problem with the token or skolengo did an update/is down :heavysob:")
			time.Sleep(1 * time.Minute)
			panic(err)
		}

		for _, day := range timetable {
			if len(day.Lessons) == 0 {
				continue
			}

			first, last := day.Lessons[0], day.Lessons[len(day.Lessons)-1]

			// calculate an emoji based on the numbers of lessons

			emmojiDict := map[int]string{
				1: ":fire:",
				2: ":goat:",
				3: ":yay:",
				4: ":thumbup:",
				5: ":updownvote:",
				6: ":heavysob:",
				7: ":heaviestsob:",
				8: ":heaviestersob:",
				9: ":skulley:",
			}
			emoji, ok := emmojiDict[len(day.Lessons)]
			if !ok {
				emoji = ":ten:"
			}

			var totalDuration time.Duration
			for _, lesson := range day.Lessons {
				totalDuration += lesson.EndDateTime.Sub(lesson.StartDateTime)
			}
			log.Printf("Total duration of school today: %v\n", totalDuration)

			scheduleMessage(first.StartDateTime.Add(-1*time.Minute), fmt.Sprintf(
				"i'm starting school now with %s :3d-sad-emoji:",
				first.Subject.Label,
			))
			scheduleMessage(first.StartDateTime.Add(2*time.Second), fmt.Sprintf(
				"i have %s hours of school today %s and should be done at %s",
				totalDuration.Truncate(time.Minute).String(),
				emoji,
				last.EndDateTime.Format("15:04"),
			))

			scheduleMessage(last.EndDateTime.Add(1*time.Minute), "i'm done with school for today :yay:")
		}

		nextDay := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
		time.Sleep(time.Until(nextDay))
	}
}

func scheduleMessage(t time.Time, msg string) {
	duration := time.Until(t)
	if duration <= 0 {
		return
	}
	go func() {
		time.Sleep(duration)
		if err := sendSlackMessage(msg); err != nil {
			log.Println("Error sending Slack message:", err)
		}
	}()
}

func sendSlackMessage(message string) error {
	token, channel := os.Getenv("SLACK_TOKEN"), os.Getenv("SLACK_CHANNEL_ID")
	if token == "" || channel == "" {
		return fmt.Errorf("Slack token or channel not set")
	}

	body := fmt.Sprintf(`{"channel":"%s","text":"%s"}`, channel, message)
	req, err := http.NewRequest("POST", "https://slack.com/api/chat.postMessage", strings.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result struct {
		OK    bool   `json:"ok"`
		Error string `json:"error,omitempty"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	if !result.OK {
		return fmt.Errorf("Slack API error: %s", result.Error)
	}

	log.Println("Slack message sent:", message)
	return nil
}
