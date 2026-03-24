package insights

import (
	"net/http"
	"time"

	httpclient "github.com/germangorelkin/http-client"
	"go.uber.org/zap"
)

func (c *Client) loggerFor(component string) *zap.Logger {
	if component == "" {
		return c.logger
	}
	return c.logger.Named(component)
}

func (c *Client) requestLogger(req *http.Request) *zap.Logger {
	return c.logger.With(requestFields(req)...)
}

func (c *Client) loggingInterceptor(req *http.Request, handler httpclient.Handler) (*http.Response, error) {
	start := time.Now()
	log := c.requestLogger(req)

	log.Debug("http request started")

	resp, err := handler(req)

	fields := []zap.Field{
		zap.Duration("duration", time.Since(start)),
	}
	if resp != nil {
		fields = append(fields, zap.Int("status_code", resp.StatusCode))
	}
	if err != nil {
		fields = append(fields, zap.Error(err))
		log.Error("http request failed", fields...)
		return resp, err
	}

	log.Debug("http request completed", fields...)
	return resp, nil
}

func requestFields(req *http.Request) []zap.Field {
	if req == nil {
		return nil
	}

	fields := []zap.Field{
		zap.String("method", req.Method),
	}
	if req.URL != nil {
		if req.URL.Host != "" {
			fields = append(fields, zap.String("host", req.URL.Host))
		}
		if req.URL.Path != "" {
			fields = append(fields, zap.String("path", req.URL.Path))
		}
	}

	return fields
}
