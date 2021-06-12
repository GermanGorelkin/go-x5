package logistics

import (
	"fmt"
	"net/http"
	"time"

	httpclient "github.com/germangorelkin/http-client"
)

const (
	URL_REPORT_STATUS = "/v1/logistics/report/%s/status"
	URL_REPORT_CREATE = "/v1/logistics/report"
	URL_AUTH          = "/v1/logistics/auth"
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
}

func NewClient(cfg ClintConf) (*Client, error) {
	cl, err := httpclient.New(
		// TODO timeout
		&http.Client{Timeout: 30 * time.Second},
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

	c.common.client = c
	c.Auth = (*AuthService)(&c.common)
	c.Reports = (*ReportService)(&c.common)

	return c, nil
}
