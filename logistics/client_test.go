package logistics

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
