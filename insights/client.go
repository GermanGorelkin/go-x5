package insights

import (
	"fmt"

	httpclient "github.com/germangorelkin/http-client"
	"go.uber.org/zap"
)

const (
	URL_BUILD_SECTIONS       = "%s/api/v1/public/dictionaries/report-types/%s/build-sections" // {{api_url}}/api/v1/public/dictionaries/report-types/{{reportTypeId}}/build-sections
	URL_BUILD_AVAILABLE_DATE = "%s/api/v1/public/dictionaries/availableDates?reportTypeId=%s" // {{api_url}}/api/v1/public/dictionaries/availableDates?reportTypeId={{reportTypeId}}
	URL_TREE_STORES          = "%s/api/v1/public/tree/stores?reportTypeId=%s"                 // {{api_url}}/api/v1/public/tree/stores?reportTypeId={{reportTypeId}}
	URL_TREE_PRODUCTS        = "%s/api/v1/public/tree/products?reportTypeId=%s"               // {{api_url}}/api/v1/public/tree/products?reportTypeId={{reportTypeId}}
	URL_DELIVERY             = "%s/api/v1/public/dictionaries/delivery"                       // {{api_url}}/api/v1/public/dictionaries/delivery
	URL_METRICS              = "%s/api/v1/public/dictionaries/report-types/trends/metrics"    // {{api_url}}/api/v1/public/dictionaries/report-types/trends/metrics
	URL_GRANULARITIES        = "%s/api/v1/public/dictionaries/periods?reportTypeId=%s"        // {{api_url}}/api/v1/public/dictionaries/periods?reportTypeId={{reportTypeId}}
	URL_PRODUCTS_EXPORT      = "%s/api/v1/public/tree/products/download"                      // {{api_url}}/api/v1/public/tree/products/download

	URL_CREATE_TRENDS = "%s/api/v1/public/reports/trends" // {{api_url}}/api/v1/public/reports/trends
	URL_REPORT_STATUS = "%s/api/v2/public/reports/%s"     // {{api_url}}/api/v2/public/reports/{{reportId}}
	URL_REPORT_EXPORT = "%s/api/v1/public/export/%s"      // {{api_url}}/api/v1/public/export/{{exportFileId}}

	URL_KC_TOKEN       = "%s/auth/realms/%s/protocol/openid-connect/token" // {{kc_url}}/auth/realms/{{kc_realm}}/protocol/openid-connect/token
	URL_INTERNAL_TOKEN = "%s/api/v1/public/auth/token"                     // {{api_url}}/api/v1/public/auth/token
)

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
}

type service struct {
	client *Client
}

type ClintConf struct {
	KC_URL, KC_RELM           string
	ClientID, Login, Password string
	API_URL                   string
	Verbose                   bool
	Logger                    *zap.Logger
}

func NewClient(cfg ClintConf) (*Client, error) {
	cl, err := httpclient.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build http-client:%w", err)
	}
	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	c := &Client{
		KC_URL:     cfg.KC_URL,
		KC_RELM:    cfg.KC_RELM,
		API_URL:    cfg.API_URL,
		ClientID:   cfg.ClientID,
		Login:      cfg.Login,
		Password:   cfg.Password,
		httpClient: cl,
		logger: logger.Named("insights").With(
			zap.String("api_url", cfg.API_URL),
			zap.String("kc_realm", cfg.KC_RELM),
		),
	}
	if err := c.httpClient.AddInterceptor(c.loggingInterceptor); err != nil {
		return nil, fmt.Errorf("failed to add logging interceptor: %w", err)
	}

	c.common.client = c
	c.Auth = (*AuthService)(&c.common)
	c.Parameters = (*ParametersService)(&c.common)
	c.Reports = (*ReportService)(&c.common)
	c.logger.Debug("client initialized",
		zap.Bool("verbose", cfg.Verbose),
		zap.String("client_id", cfg.ClientID),
	)

	return c, nil
}

// SetToken sets the client's token
func (c *Client) SetToken(access, refresh, jwt string) {
	// cookie := fmt.Sprintf("kc-access=%s; kc-state=%s;", access, refresh)
	// c.httpClient.SetHeader("cookie", cookie)
	c.httpClient.SetHeader("Authorization", fmt.Sprintf("Bearer %s", access))
	c.httpClient.SetHeader("x5-api-key", jwt)
	c.logger.Debug("authorization headers updated")
}

// Authorization full authorizations in the system
func (c *Client) Authorization() error {
	log := c.loggerFor("auth")
	log.Info("starting authorization flow")

	access, refresh, err := c.Auth.GetKeyCloakTokens(c.ClientID, c.Login, c.Password)
	if err != nil {
		log.Error("failed to get keycloak tokens", zap.Error(err))
		return fmt.Errorf("failed to get keycloak tokens:%w", err)
	}

	jwt, err := c.Auth.GetInternalToken(access, refresh)
	if err != nil {
		log.Error("failed to get internal token", zap.Error(err))
		return fmt.Errorf("failed to get internal token:%w", err)
	}

	c.SetToken(string(access), string(refresh), string(jwt))
	log.Info("authorization flow completed")

	return nil
}
