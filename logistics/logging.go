package logistics

import (
	"fmt"
	"net/http"

	"go.uber.org/zap"
)

func (c *Client) loggerFor(component string) *zap.Logger {
	if component == "" {
		return c.logger
	}
	return c.logger.Named(component)
}

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
