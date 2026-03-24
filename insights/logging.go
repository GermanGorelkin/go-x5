package insights

import "go.uber.org/zap"

// loggerFor returns a named child logger for the given component.
// If component is empty, the base client logger is returned unchanged.
func (c *Client) loggerFor(component string) *zap.Logger {
	if component == "" {
		return c.logger
	}
	return c.logger.Named(component)
}
