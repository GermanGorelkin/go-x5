package logistics

import (
	"fmt"

	"go.uber.org/zap"
)

type AuthService service

type RequestAuth struct {
	Login    string `json:"email"`
	Password string `json:"password"`
}

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
		return "", fmt.Errorf("failed to auth:%w", err)
	}
	log.Debug("auth request succeeded")

	return res.Result.Token, nil
}
