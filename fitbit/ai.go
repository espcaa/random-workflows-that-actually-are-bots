package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"text/template"
	"time"
)

//go:embed template.tmpl
var sleepLogTemplateString string

//go:embed system_prompt.txt
var systemPromptString string

type AiResponse struct {
	Choices []struct {
		Message AiMessage `json:"message"`
	} `json:"choices"`
}

type AiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AiRequest struct {
	Messages []AiMessage `json:"messages"`
	Model    string      `json:"model"`
}

func (m *AiResponse) GetContent() string {
	if len(m.Choices) > 0 {
		return m.Choices[0].Message.Content
	}
	return ""
}

func Complete(messages []AiMessage, model, aiBaseUrl string) (string, error) {

	request := AiRequest{
		Model:    model,
		Messages: messages,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", aiBaseUrl, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+os.Getenv("AI_API_KEY"))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var response AiResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", err
	}

	return response.GetContent(), nil
}

// system prompt helper

type SleepLogData struct {
	Date                string
	Duration            string
	Efficiency          string
	StartTime           string
	EndTime             string
	MinutesAfterWakeup  int
	MinutesAwake        int
	MinutesAsleep       int
	MinutesToFallAsleep int
	TimeInBed           int
	Stages              SleepStages
	StageDetails        []StageDetail
	History             []HistoryItem
}

type SleepStages struct {
	Deep  int
	Light int
	Rem   int
	Wake  int
}

type StageDetail struct {
	Name            string
	StartTime       string
	EndTime         string
	DurationSeconds int
}

type HistoryItem struct {
	Date       string
	Duration   string
	Efficiency int
}

func FormatSleepLog(data SleepLogData) (string, error) {

	tmpl, err := template.New("sleepLog").Parse(sleepLogTemplateString)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

func GetSystemPrompt() string {
	return systemPromptString
}

func MakeSleepLogData(sleepData *FitbitSleepResponse, rangeData *FitbitSleepResponse) SleepLogData {

	var stageDetails []StageDetail
	const fitbitTimeLayout = "2006-01-02T15:04:05.000"
	for _, s := range sleepData.Sleep[0].Levels.Data {
		startTime, _ := time.Parse(fitbitTimeLayout, s.DateTime)
		endTime := startTime.Add(time.Duration(s.Seconds) * time.Second)
		stageDetails = append(stageDetails, StageDetail{
			Name:            s.Level,
			StartTime:       s.DateTime,
			EndTime:         endTime.Format(fitbitTimeLayout),
			DurationSeconds: s.Seconds,
		})
	}

	// Build history from range data: deduplicate by dateOfSleep, preferring isMainSleep
	seen := make(map[string]bool)
	var history []HistoryItem
	if rangeData != nil {
		// Sort by dateOfSleep ascending so we can take the last 5 unique days
		for _, s := range rangeData.Sleep {
			if seen[s.DateOfSleep] {
				continue
			}
			// Only include isMainSleep entries for cleaner history
			// but fall back if there's no main sleep for that date
			if !s.IsMainSleep {
				continue
			}
			seen[s.DateOfSleep] = true
			history = append(history, HistoryItem{
				Date:       s.DateOfSleep,
				Duration:   fmt.Sprintf("%d minutes", s.Duration/60000),
				Efficiency: s.Efficiency,
			})
		}
		// Take last 5
		if len(history) > 5 {
			history = history[len(history)-5:]
		}
	}

	return SleepLogData{
		Date:                sleepData.Sleep[0].DateOfSleep,
		Duration:            fmt.Sprintf("%d minutes", sleepData.Sleep[0].Duration/60000),
		Efficiency:          fmt.Sprintf("%d%%", sleepData.Sleep[0].Efficiency),
		StartTime:           sleepData.Sleep[0].StartTime,
		EndTime:             sleepData.Sleep[0].EndTime,
		MinutesAfterWakeup:  sleepData.Sleep[0].MinutesAfterWakeup,
		MinutesAwake:        sleepData.Sleep[0].MinutesAwake,
		MinutesAsleep:       sleepData.Sleep[0].MinutesAsleep,
		MinutesToFallAsleep: sleepData.Sleep[0].MinutesToFallAsleep,
		TimeInBed:           sleepData.Sleep[0].TimeInBed,
		Stages: SleepStages{
			Deep:  sleepData.Sleep[0].Levels.Summary.Deep.Minutes,
			Light: sleepData.Sleep[0].Levels.Summary.Light.Minutes,
			Rem:   sleepData.Sleep[0].Levels.Summary.Rem.Minutes,
			Wake:  sleepData.Sleep[0].Levels.Summary.Wake.Minutes,
		},
		StageDetails: stageDetails,
		History:      history,
	}
}
