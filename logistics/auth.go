package logistics

import "fmt"

type AuthService service

type RequestAuth struct {
	Login, Password string
}

type ResponseAuth struct {
	Code        string
	Description string
	Result      struct {
		Token string
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
