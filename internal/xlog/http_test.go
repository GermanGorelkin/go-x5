package xlog

import (
	"errors"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestRequestFields_NilRequest(t *testing.T) {
	fields := RequestFields(nil)

	assert.Nil(t, fields)
}

func TestRequestFields_WithMethodAndURL(t *testing.T) {
	req := &http.Request{
		Method: http.MethodPost,
		URL: &url.URL{
			Host: "example.com",
			Path: "/api/v1/reports",
		},
	}

	fields := RequestFields(req)

	assert.Len(t, fields, 3)
	assert.Equal(t, zap.String("method", "POST"), fields[0])
	assert.Equal(t, zap.String("host", "example.com"), fields[1])
	assert.Equal(t, zap.String("path", "/api/v1/reports"), fields[2])
}

func TestRequestFields_NoHostNoPath(t *testing.T) {
	req := &http.Request{
		Method: http.MethodGet,
		URL:    &url.URL{},
	}

	fields := RequestFields(req)

	assert.Len(t, fields, 1)
	assert.Equal(t, zap.String("method", "GET"), fields[0])
}

func TestNewLoggingInterceptor_LogsSuccessfulRequest(t *testing.T) {
	core, logs := observer.New(zapcore.DebugLevel)
	logger := zap.New(core)

	interceptor := NewLoggingInterceptor(logger)

	req := &http.Request{
		Method: http.MethodGet,
		URL: &url.URL{
			Host: "example.com",
			Path: "/health",
		},
	}

	handler := func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK}, nil
	}

	resp, err := interceptor(req, handler)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 2, logs.Len())

	started := logs.All()[0]
	assert.Equal(t, "http request started", started.Message)
	assert.Equal(t, zapcore.DebugLevel, started.Level)
	assert.Equal(t, "GET", started.ContextMap()["method"])
	assert.Equal(t, "example.com", started.ContextMap()["host"])
	assert.Equal(t, "/health", started.ContextMap()["path"])

	completed := logs.All()[1]
	assert.Equal(t, "http request completed", completed.Message)
	assert.Equal(t, zapcore.DebugLevel, completed.Level)
	assert.Equal(t, "GET", completed.ContextMap()["method"])
	assert.Contains(t, completed.ContextMap(), "duration")
	assert.Equal(t, int64(200), completed.ContextMap()["status_code"])
}

func TestNewLoggingInterceptor_LogsFailedRequest(t *testing.T) {
	core, logs := observer.New(zapcore.DebugLevel)
	logger := zap.New(core)

	interceptor := NewLoggingInterceptor(logger)

	req := &http.Request{
		Method: http.MethodPost,
		URL: &url.URL{
			Host: "example.com",
			Path: "/submit",
		},
	}

	errExpected := errors.New("connection refused")
	handler := func(r *http.Request) (*http.Response, error) {
		return nil, errExpected
	}

	resp, err := interceptor(req, handler)

	assert.ErrorIs(t, err, errExpected)
	assert.Nil(t, resp)
	assert.Equal(t, 2, logs.Len())

	started := logs.All()[0]
	assert.Equal(t, "http request started", started.Message)
	assert.Equal(t, zapcore.DebugLevel, started.Level)
	assert.Equal(t, "POST", started.ContextMap()["method"])
	assert.Equal(t, "example.com", started.ContextMap()["host"])
	assert.Equal(t, "/submit", started.ContextMap()["path"])

	failed := logs.All()[1]
	assert.Equal(t, "http request failed", failed.Message)
	assert.Equal(t, zapcore.ErrorLevel, failed.Level)
	assert.Equal(t, "POST", failed.ContextMap()["method"])
	assert.Contains(t, failed.ContextMap(), "duration")
	assert.Equal(t, "connection refused", failed.ContextMap()["error"])
}
