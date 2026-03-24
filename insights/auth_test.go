package insights

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthService_GetKeyCloakTokens_OK(t *testing.T) {
	const (
		realm        = "test-realm"
		clientID     = "test-client"
		username     = "user"
		password     = "pass"
		accessToken  = "access-token-abc"
		refreshToken = "refresh-token-xyz"
	)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, fmt.Sprintf("/auth/realms/%s/protocol/openid-connect/token", realm), r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

		err := r.ParseForm()
		require.NoError(t, err)
		assert.Equal(t, clientID, r.PostFormValue("client_id"))
		assert.Equal(t, username, r.PostFormValue("username"))
		assert.Equal(t, password, r.PostFormValue("password"))
		assert.Equal(t, "password", r.PostFormValue("grant_type"))

		w.Header().Set("Content-Type", "application/json")
		_, err = fmt.Fprintf(w, `{
			"access_token": "%s",
			"expires_in": 300,
			"refresh_expires_in": 1800,
			"refresh_token": "%s",
			"token_type": "Bearer",
			"not-before-policy": 0,
			"session_state": "session-123",
			"scope": "openid"
		}`, accessToken, refreshToken)
		require.NoError(t, err)
	}))
	defer ts.Close()

	client, err := NewClient(ClintConf{
		KC_URL:   ts.URL,
		KC_RELM:  realm,
		ClientID: clientID,
		Login:    username,
		Password: password,
		API_URL:  ts.URL,
	})
	require.NoError(t, err)

	access, refresh, err := client.Auth.GetKeyCloakTokens(clientID, username, password)
	require.NoError(t, err)
	assert.Equal(t, AccessToken(accessToken), access)
	assert.Equal(t, RefreshToken(refreshToken), refresh)
}

func TestAuthService_GetKeyCloakTokens_EmptyTokens(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := fmt.Fprintln(w, `{"access_token":"","refresh_token":""}`)
		require.NoError(t, err)
	}))
	defer ts.Close()

	client, err := NewClient(ClintConf{
		KC_URL:   ts.URL,
		KC_RELM:  "test-realm",
		ClientID: "test-client",
		Login:    "user",
		Password: "pass",
		API_URL:  ts.URL,
	})
	require.NoError(t, err)

	_, _, err = client.Auth.GetKeyCloakTokens("test-client", "user", "pass")
	assert.Error(t, err)
}

func TestAuthService_GetKeyCloakTokens_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, err := fmt.Fprintln(w, `{"error":"internal server error"}`)
		require.NoError(t, err)
	}))
	defer ts.Close()

	client, err := NewClient(ClintConf{
		KC_URL:   ts.URL,
		KC_RELM:  "test-realm",
		ClientID: "test-client",
		Login:    "user",
		Password: "pass",
		API_URL:  ts.URL,
	})
	require.NoError(t, err)

	_, _, err = client.Auth.GetKeyCloakTokens("test-client", "user", "pass")
	assert.Error(t, err)
}

func TestAuthService_GetInternalToken_OK(t *testing.T) {
	const (
		accessToken = AccessToken("access-token-for-bearer")
		jwtToken    = "jwt-123"
	)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/public/auth/token", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, fmt.Sprintf("Bearer %s", accessToken), r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		_, err := fmt.Fprintf(w, `{"code":"ok","result":{"token":"%s"}}`, jwtToken)
		require.NoError(t, err)
	}))
	defer ts.Close()

	client, err := NewClient(ClintConf{
		KC_URL:   ts.URL,
		KC_RELM:  "test-realm",
		ClientID: "test-client",
		Login:    "user",
		Password: "pass",
		API_URL:  ts.URL,
	})
	require.NoError(t, err)

	jwt, err := client.Auth.GetInternalToken(accessToken, "refresh-token")
	require.NoError(t, err)
	assert.Equal(t, JWTToken(jwtToken), jwt)
}

func TestAuthService_GetInternalToken_EmptyToken(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := fmt.Fprintln(w, `{"code":"ok","result":{"token":""}}`)
		require.NoError(t, err)
	}))
	defer ts.Close()

	client, err := NewClient(ClintConf{
		KC_URL:   ts.URL,
		KC_RELM:  "test-realm",
		ClientID: "test-client",
		Login:    "user",
		Password: "pass",
		API_URL:  ts.URL,
	})
	require.NoError(t, err)

	_, err = client.Auth.GetInternalToken("some-access", "some-refresh")
	assert.Error(t, err)
}

func TestAuthService_GetInternalToken_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, err := fmt.Fprintln(w, `{"error":"internal server error"}`)
		require.NoError(t, err)
	}))
	defer ts.Close()

	client, err := NewClient(ClintConf{
		KC_URL:   ts.URL,
		KC_RELM:  "test-realm",
		ClientID: "test-client",
		Login:    "user",
		Password: "pass",
		API_URL:  ts.URL,
	})
	require.NoError(t, err)

	_, err = client.Auth.GetInternalToken("some-access", "some-refresh")
	assert.Error(t, err)
}
