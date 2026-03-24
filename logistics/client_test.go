package logistics

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestNewClient_UsesNopLoggerWhenNil(t *testing.T) {
	client, err := NewClient(ClintConf{
		Instance: "http://example.com",
	})
	require.NoError(t, err)

	assert.NotNil(t, client.logger)
}

func TestClient_AuthInterceptor_RedactsTokensInLogs(t *testing.T) {
	const (
		reportID    = "report-123"
		tokenValue  = "super-secret-token"
		bodyMessage = "Ошибка получения доступа для указанного токена"
	)

	var authCalls int
	var statusCalls int

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case URL_AUTH:
			authCalls++
			_, err := fmt.Fprintf(w, `{"code":"ok","result":{"token":"%s"}}`, tokenValue)
			require.NoError(t, err)
		case fmt.Sprintf(URL_REPORT_STATUS, reportID):
			statusCalls++
			if statusCalls == 1 {
				w.WriteHeader(http.StatusUnauthorized)
				_, err := fmt.Fprintf(w, `{"code":"unauthorized","description":"%s"}`, bodyMessage)
				require.NoError(t, err)
				return
			}
			_, err := fmt.Fprint(w, `{"code":"ok","result":{"reportStatus":"DONE","partIds":["part-1"]}}`)
			require.NoError(t, err)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer ts.Close()

	core, observedLogs := observer.New(zap.DebugLevel)
	logger := zap.New(core)

	client, err := NewClient(ClintConf{
		Instance: ts.URL,
		Login:    "login",
		Password: "password",
		AutoAuth: true,
		Logger:   logger,
	})
	require.NoError(t, err)

	res, err := client.Reports.Status(reportID)
	require.NoError(t, err)
	assert.Equal(t, DONE, res.Result.ReportStatus)
	assert.Equal(t, 2, authCalls)
	assert.Equal(t, 2, statusCalls)
	assert.Equal(t, 1, observedLogs.FilterMessage("request unauthorized, refreshing token").Len())

	for _, entry := range observedLogs.AllUntimed() {
		assert.NotContains(t, entry.Message, tokenValue)
		assert.NotContains(t, entry.Message, bodyMessage)
		for key, value := range entry.ContextMap() {
			assert.NotContains(t, fmt.Sprint(value), tokenValue, "field %s leaked token", key)
			assert.NotContains(t, fmt.Sprint(value), bodyMessage, "field %s leaked unauthorized body", key)
		}
	}
}

func TestClient_SetToken_FormatsBearer(t *testing.T) {
	client, err := NewClient(ClintConf{
		Instance: "http://example.com",
	})
	require.NoError(t, err)

	client.SetToken("abc")
	assert.Equal(t, "Bearer abc", client.Token)
}

func TestClient_isUnauthorized_NilResponse(t *testing.T) {
	client, err := NewClient(ClintConf{
		Instance: "http://example.com",
	})
	require.NoError(t, err)

	assert.False(t, client.isUnauthorized(nil))
}

func TestClient_isUnauthorized_401StatusCode(t *testing.T) {
	client, err := NewClient(ClintConf{
		Instance: "http://example.com",
	})
	require.NoError(t, err)

	resp := &http.Response{
		StatusCode: http.StatusUnauthorized,
		Body:       io.NopCloser(strings.NewReader("")),
	}
	assert.True(t, client.isUnauthorized(resp))
}

func TestClient_isUnauthorized_BodyContainsErrorMessage(t *testing.T) {
	client, err := NewClient(ClintConf{
		Instance: "http://example.com",
	})
	require.NoError(t, err)

	bodyContent := `{"code":"error","description":"Ошибка получения доступа для указанного токена"}`
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(bodyContent)),
	}

	assert.True(t, client.isUnauthorized(resp))

	// Verify the body is still readable after the call (body rewound).
	rewound, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, bodyContent, string(rewound))
}

func TestClient_isUnauthorized_200OKNormalBody(t *testing.T) {
	client, err := NewClient(ClintConf{
		Instance: "http://example.com",
	})
	require.NoError(t, err)

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("ok")),
	}
	assert.False(t, client.isUnauthorized(resp))
}

func TestClient_AuthInterceptor_SkipsAuthForAuthEndpoint(t *testing.T) {
	var authCalls int

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case URL_AUTH:
			authCalls++
			_, err := fmt.Fprint(w, `{"code":"ok","result":{"token":"test-token"}}`)
			require.NoError(t, err)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer ts.Close()

	client, err := NewClient(ClintConf{
		Instance: ts.URL,
		Login:    "login",
		Password: "password",
		AutoAuth: true,
	})
	require.NoError(t, err)

	// Directly call the auth endpoint via the Auth service.
	token, err := client.Auth.Auth("login", "password")
	require.NoError(t, err)
	assert.Equal(t, "test-token", token)

	// Only 1 auth call: the explicit one. No pre-auth should have happened.
	assert.Equal(t, 1, authCalls)
}

func TestClient_AuthInterceptor_ExhaustsRetries(t *testing.T) {
	const reportID = "report-456"

	var authCalls int
	var statusCalls int

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case URL_AUTH:
			authCalls++
			_, err := fmt.Fprint(w, `{"code":"ok","result":{"token":"valid-token"}}`)
			require.NoError(t, err)
		case fmt.Sprintf(URL_REPORT_STATUS, reportID):
			statusCalls++
			w.WriteHeader(http.StatusUnauthorized)
			_, err := fmt.Fprint(w, `{"code":"unauthorized","description":"always unauthorized"}`)
			require.NoError(t, err)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer ts.Close()

	client, err := NewClient(ClintConf{
		Instance: ts.URL,
		Login:    "login",
		Password: "password",
		AutoAuth: true,
	})
	require.NoError(t, err)

	_, err = client.Reports.Status(reportID)
	assert.Error(t, err)

	// Auth should be called multiple times: initial pre-auth + retry refresh(s).
	assert.GreaterOrEqual(t, authCalls, 2)
	// Status endpoint should be called at least 2 times (initial + retry).
	assert.GreaterOrEqual(t, statusCalls, 2)
}

func TestNewClient_WithLogger(t *testing.T) {
	core, _ := observer.New(zap.DebugLevel)
	logger := zap.New(core)

	client, err := NewClient(ClintConf{
		Instance: "http://example.com",
		Logger:   logger,
	})
	require.NoError(t, err)

	assert.NotNil(t, client.logger)
	assert.NotNil(t, client.Auth)
	assert.NotNil(t, client.Reports)
}
