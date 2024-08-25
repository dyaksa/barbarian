package httpclient

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

type Config struct {
	BaseUrl string

	Name          string
	MaxRequests   uint32
	Interval      time.Duration
	Timeout       time.Duration
	ReadyToTrip   func(cunts Counts) bool
	OnStateChange func(name string, to State, from State)

	ConsiderServerErrorAsFailure bool
	ServerErrorThreshold         int
}

type Client struct {
	httpClient *http.Client
	breaker    *CircuitBreaker

	baseUrl                      string
	considerServerErrorAsFailure bool
	serverErrorThreshold         int
}

func NewClient(config *Config) (c *Client) {
	c = &Client{
		httpClient:                   createHTTPClient(),
		baseUrl:                      config.BaseUrl,
		considerServerErrorAsFailure: config.ConsiderServerErrorAsFailure,
		serverErrorThreshold:         config.ServerErrorThreshold,
	}

	c.breaker = NewCircuitBreaker(Settings{
		Name:          config.Name,
		MaxRequests:   config.MaxRequests,
		Timeout:       config.Timeout,
		Interval:      config.Interval,
		ReadyToTrip:   config.ReadyToTrip,
		OnStateChange: config.OnStateChange,
	})

	return c
}

func createHTTPClient() *http.Client {
	return &http.Client{
		Transport: createHTTPTransport(),
		Timeout:   30 * time.Second,
	}
}

func createHTTPTransport() *http.Transport {
	return &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConnsPerHost:   100,
		MaxConnsPerHost:       100,
	}
}

func (c *Client) executeRequest(ctx context.Context, method, url string, options ...RequestOption) (r *http.Response, err error) {
	resp, err := c.breaker.Execute(func() (interface{}, error) {
		req, err := http.NewRequestWithContext(ctx, method, url, nil)
		if err != nil {
			return nil, err
		}

		for _, option := range options {
			if err := option(req); err != nil {
				return nil, err
			}
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return resp, err
		}

		if c.considerServerErrorAsFailure && resp.StatusCode >= c.serverErrorThreshold {
			return resp, fmt.Errorf("server error: %d", resp.StatusCode)
		}

		return resp, nil
	})

	if err != nil {
		return nil, err
	}

	return resp.(*http.Response), nil
}

func (c *Client) Do(req *http.Request) (res *http.Response, err error) {
	var options []RequestOption
	if req.Body != nil {
		options = append(options, BodyJSON(req.Body))
	}
	return c.executeRequest(req.Context(), req.Method, req.URL.String(), options...)
}

func (c *Client) Get(ctx context.Context, path string, options ...RequestOption) (res *http.Response, err error) {
	var url bytes.Buffer
	url.WriteString(c.baseUrl)
	url.WriteString(path)
	return c.executeRequest(ctx, http.MethodGet, url.String(), options...)
}

func (c *Client) Post(ctx context.Context, path string, options ...RequestOption) (res *http.Response, err error) {
	var url bytes.Buffer
	url.WriteString(c.baseUrl)
	url.WriteString(path)
	return c.executeRequest(ctx, http.MethodPost, url.String(), options...)
}

func (c *Client) Put(ctx context.Context, path string, options ...RequestOption) (res *http.Response, err error) {
	var url bytes.Buffer
	url.WriteString(c.baseUrl)
	url.WriteString(path)
	return c.executeRequest(ctx, http.MethodPut, url.String(), options...)
}

func (c *Client) Patch(ctx context.Context, path string, options ...RequestOption) (res *http.Response, err error) {
	var url bytes.Buffer
	url.WriteString(c.baseUrl)
	url.WriteString(path)
	return c.executeRequest(ctx, http.MethodPatch, url.String(), options...)
}

func (c *Client) Delete(ctx context.Context, path string, options ...RequestOption) (res *http.Response, err error) {
	var url bytes.Buffer
	url.WriteString(c.baseUrl)
	url.WriteString(path)
	return c.executeRequest(ctx, http.MethodDelete, url.String(), options...)
}
