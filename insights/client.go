package insights

import (
	"fmt"

	httpclient "github.com/germangorelkin/http-client"
)

const (
	URL_REPORT_DOWNLOAD = "/v1/logistics/report/%s/download"
	URL_REPORT_STATUS   = "/v1/logistics/report/%s/status"
	URL_REPORT_CREATE   = "/v1/logistics/report"
	URL_AUTH            = "/v1/logistics/auth"

	URL_KC_TOKEN       = "%s/auth/realms/%s/protocol/openid-connect/token" // {{kc_url}}/auth/realms/{{kc_realm}}/protocol/openid-connect/token
	URL_INTERNAL_TOKEN = "%s/api/v1/public/auth/token"                     // {{api_url}}/api/v1/public/auth/token
)

type Client struct {
	KC_URL, KC_RELM           string
	ClientID, Login, Password string

	API_URL string

	Auth *AuthService
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
	AutoAuth                  bool
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

	// if cfg.AutoAuth {
	// 	c.httpClient.AddInterceptor(c.AuthInterceptor)
	// }

	c.common.client = c
	c.Auth = (*AuthService)(&c.common)
	//c.Reports = (*ReportService)(&c.common)

	return c, nil
}

func (c *Client) SetToken(access, refresh, jwt string) {
	cookie := fmt.Sprintf("kc-access=%s; kc-state=%s;", access, refresh)
	c.httpClient.SetHeader("cookie", cookie)
	c.httpClient.SetHeader("x5-api-key", jwt)
}

/*
func (c *Client) auth() error {
	log.Printf("%s", "get auth token...")
	token, err := c.Auth.Auth(c.Login, c.Password)
	if err != nil {
		return err
	}
	c.SetToken(token)
	log.Printf("new token:%s", token)
	return nil
}

func (c *Client) isUnauthorized(r *http.Response) bool {
	if r.StatusCode == 401 {
		return true
	}

	body, _ := ioutil.ReadAll(r.Body)
	r.ContentLength = int64(len(body))
	r.Body = ioutil.NopCloser(bytes.NewBuffer(body))

	if bytes.Contains(body, []byte("Ошибка получения доступа для указанного токена")) {
		log.Printf("Unauthorized:%s", body)
		return true
	}

	return false
}

func (c *Client) AuthInterceptor(req *http.Request, handler httpclient.Handler) (resp *http.Response, err error) {
	if req.Header.Get("Authorization") == "" && req.URL.Path != URL_AUTH {
		if err = c.auth(); err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", c.Token)
	}

	attempt := 2 // TODO
	for i := 0; i < attempt; i++ {
		resp, err = handler(req)

		if ok := c.isUnauthorized(resp); !ok {
			break
		}

		if err != nil {
			log.Printf("%v", err)
		}

		if err = c.auth(); err != nil {
			return resp, err
		}
		req.Header.Set("Authorization", c.Token)
	}

	return resp, err
}
*/
