package logistics

import (
	"fmt"
	"net/http"
	"time"

	httpclient "github.com/germangorelkin/http-client"
)

type Client struct {
	Instance   string
	httpClient *httpclient.Client

	common service // Reuse a single struct instead of allocating one for each service on the heap.
}

type service struct {
	client *Client
}

type ClintConf struct {
	Instance string
	Verbose  bool
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
		httpClient: cl,
	}

	return c, nil
}
