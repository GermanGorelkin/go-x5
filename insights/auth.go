package insights

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type (
	AccessToken  string
	RefreshToken string
	JWTToken     string
)

// AuthService handles communication with the auth related KeyCloak and internal token
type AuthService service

// ResponseKeyCloakTokens is a response from KeyCloak
type ResponseKeyCloakTokens struct {
	AccessToken      AccessToken  `json:"access_token"`
	ExpiresIn        int          `json:"expires_in"`
	RefreshExpiresIn int          `json:"refresh_expires_in"`
	RefreshToken     RefreshToken `json:"refresh_token"`
	TokenType        string       `json:"token_type"`
	NotBeforePolicy  int          `json:"not-before-policy"`
	SessionState     string       `json:"session_state"`
	Scope            string       `json:"scope"`
}

// ResponseInternalToken is a response from API Token
type ResponseInternalToken struct {
	Code   string `json:"code"`
	Result struct {
		Token JWTToken `json:"token"`
	} `json:"result"`
}

// GetKeyCloakTokens returns access and refresh tokens
func (srv *AuthService) GetKeyCloakTokens(clientID, username, password string) (AccessToken, RefreshToken, error) {
	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("username", username)
	data.Set("password", password)
	data.Set("grant_type", "password")
	encodedData := data.Encode()

	url := fmt.Sprintf(URL_KC_TOKEN, srv.client.KC_URL, srv.client.KC_RELM)

	req, err := http.NewRequest("POST", url, strings.NewReader(encodedData))
	if err != nil {
		return "", "", fmt.Errorf("failed to build NewRequest:%w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var res ResponseKeyCloakTokens
	_, err = srv.client.httpClient.Do(context.Background(), req, &res)
	if err != nil || res.AccessToken == "" || res.RefreshToken == "" {
		return "", "", fmt.Errorf("failed to kc auth:%w", err)
	}

	return res.AccessToken, res.RefreshToken, nil
}

// GetInternalToken returns internal token
func (srv *AuthService) GetInternalToken(access AccessToken, refresh RefreshToken) (JWTToken, error) {
	cookie := fmt.Sprintf("kc-access=%s; kc-state=%s;", access, refresh)
	srv.client.httpClient.SetHeader("cookie", cookie)

	url := fmt.Sprintf(URL_INTERNAL_TOKEN, srv.client.API_URL)

	var res ResponseInternalToken
	err := srv.client.httpClient.Get(url, &res)
	if err != nil || res.Result.Token == "" {
		return "", fmt.Errorf("failed to internal auth:%w", err)
	}

	return res.Result.Token, nil
}
