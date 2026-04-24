package insights

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestNewClient_UsesNopLoggerWhenNil(t *testing.T) {
	client, err := NewClient(ClintConf{
		API_URL: "http://example.com",
	})
	require.NoError(t, err)

	assert.NotNil(t, client.logger)
}

func TestClient_Authorization_RedactsTokensInLogs(t *testing.T) {
	const (
		realm        = "test-realm"
		accessToken  = "access-secret"
		refreshToken = "refresh-secret"
		jwtToken     = "jwt-secret"
	)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case fmt.Sprintf("/auth/realms/%s/protocol/openid-connect/token", realm):
			_, err := fmt.Fprintf(w, `{"access_token":"%s","refresh_token":"%s"}`, accessToken, refreshToken)
			require.NoError(t, err)
		case "/api/v1/public/auth/token":
			assert.Equal(t, fmt.Sprintf("Bearer %s", accessToken), r.Header.Get("Authorization"))
			_, err := fmt.Fprintf(w, `{"code":"ok","result":{"token":"%s"}}`, jwtToken)
			require.NoError(t, err)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer ts.Close()

	core, observedLogs := observer.New(zap.DebugLevel)
	logger := zap.New(core)

	client, err := NewClient(ClintConf{
		KC_URL:   ts.URL,
		KC_RELM:  realm,
		ClientID: "client-id",
		Login:    "login",
		Password: "password",
		API_URL:  ts.URL,
		Logger:   logger,
	})
	require.NoError(t, err)

	require.NoError(t, client.Authorization())
	assert.Equal(t, 1, observedLogs.FilterMessage("authorization flow completed").Len())

	for _, entry := range observedLogs.AllUntimed() {
		assert.NotContains(t, entry.Message, accessToken)
		assert.NotContains(t, entry.Message, refreshToken)
		assert.NotContains(t, entry.Message, jwtToken)
		for key, value := range entry.ContextMap() {
			assert.NotContains(t, fmt.Sprint(value), accessToken, "field %s leaked access token", key)
			assert.NotContains(t, fmt.Sprint(value), refreshToken, "field %s leaked refresh token", key)
			assert.NotContains(t, fmt.Sprint(value), jwtToken, "field %s leaked jwt token", key)
		}
	}
}

func TestClient_Authorization_ReusesValidKeyCloakToken(t *testing.T) {
	const realm = "test-realm"

	var (
		passwordGrantCalls int
		refreshGrantCalls  int
		internalTokenCalls int
	)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case fmt.Sprintf("/auth/realms/%s/protocol/openid-connect/token", realm):
			require.NoError(t, r.ParseForm())

			switch r.PostFormValue("grant_type") {
			case "password":
				passwordGrantCalls++
				_, err := fmt.Fprint(w, `{"access_token":"access-1","expires_in":300,"refresh_expires_in":1800,"refresh_token":"refresh-1"}`)
				require.NoError(t, err)
			case "refresh_token":
				refreshGrantCalls++
				t.Fatal("did not expect refresh token grant while access token is still valid")
			default:
				t.Fatalf("unexpected grant type: %s", r.PostFormValue("grant_type"))
			}
		case "/api/v1/public/auth/token":
			internalTokenCalls++
			assert.Equal(t, "Bearer access-1", r.Header.Get("Authorization"))

			_, err := fmt.Fprintf(w, `{"code":"ok","result":{"token":"jwt-%d"}}`, internalTokenCalls)
			require.NoError(t, err)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer ts.Close()

	client, err := NewClient(ClintConf{
		KC_URL:   ts.URL,
		KC_RELM:  realm,
		ClientID: "client-id",
		Login:    "login",
		Password: "password",
		API_URL:  ts.URL,
	})
	require.NoError(t, err)

	require.NoError(t, client.Authorization())
	require.NoError(t, client.Authorization())

	assert.Equal(t, 1, passwordGrantCalls)
	assert.Equal(t, 0, refreshGrantCalls)
	assert.Equal(t, 1, internalTokenCalls)
}

func TestClient_Authorization_RefreshesExpiredKeyCloakToken(t *testing.T) {
	const realm = "test-realm"

	var (
		passwordGrantCalls int
		refreshGrantCalls  int
		internalTokenCalls int
	)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case fmt.Sprintf("/auth/realms/%s/protocol/openid-connect/token", realm):
			require.NoError(t, r.ParseForm())

			switch r.PostFormValue("grant_type") {
			case "password":
				passwordGrantCalls++
				_, err := fmt.Fprint(w, `{"access_token":"access-1","expires_in":0,"refresh_expires_in":1800,"refresh_token":"refresh-1"}`)
				require.NoError(t, err)
			case "refresh_token":
				refreshGrantCalls++
				assert.Equal(t, "refresh-1", r.PostFormValue("refresh_token"))
				_, err := fmt.Fprint(w, `{"access_token":"access-2","expires_in":300,"refresh_expires_in":1800,"refresh_token":"refresh-2"}`)
				require.NoError(t, err)
			default:
				t.Fatalf("unexpected grant type: %s", r.PostFormValue("grant_type"))
			}
		case "/api/v1/public/auth/token":
			internalTokenCalls++
			if internalTokenCalls == 1 {
				assert.Equal(t, "Bearer access-1", r.Header.Get("Authorization"))
			} else {
				assert.Equal(t, "Bearer access-2", r.Header.Get("Authorization"))
			}

			_, err := fmt.Fprintf(w, `{"code":"ok","result":{"token":"jwt-%d"}}`, internalTokenCalls)
			require.NoError(t, err)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer ts.Close()

	client, err := NewClient(ClintConf{
		KC_URL:   ts.URL,
		KC_RELM:  realm,
		ClientID: "client-id",
		Login:    "login",
		Password: "password",
		API_URL:  ts.URL,
	})
	require.NoError(t, err)

	require.NoError(t, client.Authorization())
	require.NoError(t, client.Authorization())

	assert.Equal(t, 1, passwordGrantCalls)
	assert.Equal(t, 1, refreshGrantCalls)
	assert.Equal(t, 2, internalTokenCalls)
}

