package logistics

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestLoggerFor_EmptyComponent_ReturnsBaseLogger(t *testing.T) {
	core, _ := observer.New(zap.DebugLevel)
	logger := zap.New(core)
	c := &Client{logger: logger}

	got := c.loggerFor("")

	assert.Same(t, logger, got, "loggerFor with empty component should return the base logger")
}

func TestLoggerFor_WithComponent_ReturnsNamedLogger(t *testing.T) {
	core, observed := observer.New(zap.DebugLevel)
	logger := zap.New(core)
	c := &Client{logger: logger}

	named := c.loggerFor("auth")
	named.Info("test message")

	require.Equal(t, 1, observed.Len(), "expected exactly one log entry")
	entry := observed.All()[0]
	assert.Equal(t, "auth", entry.LoggerName, "expected logger name to include the component")
}

func TestRewindRequestBody_NilRequest(t *testing.T) {
	err := rewindRequestBody(nil)
	assert.NoError(t, err)
}

func TestRewindRequestBody_NilBody(t *testing.T) {
	req := &http.Request{}
	err := rewindRequestBody(req)
	assert.NoError(t, err)
}

func TestRewindRequestBody_NilGetBody(t *testing.T) {
	req := &http.Request{
		Body: io.NopCloser(strings.NewReader("data")),
	}
	// GetBody is nil by default on a bare http.Request
	require.Nil(t, req.GetBody)

	err := rewindRequestBody(req)
	assert.NoError(t, err)
}

func TestRewindRequestBody_Success(t *testing.T) {
	const payload = "hello"

	req, err := http.NewRequest("POST", "http://example.com", strings.NewReader(payload))
	require.NoError(t, err)

	// Drain the body so it's empty.
	drained, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	assert.Equal(t, payload, string(drained))

	// After draining, reading again should yield nothing.
	empty, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	assert.Empty(t, empty)

	// Rewind should restore the body.
	err = rewindRequestBody(req)
	require.NoError(t, err)

	restored, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	assert.Equal(t, payload, string(restored))
}

func TestRewindRequestBody_GetBodyError(t *testing.T) {
	errGetBody := errors.New("clone failed")

	req := &http.Request{
		Body: io.NopCloser(strings.NewReader("data")),
		GetBody: func() (io.ReadCloser, error) {
			return nil, errGetBody
		},
	}

	err := rewindRequestBody(req)
	require.Error(t, err)
	assert.ErrorIs(t, err, errGetBody)
	assert.Contains(t, err.Error(), "failed to clone request body")
}
