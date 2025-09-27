package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type FitbitSleepResponse struct {
	Sleep []struct {
		DateOfSleep         string `json:"dateOfSleep"`
		Duration            int64  `json:"duration"`
		Efficiency          int    `json:"efficiency"`
		StartTime           string `json:"startTime"`
		EndTime             string `json:"endTime"`
		InfoCode            int    `json:"infoCode"`
		IsMainSleep         bool   `json:"isMainSleep"`
		MinutesAfterWakeup  int    `json:"minutesAfterWakeup"`
		MinutesAwake        int    `json:"minutesAwake"`
		MinutesAsleep       int    `json:"minutesAsleep"`
		MinutesToFallAsleep int    `json:"minutesToFallAsleep"`
		LogType             string `json:"logType"`
		TimeInBed           int    `json:"timeInBed"`
		Type                string `json:"type"`

		Levels struct {
			Data []struct {
				DateTime string `json:"dateTime"`
				Level    string `json:"level"`
				Seconds  int    `json:"seconds"`
			} `json:"data"`
			ShortData []struct {
				DateTime string `json:"dateTime"`
				Level    string `json:"level"`
				Seconds  int    `json:"seconds"`
			} `json:"shortData"`
			Summary struct {
				Deep  struct{ Count, Minutes, ThirtyDayAvgMinutes int } `json:"deep"`
				Light struct{ Count, Minutes, ThirtyDayAvgMinutes int } `json:"light"`
				Rem   struct{ Count, Minutes, ThirtyDayAvgMinutes int } `json:"rem"`
				Wake  struct{ Count, Minutes, ThirtyDayAvgMinutes int } `json:"wake"`
			} `json:"summary"`
		} `json:"levels"`

		Summary struct {
			TotalMinutesAsleep int `json:"totalMinutesAsleep"`
			TotalSleepRecords  int `json:"totalSleepRecords"`
			TotalTimeInBed     int `json:"totalTimeInBed"`
			Stages             struct {
				Deep  int `json:"deep"`
				Light int `json:"light"`
				Rem   int `json:"rem"`
				Wake  int `json:"wake"`
			} `json:"stages"`
		} `json:"summary"`

		LogID int64 `json:"logId"`
	} `json:"sleep"`
}

func getSleep(client *FitbitClient, date string) (*FitbitSleepResponse, error) {
	req, err := http.NewRequest("GET", "https://api.fitbit.com/1.2/user/"+client.UserID+"/sleep/date/"+date+".json", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+client.AccessToken)

	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fitbit sleep endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	// print the full json response

	fmt.Println("Fitbit sleep response:", string(body))

	var sleepResp FitbitSleepResponse
	if err := json.Unmarshal(body, &sleepResp); err != nil {
		return nil, err
	}

	return &sleepResp, nil

}

func refreshToken(client *FitbitClient) error {

	var data = fmt.Sprintf("client_id=%s&grant_type=refresh_token&refresh_token=%s", client.SecretClient.ClientID, client.RefreshToken)
	req, err := http.NewRequest("POST", "https://api.fitbit.com/oauth2/token", bytes.NewBufferString(data))
	if err != nil {
		return err
	}

	// Basic auth header
	authString := client.SecretClient.ClientID + ":" + client.SecretClient.Secret
	base64Auth := base64.StdEncoding.EncodeToString([]byte(authString))
	req.Header.Set("Authorization", "Basic "+base64Auth)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fitbit token endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp FitbitTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return err
	}

	client.AccessToken = tokenResp.AccessToken
	client.RefreshToken = tokenResp.RefreshToken
	client.ExpiresIn = tokenResp.ExpiresIn
	client.TokenType = tokenResp.TokenType
	client.UserID = tokenResp.UserID

	// store the token in a json file
	tokenFile, err := json.MarshalIndent(tokenResp, "", "  ")
	if err != nil {
		return err
	}

	err = writeFile("tokens.json", tokenFile)
	if err != nil {
		return err
	}

	fmt.Println("Fitbit token refreshed and saved to tokens.json")

	return nil
}
