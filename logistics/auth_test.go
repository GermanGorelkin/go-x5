package logistics

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAuthService_Auth_OK(t *testing.T) {
	login := "login"
	password := "pass"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/logistics/auth", r.URL.Path)

		b, err := ioutil.ReadAll(r.Body)
		assert.NoError(t, err)

		var req RequestAuth
		err = json.Unmarshal(b, &req)
		assert.NoError(t, err)
		assert.Equal(t, login, req.Login)
		assert.Equal(t, password, req.Password)

		_, err = fmt.Fprintln(w, `{
			"code": "ok",
			"result": {
				"token": "a.b.c"
			}
		}`)
		assert.NoError(t, err)
	}))
	defer ts.Close()

	client, err := NewClient(ClintConf{
		Instance: ts.URL,
	})
	assert.NoError(t, err)

	token, err := client.Auth.Auth(login, password)
	assert.NoError(t, err)
	assert.Equal(t, "a.b.c", token)
}

func TestAuthService_Auth_400(t *testing.T) {
	login := "login"
	password := "pass"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/logistics/auth", r.URL.Path)

		b, err := ioutil.ReadAll(r.Body)
		assert.NoError(t, err)

		var req RequestAuth
		err = json.Unmarshal(b, &req)
		assert.NoError(t, err)
		assert.Equal(t, login, req.Login)
		assert.Equal(t, password, req.Password)

		w.WriteHeader(http.StatusBadRequest)
		_, err = fmt.Fprintln(w, `{
			"code": "validation_error",
			"description": "Ошибка авторизации"
		}`)
		assert.NoError(t, err)
	}))
	defer ts.Close()

	client, err := NewClient(ClintConf{
		Instance: ts.URL,
	})
	assert.NoError(t, err)

	_, err = client.Auth.Auth(login, password)
	assert.Error(t, err)
}
