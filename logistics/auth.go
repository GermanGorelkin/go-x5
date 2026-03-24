package logistics

import (
	"fmt"

	"go.uber.org/zap"
)

// AuthService handles authentication against the X5 Logistics API.
// It is backed by the shared service struct so it can reuse the parent Client.
type AuthService service

// RequestAuth is the JSON payload sent to the auth endpoint.
// Login is serialized as "email" to match the API contract.
type RequestAuth struct {
	Login    string `json:"email"`
	Password string `json:"password"`
}

// ResponseAuth is the JSON payload returned by the auth endpoint.
// A successful response contains Code "ok" and a bearer token inside Result.
type ResponseAuth struct {
	Code        string `json:"code"`
	Description string `json:"description"`
	Result      struct {
		Token string `json:"token"`
	}
}

// Auth gets token for the given login and password
func (srv *AuthService) Auth(login, password string) (string, error) {
	log := srv.client.loggerFor("auth")
	// TODO validation
	req := RequestAuth{
		Login:    login,
		Password: password,
	}
	var res ResponseAuth
	log.Debug("sending auth request")
	err := srv.client.httpClient.Post(URL_AUTH, req, &res)
	if err != nil || res.Code != "ok" {
		log.Error("auth request failed", zap.Error(err), zap.String("code", res.Code))
		return "", fmt.Errorf("auth request failed: %w", err)
	}
	log.Debug("auth request succeeded")

	return res.Result.Token, nil
}
