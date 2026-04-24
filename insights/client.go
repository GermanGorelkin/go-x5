// Package insights provides an HTTP client for the X5 Insights (analytics) API.
// It handles authorization via KeyCloak + internal JWT, report parameter fetching,
// and trends analysis report creation, polling, and download.
package insights

import (
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/germangorelkin/go-x5/internal/xlog"
	httpclient "github.com/germangorelkin/http-client"
	"go.uber.org/zap"
)

const (
	// URL_BUILD_SECTIONS fetches the list of report build-sections for a given report type.
	URL_BUILD_SECTIONS = "%s/api/v1/public/dictionaries/report-types/%s/build-sections" // {{api_url}}/api/v1/public/dictionaries/report-types/{{reportTypeId}}/build-sections
	// URL_BUILD_AVAILABLE_DATE fetches the min/max available dates for a given report type.
	URL_BUILD_AVAILABLE_DATE = "%s/api/v1/public/dictionaries/availableDates?reportTypeId=%s" // {{api_url}}/api/v1/public/dictionaries/availableDates?reportTypeId={{reportTypeId}}
	// URL_TREE_STORES fetches the hierarchical store classifier tree (networks → districts → regions → cities).
	URL_TREE_STORES = "%s/api/v1/public/tree/stores?reportTypeId=%s" // {{api_url}}/api/v1/public/tree/stores?reportTypeId={{reportTypeId}}
	// URL_TREE_PRODUCTS fetches the hierarchical product classifier tree.
	URL_TREE_PRODUCTS = "%s/api/v1/public/tree/products?reportTypeId=%s" // {{api_url}}/api/v1/public/tree/products?reportTypeId={{reportTypeId}}
	// URL_DELIVERY fetches the list of available delivery types.
	URL_DELIVERY = "%s/api/v1/public/dictionaries/delivery" // {{api_url}}/api/v1/public/dictionaries/delivery
	// URL_METRICS fetches the available metric groups for trends reports.
	URL_METRICS = "%s/api/v1/public/dictionaries/report-types/trends/metrics" // {{api_url}}/api/v1/public/dictionaries/report-types/trends/metrics
	// URL_GRANULARITIES fetches the available period granularities (e.g. week, month) for a report type.
	URL_GRANULARITIES = "%s/api/v1/public/dictionaries/periods?reportTypeId=%s" // {{api_url}}/api/v1/public/dictionaries/periods?reportTypeId={{reportTypeId}}
	// URL_PRODUCTS_EXPORT triggers a product tree export/download as a file.
	URL_PRODUCTS_EXPORT = "%s/api/v1/public/tree/products/download" // {{api_url}}/api/v1/public/tree/products/download

	// URL_CREATE_TRENDS creates a new trends analysis report.
	URL_CREATE_TRENDS = "%s/api/v1/public/reports/trends" // {{api_url}}/api/v1/public/reports/trends
	// URL_REPORT_STATUS polls the current status of a report by its ID.
	URL_REPORT_STATUS = "%s/api/v2/public/reports/%s" // {{api_url}}/api/v2/public/reports/{{reportId}}
	// URL_REPORT_EXPORT downloads the generated export file by its export file ID.
	URL_REPORT_EXPORT = "%s/api/v1/public/export/%s" // {{api_url}}/api/v1/public/export/{{exportFileId}}

	// URL_KC_TOKEN obtains access and refresh tokens from the KeyCloak OpenID Connect endpoint.
	URL_KC_TOKEN = "%s/auth/realms/%s/protocol/openid-connect/token" // {{kc_url}}/auth/realms/{{kc_realm}}/protocol/openid-connect/token
	// URL_INTERNAL_TOKEN exchanges a KeyCloak access token for an internal X5 API JWT.
	URL_INTERNAL_TOKEN = "%s/api/v1/public/auth/token" // {{api_url}}/api/v1/public/auth/token

	keycloakExpirySkew = 30 * time.Second
)

