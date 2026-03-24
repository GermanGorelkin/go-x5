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
