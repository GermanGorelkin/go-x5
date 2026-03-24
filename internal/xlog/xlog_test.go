package xlog

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestConfig_DefaultLevelIsInfo(t *testing.T) {
	cfg := Config(false)

	assert.Equal(t, "json", cfg.Encoding)
	assert.Equal(t, zap.InfoLevel, cfg.Level.Level())
}

func TestConfig_VerboseLevelIsDebug(t *testing.T) {
	cfg := Config(true)

	assert.Equal(t, zap.DebugLevel, cfg.Level.Level())
}

func TestNewWithSyncer_JSONOutput(t *testing.T) {
	var buf bytes.Buffer

	logger, err := newWithSyncer("test-command", true, zapcore.AddSync(&buf))
	require.NoError(t, err)

	logger.Info("hello", zap.String("component", "test"))

	var payload map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &payload))

	assert.Equal(t, "hello", payload["msg"])
	assert.Equal(t, "info", payload["level"])
	assert.Equal(t, "test-command", payload["command"])
	assert.Equal(t, "test", payload["component"])
	assert.NotEmpty(t, payload["ts"])
	assert.NotEmpty(t, payload["caller"])
}
