package logistics

import (
	"fmt"
	"net/http"

	"go.uber.org/zap"
)

// loggerFor returns a child logger scoped to the given component name.
// If component is empty the base client logger is returned unchanged.
func (c *Client) loggerFor(component string) *zap.Logger {
	if component == "" {
		return c.logger
	}
	return c.logger.Named(component)
}

// rewindRequestBody resets the request body to its original state so the
// request can be retried. It uses req.GetBody (set automatically by the
// standard library for common body types) to obtain a fresh io.ReadCloser.
// If the request, body, or GetBody function is nil the call is a no-op.
func rewindRequestBody(req *http.Request) error {
	if req == nil || req.Body == nil || req.GetBody == nil {
		return nil
	}

	body, err := req.GetBody()
	if err != nil {
		return fmt.Errorf("failed to clone request body: %w", err)
	}
	req.Body = body

	return nil
}
