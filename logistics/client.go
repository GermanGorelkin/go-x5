package logistics

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	httpclient "github.com/germangorelkin/http-client"
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
	if cfg.Verbose {
		if err := cl.AddInterceptor(httpclient.DumpInterceptor); err != nil {
			return nil, err
		}
	}

	c := &Client{
		Instance:   cfg.Instance,
		Login:      cfg.Login,
		Password:   cfg.Password,
		httpClient: cl,
	}

	if cfg.AutoAuth {
		c.httpClient.AddInterceptor(c.AuthInterceptor)
	}

	c.common.client = c
	c.Auth = (*AuthService)(&c.common)
	c.Reports = (*ReportService)(&c.common)

	return c, nil
}

func (c *Client) SetToken(token string) {
	c.Token = fmt.Sprintf("%s %s", "Bearer", token)
	c.httpClient.SetAuthorization(c.Token)
}

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
	// если нет токена и запрос НЕ на его получения, тогда сначала запршиваем токен
	if req.Header.Get("Authorization") == "" && req.URL.Path != URL_AUTH {
		if err = c.auth(); err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", c.Token)
	}

	attempt := 2 // TODO
	for i := 0; i < attempt; i++ {
		resp, err = handler(req)

		// если ошибка НЕ в авторизации, тогда выходим
		if ok := c.isUnauthorized(resp); !ok {
			break
		}

		// если ошибка в авторизации при запросе токена, тогда выходим
		if req.URL.Path == URL_AUTH {
			break
		}

		// если ошибка в авторизации(протух токен) тогда запрашиваем токен и повторяем попытку
		if err = c.auth(); err != nil {
			return resp, err
		}
		req.Header.Set("Authorization", c.Token)
	}

	return resp, err
}
