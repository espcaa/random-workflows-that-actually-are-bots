package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

type FitbitTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	UserID       string `json:"user_id"`
}

func HandleFitbitCallback(w http.ResponseWriter, r *http.Request, c *SetupClient) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Missing code", http.StatusBadRequest)
		return
	}

	// Exchange the authorization code for an access token
	err := exchangeCodeForToken(code, c)
	if err != nil {
		http.Error(w, "Failed to exchange code for token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// return 200 + a html page saying success

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/html")

	var html = `
	<!DOCTYPE html>
	<html lang="en">
	<head>
	    <meta charset="UTF-8">
	    <meta name="viewport" content="width=device-width, initial-scale=1.0">
	    <title>yay ts should work</title>
	</head>
	<body>
	    <h1>setup complete ^-^</h1>
	</body>
	</html>
	`

	w.Write([]byte(html))
}

func exchangeCodeForToken(code string, c *SetupClient) error {
	data := url.Values{}
	data.Set("client_id", c.ClientID)
	data.Set("code", code)
	data.Set("code_verifier", c.CodeVerifier)
	data.Set("grant_type", "authorization_code")
	data.Set("redirect_uri", "https://fitbit.alice.hackclub.app/callback")
	data.Set("callback_uri", "https://fitbit.alice.hackclub.app/callback")

	req, err := http.NewRequest("POST", "https://api.fitbit.com/oauth2/token", bytes.NewBufferString(data.Encode()))
	if err != nil {
		return err
	}

	// print the request body

	fmt.Println("Request body:", data.Encode())

	// Basic auth header
	authString := c.ClientID + ":" + c.Secret
	base64Auth := base64.StdEncoding.EncodeToString([]byte(authString))
	req.Header.Set("Authorization", "Basic "+base64Auth)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
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

	// store the token in a json file
	tokenFile, err := json.MarshalIndent(tokenResp, "", "  ")
	if err != nil {
		return err
	}

	err = writeFile("tokens.json", tokenFile)
	if err != nil {
		return err
	}

	fmt.Println("Fitbit token saved to fitbit_token.json")

	return nil
}

func writeFile(filename string, data []byte) error {
	return os.WriteFile(filename, data, 0600)
}
