package httpclient

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/dyaksa/barbarian"
)

type Config struct {
	BaseUrl string

	HTTPTimeout time.Duration

	Name          string
	MaxRequests   uint32
	Interval      time.Duration
	Timeout       time.Duration
	ReadyToTrip   func(counts Counts) bool
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
	plugins                      []barbarian.Plugins
}

func NewClient(config *Config) (c *Client) {
	c = &Client{
		httpClient:                   createHTTPClient(),
		baseUrl:                      config.BaseUrl,
		considerServerErrorAsFailure: config.ConsiderServerErrorAsFailure,
		serverErrorThreshold:         config.ServerErrorThreshold,
	}

	if config.HTTPTimeout != 0 {
		c.httpClient.Timeout = config.HTTPTimeout
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

var _ barbarian.Client = (*Client)(nil)

func (c *Client) executeRequest(ctx context.Context, method, url string, options ...barbarian.RequestOption) (r *http.Response, err error) {
	resp, err := c.breaker.Execute(func() (interface{}, error) {
		req, err := http.NewRequestWithContext(ctx, method, url, nil)
		if err != nil {
			return nil, err
		}

		c.reportRequest(req)

		for _, option := range options {
			if err := option(req); err != nil {
				return nil, err
			}
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			c.reportError(req, err)
			return resp, err
		}

		if c.considerServerErrorAsFailure && resp.StatusCode >= c.serverErrorThreshold {
			c.reportError(req, fmt.Errorf("response status code: %d", resp.StatusCode))
			return resp, fmt.Errorf("server error: %d", resp.StatusCode)
		}

		c.reportResponse(req, resp)
		return resp, nil
	})

	if err != nil {
		return nil, err
	}

	return resp.(*http.Response), nil
}

func (c *Client) AddPlugin(plugins ...barbarian.Plugins) {
	c.plugins = append(c.plugins, plugins...)
}

func (c *Client) Do(req *http.Request) (res *http.Response, err error) {
	var options []barbarian.RequestOption
	if req.Body != nil {
		options = append(options, BodyJSON(req.Body))
	}
	return c.executeRequest(req.Context(), req.Method, req.URL.String(), options...)
}

func (c *Client) Get(ctx context.Context, path string, options ...barbarian.RequestOption) (res *http.Response, err error) {
	var url bytes.Buffer
	url.WriteString(c.baseUrl)
	url.WriteString(path)
	return c.executeRequest(ctx, http.MethodGet, url.String(), options...)
}

func (c *Client) Post(ctx context.Context, path string, options ...barbarian.RequestOption) (res *http.Response, err error) {
	var url bytes.Buffer
	url.WriteString(c.baseUrl)
	url.WriteString(path)
	return c.executeRequest(ctx, http.MethodPost, url.String(), options...)
}

func (c *Client) Put(ctx context.Context, path string, options ...barbarian.RequestOption) (res *http.Response, err error) {
	var url bytes.Buffer
	url.WriteString(c.baseUrl)
	url.WriteString(path)
	return c.executeRequest(ctx, http.MethodPut, url.String(), options...)
}

func (c *Client) Patch(ctx context.Context, path string, options ...barbarian.RequestOption) (res *http.Response, err error) {
	var url bytes.Buffer
	url.WriteString(c.baseUrl)
	url.WriteString(path)
	return c.executeRequest(ctx, http.MethodPatch, url.String(), options...)
}

func (c *Client) Delete(ctx context.Context, path string, options ...barbarian.RequestOption) (res *http.Response, err error) {
	var url bytes.Buffer
	url.WriteString(c.baseUrl)
	url.WriteString(path)
	return c.executeRequest(ctx, http.MethodDelete, url.String(), options...)
}

func (c *Client) reportRequest(req *http.Request) {
	for _, plugin := range c.plugins {
		plugin.OnRequestStart(req)
	}
}

func (c *Client) reportResponse(req *http.Request, res *http.Response) {
	for _, plugin := range c.plugins {
		plugin.OnRequestEnd(req, res)
	}
}

func (c *Client) reportError(req *http.Request, err error) {
	for _, plugin := range c.plugins {
		plugin.OnRequestError(req, err)
	}
}
