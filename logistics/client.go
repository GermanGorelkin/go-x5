package logistics

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	httpclient "github.com/germangorelkin/http-client"
	"go.uber.org/zap"
)

const (
	URL_REPORT_DOWNLOAD = "/v2/logistics/report/%s/download"
	URL_REPORT_STATUS   = "/v2/logistics/report/%s/status"
	URL_REPORT_CREATE   = "/v2/logistics/report"
	URL_AUTH            = "/v2/logistics/auth"
)

type Client struct {
	Instance        string
	Login, Password string
	Token           string

	Auth    *AuthService
	Reports *ReportService

	httpClient *httpclient.Client
	logger     *zap.Logger
	common     service // Reuse a single struct instead of allocating one for each service on the heap.
}

type service struct {
	client *Client
}

type ClintConf struct {
	Instance        string
	Login, Password string
	Verbose         bool
	AutoAuth        bool
	Logger          *zap.Logger
}

func NewClient(cfg ClintConf) (*Client, error) {
	cl, err := httpclient.New(
		// TODO timeout
		// &http.Client{Timeout: 30 * time.Second},
		nil,
		httpclient.WithBaseURL(cfg.Instance))
	if err != nil {
		return nil, fmt.Errorf("failed to build http-client:%w", err)
	}
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

	if cfg.AutoAuth {
		if err := c.httpClient.AddInterceptor(c.AuthInterceptor); err != nil {
			return nil, fmt.Errorf("failed to add auth interceptor: %w", err)
		}
	}
	if err := c.httpClient.AddInterceptor(c.loggingInterceptor); err != nil {
		return nil, fmt.Errorf("failed to add logging interceptor: %w", err)
	}

	c.common.client = c
	c.Auth = (*AuthService)(&c.common)
	c.Reports = (*ReportService)(&c.common)
	c.logger.Debug("client initialized",
		zap.Bool("auto_auth", cfg.AutoAuth),
		zap.Bool("verbose", cfg.Verbose),
	)

	return c, nil
}

func (c *Client) SetToken(token string) {
	c.Token = fmt.Sprintf("%s %s", "Bearer", token)
	c.httpClient.SetAuthorization(c.Token)
	c.logger.Debug("authorization token updated")
}

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

func (c *Client) AuthInterceptor(req *http.Request, handler httpclient.Handler) (resp *http.Response, err error) {
	log := c.requestLogger(req).Named("auth")

	// если нет токена и запрос НЕ на его получения, тогда сначала запршиваем токен
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

		// если ошибка НЕ в авторизации, тогда выходим
		if ok := c.isUnauthorized(resp); !ok {
			return resp, err
		}

		// если ошибка в авторизации при запросе токена, тогда выходим
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

		// если ошибка в авторизации(протух токен) тогда запрашиваем токен и повторяем попытку
		if err = c.auth(); err != nil {
			log.Error("failed to refresh token", zap.Error(err), zap.Int("attempt", attempt))
			return resp, err
		}
		req.Header.Set("Authorization", c.Token)
	}

	log.Error("request remained unauthorized after retries", zap.Int("max_attempts", maxAttempts))
	return resp, err
}
