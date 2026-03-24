package xlog

import (
	"net/http"
	"time"

	httpclient "github.com/germangorelkin/http-client"
	"go.uber.org/zap"
)

// RequestFields extracts standard HTTP request fields for structured logging.
func RequestFields(req *http.Request) []zap.Field {
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

// NewLoggingInterceptor returns an HTTP client interceptor that logs request
// start, completion, and failures using the provided logger.
func NewLoggingInterceptor(logger *zap.Logger) func(*http.Request, httpclient.Handler) (*http.Response, error) {
	return func(req *http.Request, handler httpclient.Handler) (*http.Response, error) {
		start := time.Now()
		log := logger.With(RequestFields(req)...)

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
}
