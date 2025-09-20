package main

import (
	"os"
	"os/exec"
)

var channelID string = "C080SMXTRS8"

func main() {
	// get the command line arguments

	var args []string
	for i, arg := range os.Args {
		if i == 0 {
			continue
		}
		args = append(args, arg)
	}

	// check if the first arg is sleep or wake

	if len(args) == 0 {
		println("Usage: wake-sleep <sleep|wake>")
		return
	}

	switch args[0] {
	case "sleep":
		sendMessageToSlack("laptop's closed :(")
	case "wake":
		sendMessageToSlack("laptop's opened :D")
	default:
		println("Usage: wake-sleep <sleep|wake>")
	}
}

func sendMessageToSlack(message string) {
	// send a message to slack using curl
	slackbottoken := os.Getenv("SLACK_WORKFLOW_BOT_TOKEN")
	if slackbottoken == "" {
		println("SLACK_WORKFLOW_BOT_TOKEN is not set")
		return
	}

	resp, err := exec.Command("curl", "-X", "POST", "https://slack.com/api/chat.postMessage", "-H", "Authorization: Bearer "+slackbottoken, "-H", "Content-type: application/json", "--data", `{"channel":"`+channelID+`","text":"`+message+`"}`).Output()
	if err != nil {
		println("Error sending message to Slack:", err.Error())
		return
	}
	println("Response from Slack:", string(resp))
}
