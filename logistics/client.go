// Package logistics provides an HTTP client for the X5 Logistics API.
// It supports report creation, status polling, and report download,
// with optional automatic authentication and token refresh.
package logistics

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/germangorelkin/go-x5/internal/xlog"
	httpclient "github.com/germangorelkin/http-client"
	"go.uber.org/zap"
)

const (
	// URL_REPORT_DOWNLOAD is the endpoint for downloading a report part by its ID.
	URL_REPORT_DOWNLOAD = "/v2/logistics/report/%s/download"
	// URL_REPORT_STATUS is the endpoint for checking a report's generation status by its ID.
	URL_REPORT_STATUS = "/v2/logistics/report/%s/status"
	// URL_REPORT_CREATE is the endpoint for creating (requesting) a new report.
	URL_REPORT_CREATE = "/v2/logistics/report"
	// URL_AUTH is the endpoint for obtaining an authentication token.
	URL_AUTH = "/v2/logistics/auth"
)

// Client is the top-level X5 Logistics API client. It holds connection
// credentials, the current bearer token, and exposes domain-specific
// sub-services (Auth and Reports).
//
// Auth and Reports are thin wrappers around a shared "common" service
// struct that points back to this Client. This avoids allocating a
// separate struct for each service on the heap while still providing
// a convenient, grouped API surface.
type Client struct {
	// Instance is the base URL of the X5 Logistics API (e.g. "https://api.x5.ru").
	Instance string
	// Login and Password are the credentials used for authentication.
	Login, Password string
	// Token is the current Bearer token set after a successful auth call.
	Token string

	// Auth provides methods for obtaining authentication tokens.
	Auth *AuthService
	// Reports provides methods for creating, polling, and downloading reports.
	Reports *ReportService

	// httpClient is the underlying HTTP client used for all API requests.
	httpClient *httpclient.Client
	// logger is the structured logger scoped to the logistics package.
	logger *zap.Logger
	// common is a shared service value reused by AuthService and ReportService
	// so that each sub-service can access the parent Client without a separate allocation.
	common service // Reuse a single struct instead of allocating one for each service on the heap.
}

// service is the base type embedded (via type conversion) into every
// domain-specific service. It carries a back-pointer to the owning Client.
type service struct {
	client *Client
}

// ClintConf holds configuration options for constructing a new Client.
type ClintConf struct {
	Instance        string
	Login, Password string
	AutoAuth        bool
	Logger          *zap.Logger
}

// NewClient creates and configures a new logistics API Client.
//
// Setup steps:
//  1. Build the underlying HTTP client with the given base URL.
//  2. Attach a structured logging interceptor so every request/response is logged.
//  3. If AutoAuth is enabled, attach the AuthInterceptor which transparently
//     obtains and refreshes Bearer tokens on 401 responses.
//  4. Wire up the Auth and Reports sub-services via the shared common struct.
func NewClient(cfg ClintConf) (*Client, error) {
	// Create the underlying HTTP client; passing nil uses http.DefaultClient.
	cl, err := httpclient.New(
		// TODO timeout
		// &http.Client{Timeout: 30 * time.Second},
		nil,
		httpclient.WithBaseURL(cfg.Instance))
	if err != nil {
		return nil, fmt.Errorf("failed to build http-client:%w", err)
	}

	// Fall back to a no-op logger when none is provided.
	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	c := &Client{
		Instance:   cfg.Instance,
		Login:      cfg.Login,
		Password:   cfg.Password,
		httpClient: cl,
		logger: logger.Named("logistics").With(
			zap.String("instance", cfg.Instance),
		),
	}

	// Register the logging interceptor so every outgoing request and incoming
	// response is recorded via structured logging.
	if err := c.httpClient.AddInterceptor(xlog.NewLoggingInterceptor(c.logger)); err != nil {
		return nil, fmt.Errorf("failed to add logging interceptor: %w", err)
	}

	// When AutoAuth is enabled, register the auth interceptor that will
	// automatically obtain a token before the first request and refresh it
	// whenever a 401 response is received.
	if cfg.AutoAuth {
		if err := c.httpClient.AddInterceptor(c.AuthInterceptor); err != nil {
			return nil, fmt.Errorf("failed to add auth interceptor: %w", err)
		}
	}

	// Wire up the shared service so Auth and Reports both point back to this Client.
	c.common.client = c
	c.Auth = (*AuthService)(&c.common)
	c.Reports = (*ReportService)(&c.common)
	c.logger.Debug("client initialized",
		zap.Bool("auto_auth", cfg.AutoAuth),
	)

	return c, nil
}

