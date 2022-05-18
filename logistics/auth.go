package logistics

import "fmt"

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
	// TODO validation
	req := RequestAuth{
		Login:    login,
		Password: password,
	}
	var res ResponseAuth
	err := srv.client.httpClient.Post(URL_AUTH, req, &res)
	if err != nil || res.Code != "ok" {
		return "", fmt.Errorf("failed to auth:%w", err)
	}

	return res.Result.Token, nil
}
