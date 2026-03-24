package insights

import "go.uber.org/zap"

func (c *Client) loggerFor(component string) *zap.Logger {
	if component == "" {
		return c.logger
	}
	return c.logger.Named(component)
}