// SetToken stores the given raw token as a Bearer token on the client and
// updates the underlying HTTP client's Authorization header.
func (c *Client) SetToken(token string) {
	c.Token = fmt.Sprintf("%s %s", "Bearer", token)
	c.httpClient.SetAuthorization(c.Token)
	c.logger.Debug("authorization token updated")
}

// auth performs a full authentication flow using the client's Login and Password,
// then stores the resulting token via SetToken.
func (c *Client) auth() error {
	log := c.loggerFor("auth")
	log.Info("requesting access token")
	token, err := c.Auth.Auth(c.Login, c.Password)
	if err != nil {
		log.Error("failed to get access token", zap.Error(err))
		return err
	}
	c.SetToken(token)
	log.Info("access token updated")
	return nil
}

// isUnauthorized inspects an HTTP response to decide whether the server
// rejected the request due to missing or invalid credentials. It checks
// both the 401 status code and a known Russian-language error message
// that the X5 API may return in the body.
func (c *Client) isUnauthorized(r *http.Response) bool {
	if r == nil {
		return false
	}
	if r.StatusCode == 401 {
		return true
	}

	body, _ := io.ReadAll(r.Body)
	r.ContentLength = int64(len(body))
	r.Body = io.NopCloser(bytes.NewBuffer(body))

	return bytes.Contains(body, []byte("Ошибка получения доступа для указанного токена"))
}

// AuthInterceptor is an HTTP client interceptor that transparently handles
// authentication. It ensures a valid Bearer token is present before sending
// a request and retries once with a fresh token when the server responds
// with an authorization error (e.g. expired token).
func (c *Client) AuthInterceptor(req *http.Request, handler httpclient.Handler) (resp *http.Response, err error) {
	log := c.logger.With(xlog.RequestFields(req)...).Named("auth")

	// If no token is set and this is not an auth request, obtain a token first.
	if req.Header.Get("Authorization") == "" && req.URL.Path != URL_AUTH {
		log.Debug("authorization header missing, authorizing request")
		if err = c.auth(); err != nil {
			log.Error("pre-request authorization failed", zap.Error(err))
			return nil, err
		}
		req.Header.Set("Authorization", c.Token)
	}

	const maxAttempts = 2
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if attempt > 1 {
			if err = rewindRequestBody(req); err != nil {
				log.Error("failed to rewind request body", zap.Error(err), zap.Int("attempt", attempt))
				return resp, err
			}
		}

		resp, err = handler(req)

		// If the response is not an authorization error, return immediately.
		if ok := c.isUnauthorized(resp); !ok {
			return resp, err
		}

		// If the auth endpoint itself returned unauthorized, give up.
		if req.URL.Path == URL_AUTH {
			log.Warn("authorization endpoint returned unauthorized", zap.Int("attempt", attempt))
			return resp, err
		}

		if attempt == maxAttempts {
			break
		}

		log.Warn("request unauthorized, refreshing token",
			zap.Int("attempt", attempt),
			zap.Int("max_attempts", maxAttempts),
		)

		// Token expired — refresh and retry.
		if err = c.auth(); err != nil {
			log.Error("failed to refresh token", zap.Error(err), zap.Int("attempt", attempt))
			return resp, err
		}
		req.Header.Set("Authorization", c.Token)
	}

	log.Error("request remained unauthorized after retries", zap.Int("max_attempts", maxAttempts))
	return resp, err
}