func TestClient_Authorization_RetriesPasswordGrantWhenInternalTokenRejectsRefreshedAccess(t *testing.T) {
	const realm = "test-realm"

	var (
		passwordGrantCalls int
		refreshGrantCalls  int
		internalTokenCalls int
	)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case fmt.Sprintf("/auth/realms/%s/protocol/openid-connect/token", realm):
			require.NoError(t, r.ParseForm())

			switch r.PostFormValue("grant_type") {
			case "password":
				passwordGrantCalls++
				if passwordGrantCalls == 1 {
					_, err := fmt.Fprint(w, `{"access_token":"access-1","expires_in":0,"refresh_expires_in":1800,"refresh_token":"refresh-1"}`)
					require.NoError(t, err)
					return
				}
				_, err := fmt.Fprint(w, `{"access_token":"access-3","expires_in":300,"refresh_expires_in":1800,"refresh_token":"refresh-3"}`)
				require.NoError(t, err)
			case "refresh_token":
				refreshGrantCalls++
				assert.Equal(t, "refresh-1", r.PostFormValue("refresh_token"))
				_, err := fmt.Fprint(w, `{"access_token":"access-2","expires_in":300,"refresh_expires_in":1800,"refresh_token":"refresh-2"}`)
				require.NoError(t, err)
			default:
				t.Fatalf("unexpected grant type: %s", r.PostFormValue("grant_type"))
			}
		case "/api/v1/public/auth/token":
			internalTokenCalls++
			switch r.Header.Get("Authorization") {
			case "Bearer access-1":
				_, err := fmt.Fprint(w, `{"code":"ok","result":{"token":"jwt-1"}}`)
				require.NoError(t, err)
			case "Bearer access-2":
				w.WriteHeader(http.StatusUnauthorized)
				_, err := fmt.Fprint(w, `{"code":"access_forbidden","description":"auth failed"}`)
				require.NoError(t, err)
			case "Bearer access-3":
				_, err := fmt.Fprint(w, `{"code":"ok","result":{"token":"jwt-3"}}`)
				require.NoError(t, err)
			default:
				t.Fatalf("unexpected authorization header: %s", r.Header.Get("Authorization"))
			}
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer ts.Close()

	client, err := NewClient(ClintConf{
		KC_URL:   ts.URL,
		KC_RELM:  realm,
		ClientID: "client-id",
		Login:    "login",
		Password: "password",
		API_URL:  ts.URL,
	})
	require.NoError(t, err)

	require.NoError(t, client.Authorization())
	require.NoError(t, client.Authorization())

	assert.Equal(t, AccessToken("access-3"), client.authCache.state.access)
	assert.Equal(t, RefreshToken("refresh-3"), client.authCache.state.refresh)
	assert.Equal(t, JWTToken("jwt-3"), client.authCache.state.jwt)
	assert.Equal(t, 2, passwordGrantCalls)
	assert.Equal(t, 1, refreshGrantCalls)
	assert.Equal(t, 3, internalTokenCalls)
}

func TestClient_Authorization_ReusesSharedAuthCacheAcrossClients(t *testing.T) {
	const realm = "test-realm"

	var (
		passwordGrantCalls int
		internalTokenCalls int
	)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case fmt.Sprintf("/auth/realms/%s/protocol/openid-connect/token", realm):
			passwordGrantCalls++
			_, err := fmt.Fprint(w, `{"access_token":"access-1","expires_in":300,"refresh_expires_in":1800,"refresh_token":"refresh-1"}`)
			require.NoError(t, err)
		case "/api/v1/public/auth/token":
			internalTokenCalls++
			assert.Equal(t, "Bearer access-1", r.Header.Get("Authorization"))
			_, err := fmt.Fprint(w, `{"code":"ok","result":{"token":"jwt-1"}}`)
			require.NoError(t, err)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer ts.Close()

	authCache := NewAuthCache()
	cfg := ClintConf{
		KC_URL:    ts.URL,
		KC_RELM:   realm,
		ClientID:  "client-id",
		Login:     "login",
		Password:  "password",
		API_URL:   ts.URL,
		AuthCache: authCache,
	}

	clientA, err := NewClient(cfg)
	require.NoError(t, err)
	clientB, err := NewClient(cfg)
	require.NoError(t, err)

	require.NoError(t, clientA.Authorization())
	require.NoError(t, clientB.Authorization())

	assert.Equal(t, 1, passwordGrantCalls)
	assert.Equal(t, 1, internalTokenCalls)
}
