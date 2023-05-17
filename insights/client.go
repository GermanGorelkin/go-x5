package insights

import (
	"fmt"

	httpclient "github.com/germangorelkin/http-client"
)

const (
	URL_BUILD_SECTIONS       = "%s/api/v1/public/dictionaries/report-types/%s/build-sections" // {{api_url}}/api/v1/public/dictionaries/report-types/{{reportTypeId}}/build-sections
	URL_BUILD_AVAILABLE_DATE = "%s/api/v1/public/dictionaries/availableDates?reportTypeId=%s" // {{api_url}}/api/v1/public/dictionaries/availableDates?reportTypeId={{reportTypeId}}
	URL_TREE_STORES          = "%s/api/v1/public/tree/stores?reportTypeId=%s"                 // {{api_url}}/api/v1/public/tree/stores?reportTypeId={{reportTypeId}}
	URL_TREE_PRODUCTS        = "%s/api/v1/public/tree/products?reportTypeId=%s"               // {{api_url}}/api/v1/public/tree/products?reportTypeId={{reportTypeId}}

	URL_KC_TOKEN       = "%s/auth/realms/%s/protocol/openid-connect/token" // {{kc_url}}/auth/realms/{{kc_realm}}/protocol/openid-connect/token
	URL_INTERNAL_TOKEN = "%s/api/v1/public/auth/token"                     // {{api_url}}/api/v1/public/auth/token
)

type Client struct {
	KC_URL, KC_RELM           string
	ClientID, Login, Password string

	API_URL string

	Auth       *AuthService
	Parameters *ParametersService
	//Reports *ReportService

	httpClient *httpclient.Client
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
}

func NewClient(cfg ClintConf) (*Client, error) {
	cl, err := httpclient.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build http-client:%w", err)
	}
	if cfg.Verbose {
		if err := cl.AddInterceptor(httpclient.DumpInterceptor); err != nil {
			return nil, err
		}
	}

	c := &Client{
		KC_URL:     cfg.KC_URL,
		KC_RELM:    cfg.KC_RELM,
		API_URL:    cfg.API_URL,
		ClientID:   cfg.ClientID,
		Login:      cfg.Login,
		Password:   cfg.Password,
		httpClient: cl,
	}

	c.common.client = c
	c.Auth = (*AuthService)(&c.common)
	c.Parameters = (*ParametersService)(&c.common)
	//c.Reports = (*ReportService)(&c.common)

	return c, nil
}

// SetToken sets the client's token
func (c *Client) SetToken(access, refresh, jwt string) {
	cookie := fmt.Sprintf("kc-access=%s; kc-state=%s;", access, refresh)
	c.httpClient.SetHeader("cookie", cookie)
	c.httpClient.SetHeader("x5-api-key", jwt)
}

// Authorization full authorizations in the system
func (c *Client) Authorization() error {
	access, refresh, err := c.Auth.GetKeyCloakTokens(c.ClientID, c.Login, c.Password)
	if err != nil {
		return fmt.Errorf("failed to get keycloak tokens:%w", err)
	}

	jwt, err := c.Auth.GetInternalToken(access, refresh)
	if err != nil {
		return fmt.Errorf("failed to get internal token:%w", err)
	}

	c.SetToken(string(access), string(refresh), string(jwt))

	return nil
}
