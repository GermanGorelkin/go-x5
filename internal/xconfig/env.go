// Package xconfig provides helpers for reading typed values from environment
// variables with sensible defaults.
package xconfig

import (
	"fmt"
	"os"
	"strconv"
)

// Int reads an environment variable and parses it as int.
// Returns defaultValue when the variable is empty.
func Int(key string, defaultValue int) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("failed to parse %s=%q: %w", key, value, err)
	}

	return parsed, nil
}

// Bool reads an environment variable and parses it as bool.
// Returns defaultValue when the variable is empty.
func Bool(key string, defaultValue bool) (bool, error) {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue, nil
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("failed to parse %s=%q: %w", key, value, err)
	}

	return parsed, nil
}
