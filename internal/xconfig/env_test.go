package xconfig

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInt_ReturnsDefault_WhenEmpty(t *testing.T) {
	// Env var is not set, so os.Getenv returns "".
	t.Setenv("TEST_INT_EMPTY", "")

	got, err := Int("TEST_INT_EMPTY", 99)
	assert.NoError(t, err)
	assert.Equal(t, 99, got)
}

func TestInt_ReturnsDefault_WhenUnset(t *testing.T) {
	// Deliberately do NOT call t.Setenv – the variable should not exist.
	got, err := Int("TEST_INT_TOTALLY_UNSET", 7)
	assert.NoError(t, err)
	assert.Equal(t, 7, got)
}

func TestInt_ParsesValue(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		want     int
	}{
		{"positive", "42", 42},
		{"zero", "0", 0},
		{"negative", "-5", -5},
		{"large", "100000", 100000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TEST_INT_PARSE", tt.envValue)

			got, err := Int("TEST_INT_PARSE", 1)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestInt_Error_InvalidValue(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
	}{
		{"letters", "abc"},
		{"float", "3.14"},
		{"bool_string", "true"},
		{"mixed", "12abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TEST_INT_INVALID", tt.envValue)

			got, err := Int("TEST_INT_INVALID", 0)
			assert.Error(t, err)
			assert.Equal(t, 0, got)
			assert.Contains(t, err.Error(), "failed to parse")
			assert.Contains(t, err.Error(), tt.envValue)
		})
	}
}

func TestInt_DefaultZero_WhenEmpty(t *testing.T) {
	t.Setenv("TEST_INT_DEF_ZERO", "")

	got, err := Int("TEST_INT_DEF_ZERO", 0)
	assert.NoError(t, err)
	assert.Equal(t, 0, got)
}

func TestBool_ReturnsDefault_WhenEmpty(t *testing.T) {
	t.Setenv("TEST_BOOL_EMPTY", "")

	got, err := Bool("TEST_BOOL_EMPTY", false)
	assert.NoError(t, err)
	assert.False(t, got)
}

func TestBool_ReturnsDefaultTrue_WhenEmpty(t *testing.T) {
	t.Setenv("TEST_BOOL_EMPTY_TRUE", "")

	got, err := Bool("TEST_BOOL_EMPTY_TRUE", true)
	assert.NoError(t, err)
	assert.True(t, got)
}

func TestBool_ReturnsDefault_WhenUnset(t *testing.T) {
	got, err := Bool("TEST_BOOL_TOTALLY_UNSET", true)
	assert.NoError(t, err)
	assert.True(t, got)
}

func TestBool_ParsesTrue(t *testing.T) {
	trueValues := []string{"true", "TRUE", "True", "1", "t", "T"}

	for _, v := range trueValues {
		t.Run(v, func(t *testing.T) {
			t.Setenv("TEST_BOOL_TRUE", v)

			got, err := Bool("TEST_BOOL_TRUE", false)
			assert.NoError(t, err)
			assert.True(t, got)
		})
	}
}

func TestBool_ParsesFalse(t *testing.T) {
	falseValues := []string{"false", "FALSE", "False", "0", "f", "F"}

	for _, v := range falseValues {
		t.Run(v, func(t *testing.T) {
			t.Setenv("TEST_BOOL_FALSE", v)

			got, err := Bool("TEST_BOOL_FALSE", true)
			assert.NoError(t, err)
			assert.False(t, got)
		})
	}
}

func TestBool_Error_InvalidValue(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
	}{
		{"number", "42"},
		{"word", "yes"},
		{"empty_space", " "},
		{"mixed", "truthy"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TEST_BOOL_INVALID", tt.envValue)

			got, err := Bool("TEST_BOOL_INVALID", false)
			assert.Error(t, err)
			assert.False(t, got)
			assert.Contains(t, err.Error(), "failed to parse")
			assert.Contains(t, err.Error(), tt.envValue)
		})
	}
}