// Client is the top-level API client for the X5 Insights service.
// It holds KeyCloak and API credentials, an HTTP transport, and
// sub-service handles for authentication, parameter fetching, and report operations.
type Client struct {
	KC_URL, KC_RELM           string
	ClientID, Login, Password string

	API_URL string

	Auth       *AuthService
	Parameters *ParametersService
	Reports    *ReportService

	httpClient *httpclient.Client
	logger     *zap.Logger
	common     service // Reuse a single struct instead of allocating one for each service on the heap.

	authCache *AuthCache
}

// service is the shared base struct embedded into every API sub-service
// so they can access the parent Client without extra allocations.
type service struct {
	client *Client
}

type authState struct {
	access           AccessToken
	refresh          RefreshToken
	jwt              JWTToken
	accessExpiresAt  time.Time
	refreshExpiresAt time.Time
}

// AuthCache stores authorization tokens that can be shared by several clients
// using the same credentials.
type AuthCache struct {
	mu    sync.Mutex
	state authState
}

// NewAuthCache creates an empty shared authorization token cache.
func NewAuthCache() *AuthCache {
	return &AuthCache{}
}

// ClintConf holds the configuration parameters required to construct a new Client.
// All fields except Logger are mandatory; when Logger is nil a no-op logger is used.
type ClintConf struct {
	KC_URL, KC_RELM           string
	ClientID, Login, Password string
	API_URL                   string
	Logger                    *zap.Logger
	AuthCache                 *AuthCache
}

// NewClient builds a fully initialised Client from the supplied configuration.
// It creates the underlying HTTP client, attaches a logging interceptor, and
// wires up the Auth, Parameters, and Reports sub-services.
func NewClient(cfg ClintConf) (*Client, error) {
	// Create the underlying HTTP client (nil uses the default http.Client).
	cl, err := httpclient.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build http-client:%w", err)
	}

	// Fall back to a no-op logger when none is provided.
	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	authCache := cfg.AuthCache
	if authCache == nil {
		authCache = NewAuthCache()
	}

	// Populate the core client fields from configuration.
	c := &Client{
		KC_URL:     cfg.KC_URL,
		KC_RELM:    cfg.KC_RELM,
		API_URL:    cfg.API_URL,
		ClientID:   cfg.ClientID,
		Login:      cfg.Login,
		Password:   cfg.Password,
		httpClient: cl,
		authCache:  authCache,
		logger: logger.Named("insights").With(
			zap.String("api_url", cfg.API_URL),
			zap.String("kc_realm", cfg.KC_RELM),
		),
	}

	// Register a logging interceptor so every HTTP request/response is traced.
	if err := c.httpClient.AddInterceptor(xlog.NewLoggingInterceptor(c.logger)); err != nil {
		return nil, fmt.Errorf("failed to add logging interceptor: %w", err)
	}

	// Wire up sub-services using the shared "common" struct to avoid per-service heap allocations.
	c.common.client = c
	c.Auth = (*AuthService)(&c.common)
	c.Parameters = (*ParametersService)(&c.common)
	c.Reports = (*ReportService)(&c.common)
	c.logger.Debug("client initialized",
		zap.String("client_id", cfg.ClientID),
	)

	return c, nil
}

// SetToken sets the client's authorization headers used for all subsequent API requests.
// access is the KeyCloak access token, refresh is unused at header level but kept for symmetry,
// and jwt is the internal X5 API key.
func (c *Client) SetToken(access, refresh, jwt string) {
	c.httpClient.SetHeader("Authorization", fmt.Sprintf("Bearer %s", access))
	c.httpClient.SetHeader("x5-api-key", jwt)
	c.logger.Debug("authorization headers updated")
}

// Authorization performs the full authorization flow: it obtains KeyCloak tokens
// via username/password grant, exchanges them for an internal JWT, and stores
// all tokens in the client's HTTP headers for subsequent requests.
func (c *Client) Authorization() error {
	log := c.loggerFor("auth")
	log.Info("starting authorization flow")

	c.authCache.mu.Lock()
	defer c.authCache.mu.Unlock()

	access, refresh, err := c.ensureKeyCloakTokens(log)
	if err != nil {
		log.Error("failed to get keycloak tokens", zap.Error(err))
		return fmt.Errorf("failed to get keycloak tokens:%w", err)
	}

	jwt := c.authCache.state.jwt
	if jwt == "" {
		access, refresh, jwt, err = c.exchangeInternalToken(log, access, refresh)
		if err != nil {
			log.Error("failed to get internal token", zap.Error(err))
			return fmt.Errorf("failed to get internal token:%w", err)
		}
		c.authCache.state.jwt = jwt
	} else {
		log.Debug("reusing cached internal token")
	}

	c.SetToken(string(access), string(refresh), string(jwt))
	log.Info("authorization flow completed")

	return nil
}

