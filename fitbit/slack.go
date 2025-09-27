package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

type SlackMessage struct {
	Channel string `json:"channel"`
	Text    string `json:"text"`
}

func sendSlackMessage(message SlackMessage) error {
	token := os.Getenv("SLACK_BOT_TOKEN")
	log.Println("Sending Slack message to channel:", message.Channel)
	log.Println("Message text:", message.Text)
	log.Println("Token:", token)
	if token == "" {
		return fmt.Errorf("$SLACK_TOKEN not set")
	}

	body := fmt.Sprintf(`{"channel":"%s","text":"%s"}`, message.Channel, message.Text)
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
		return errors.New(result.Error)
	}

	log.Println("Slack message sent:", message)
	return nil
}
