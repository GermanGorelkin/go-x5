package insights

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"go.uber.org/zap"
)

// AccessToken represents a KeyCloak access token used for Bearer authentication.
type AccessToken string

// RefreshToken represents a KeyCloak refresh token used to obtain new access tokens.
type RefreshToken string

// JWTToken represents an internal JWT token issued by the X5 Insights API (x5-api-key header).
type JWTToken string

// AuthService handles communication with the KeyCloak OAuth2 endpoint and
// the internal X5 Insights token endpoint to obtain authorization credentials.
type AuthService service

// ResponseKeyCloakTokens holds the full OAuth2 token response returned by KeyCloak,
// including access/refresh tokens, expiry times, and session metadata.
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

// ResponseInternalToken holds the response from the X5 Insights internal auth endpoint.
// The nested Result.Token field contains the JWT used as the x5-api-key header value.
type ResponseInternalToken struct {
	Code   string `json:"code"`
	Result struct {
		Token JWTToken `json:"token"`
	} `json:"result"`
}

// GetKeyCloakTokens performs a Resource Owner Password Credentials grant against the
// KeyCloak realm configured on the client. It sends client_id, username, and password
// as form-encoded data and returns the resulting access and refresh tokens.
func (srv *AuthService) GetKeyCloakTokens(clientID, username, password string) (AccessToken, RefreshToken, error) {
	log := srv.client.loggerFor("auth").With(zap.String("client_id", clientID))
	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("username", username)
	data.Set("password", password)
	data.Set("grant_type", "password")
	encodedData := data.Encode()

	url := fmt.Sprintf(URL_KC_TOKEN, srv.client.KC_URL, srv.client.KC_RELM)

	req, err := http.NewRequest("POST", url, strings.NewReader(encodedData))
	if err != nil {
		log.Error("failed to build keycloak request", zap.Error(err))
		return "", "", fmt.Errorf("failed to build keycloak request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var res ResponseKeyCloakTokens
	log.Debug("requesting keycloak tokens")
	_, err = srv.client.httpClient.Do(context.Background(), req, &res)
	if err != nil || res.AccessToken == "" || res.RefreshToken == "" {
		log.Error("failed to get keycloak tokens", zap.Error(err))
		return "", "", fmt.Errorf("failed to get keycloak tokens: %w", err)
	}
	log.Debug("keycloak tokens received")

	return res.AccessToken, res.RefreshToken, nil
}

// GetInternalToken exchanges a KeyCloak access token for an internal X5 Insights JWT.
// It sets the Bearer authorization header from the provided access token, calls the
// internal /auth/token endpoint, and returns the JWT that must be sent as x5-api-key.
func (srv *AuthService) GetInternalToken(access AccessToken, refresh RefreshToken) (JWTToken, error) {
	log := srv.client.loggerFor("auth")
	// cookie := fmt.Sprintf("kc-access=%s; kc-state=%s;", access, refresh)
	// srv.client.httpClient.SetHeader("cookie", cookie)
	srv.client.httpClient.SetHeader("Authorization", fmt.Sprintf("Bearer %s", access))

	url := fmt.Sprintf(URL_INTERNAL_TOKEN, srv.client.API_URL)

	var res ResponseInternalToken
	log.Debug("requesting internal token")
	err := srv.client.httpClient.Get(url, &res)
	if err != nil || res.Result.Token == "" {
		log.Error("failed to get internal token", zap.Error(err), zap.String("code", res.Code))
		return "", fmt.Errorf("failed to get internal token: %w", err)
	}
	log.Debug("internal token received")

	return res.Result.Token, nil
}