func (c *Client) exchangeInternalToken(log *zap.Logger, access AccessToken, refresh RefreshToken) (AccessToken, RefreshToken, JWTToken, error) {
	jwt, err := c.Auth.GetInternalToken(access, refresh)
	if err == nil {
		return access, refresh, jwt, nil
	}
	if !isAuthorizationFailure(err) {
		return "", "", "", err
	}

	log.Warn("internal token exchange rejected keycloak access token, retrying with password grant", zap.Error(err))
	res, fallbackErr := c.Auth.getKeyCloakTokensWithPassword(c.ClientID, c.Login, c.Password)
	if fallbackErr != nil {
		return "", "", "", fmt.Errorf("password grant fallback failed after internal token rejection: %w", fallbackErr)
	}

	c.storeKeyCloakTokens(res, time.Now())
	jwt, err = c.Auth.GetInternalToken(res.AccessToken, res.RefreshToken)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to get internal token after password grant fallback: %w", err)
	}

	return res.AccessToken, res.RefreshToken, jwt, nil
}

func (c *Client) ensureKeyCloakTokens(log *zap.Logger) (AccessToken, RefreshToken, error) {
	now := time.Now()
	if c.authCache.state.hasValidAccess(now) {
		log.Debug("reusing cached keycloak access token")
		return c.authCache.state.access, c.authCache.state.refresh, nil
	}

	if c.authCache.state.canRefresh(now) {
		log.Info("refreshing keycloak tokens")
		res, err := c.Auth.refreshKeyCloakTokens(c.ClientID, c.authCache.state.refresh)
		if err == nil {
			c.storeKeyCloakTokens(res, now)
			return res.AccessToken, res.RefreshToken, nil
		}
		log.Warn("failed to refresh keycloak tokens, falling back to password grant", zap.Error(err))
	}

	log.Info("requesting keycloak tokens with password grant")
	res, err := c.Auth.getKeyCloakTokensWithPassword(c.ClientID, c.Login, c.Password)
	if err != nil {
		return "", "", err
	}

	c.storeKeyCloakTokens(res, now)
	return res.AccessToken, res.RefreshToken, nil
}

func (c *Client) storeKeyCloakTokens(res ResponseKeyCloakTokens, now time.Time) {
	c.authCache.state = authState{
		access:           res.AccessToken,
		refresh:          res.RefreshToken,
		accessExpiresAt:  tokenExpiryTime(now, res.ExpiresIn),
		refreshExpiresAt: tokenExpiryTime(now, res.RefreshExpiresIn),
	}
}

func (s authState) hasValidAccess(now time.Time) bool {
	if s.access == "" || s.accessExpiresAt.IsZero() {
		return false
	}
	return now.Before(s.accessExpiresAt.Add(-keycloakExpirySkew))
}

func (s authState) canRefresh(now time.Time) bool {
	if s.refresh == "" || s.refreshExpiresAt.IsZero() {
		return false
	}
	return now.Before(s.refreshExpiresAt.Add(-keycloakExpirySkew))
}

func tokenExpiryTime(now time.Time, expiresInSec int) time.Time {
	if expiresInSec <= 0 {
		return now
	}
	return now.Add(time.Duration(expiresInSec) * time.Second)
}

func isAuthorizationFailure(err error) bool {
	var resErr *httpclient.ErrorResponse
	if !errors.As(err, &resErr) || resErr.Response == nil {
		return false
	}

	return resErr.Response.StatusCode == http.StatusUnauthorized ||
		resErr.Response.StatusCode == http.StatusForbidden
}
