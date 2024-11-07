package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/dyaksa/barbarian"
	"github.com/pkg/errors"
)

type Options func(*Client)

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

	RetryCount int
}

type Client struct {
	httpClient *http.Client
	breaker    *CircuitBreaker

	baseUrl                      string
	considerServerErrorAsFailure bool
	serverErrorThreshold         int
	plugins                      map[string][]barbarian.Plugin

	fallback func() (*http.Response, error)

	retrier    barbarian.Retriable
	retryCount int
}

func NewClient(config *Config) (c *Client) {
	c = &Client{
		httpClient:                   createHTTPClient(),
		plugins:                      make(map[string][]barbarian.Plugin),
		retrier:                      barbarian.NewNoRetrier(),
		retryCount:                   config.RetryCount - 1,
		baseUrl:                      config.BaseUrl,
		considerServerErrorAsFailure: config.ConsiderServerErrorAsFailure,
		serverErrorThreshold:         config.ServerErrorThreshold,
	}

	if config.HTTPTimeout != 0 {
		c.httpClient.Timeout = config.HTTPTimeout
	}

	if c.fallback == nil {
		c.fallback = defaultFallback
	}

	if c.retryCount <= 0 {
		c.retryCount = 0
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

func defaultFallback() (resp *http.Response, err error) {
	return resp, nil
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
			return nil, errors.Wrap(err, "failed to create request")
		}

		c.reportRequest(req)

		for _, option := range options {
			if err := option(req); err != nil {
				return nil, errors.Wrap(err, "failed to apply request option")
			}
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			c.reportError(req, err)
			return resp, errors.Wrap(err, "failed to execute request")
		}

		if c.considerServerErrorAsFailure && resp.StatusCode >= c.serverErrorThreshold {
			c.reportError(req, fmt.Errorf("response status code: %d", resp.StatusCode))
			return resp, fmt.Errorf("server error: %d", resp.StatusCode)
		}

		c.reportResponse(req, resp)
		return resp, nil
	})

	if err != nil {
		resp, errFallback := c.fallback()
		if errFallback != nil {
			return nil, errors.Wrap(err, "failed to execute request")
		}

		if resp == nil {
			return nil, errors.Wrap(err, "failed to execute request")
		}
		return resp, nil
	}

	return resp.(*http.Response), nil
}

func (c *Client) AddPlugin(plugin barbarian.Plugin) {
	pluginType := plugin.Type()
	c.plugins[pluginType] = append(c.plugins[pluginType], plugin)
}

func (c *Client) FallbackFunc(f func() (*http.Response, error)) {
	c.fallback = f
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	c.setRetrier()

	resp, err := c.breaker.Execute(func() (interface{}, error) {
		return c.executeWithRetry(req)
	})

	if err != nil {
		return c.handleError(err)
	}

	return resp.(*http.Response), nil
}

func (c *Client) executeWithRetry(req *http.Request) (*http.Response, error) {
	bodyReader, err := c.prepareRequestBody(req)
	if err != nil {
		return nil, err
	}

	if c.breaker.IsCircuitBreakerOpen() {
		return nil, errors.Wrap(errors.New("circuit breaker is open"), "circuit breaker")
	}

	var lastError error
	for attempt := 0; attempt <= c.retryCount; attempt++ {
		resp, err := c.performRequest(req, bodyReader)
		if err == nil && !c.isServerError(resp) {
			return resp, nil
		}

		lastError = c.handleRequestError(err)
		if attempt < c.retryCount {
			c.waitBeforeRetry(attempt)
		}
	}

	return nil, lastError
}

func (c *Client) prepareRequestBody(req *http.Request) (*bytes.Reader, error) {
	if req.Body == nil {
		return nil, nil
	}

	reqData, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read request body")
	}

	bodyReader := bytes.NewReader(reqData)
	req.Body = io.NopCloser(bodyReader)
	return bodyReader, nil
}

func (c *Client) performRequest(req *http.Request, bodyReader *bytes.Reader) (*http.Response, error) {
	c.reportRequest(req)

	if bodyReader != nil {
		_, _ = bodyReader.Seek(0, 0)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.reportError(req, err)
		return nil, errors.Wrap(err, "failed to execute request")
	}

	c.reportResponse(req, resp)
	return resp, nil
}

func (c *Client) isServerError(resp *http.Response) bool {
	return c.considerServerErrorAsFailure && resp.StatusCode >= c.serverErrorThreshold
}

func (c *Client) handleRequestError(err error) error {
	if err != nil {
		return errors.Wrap(err, "request failed")
	}
	return errors.New("server error")
}

func (c *Client) waitBeforeRetry(attempt int) {
	backoffTime := c.retrier.NextInterval(attempt)
	time.Sleep(backoffTime)
}

func (c *Client) handleError(err error) (*http.Response, error) {
	resp, errFallback := c.fallback()
	if errFallback != nil {
		return nil, errFallback
	}

	if resp == nil {
		return nil, err
	}

	return resp, nil
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
	for _, plugin := range c.plugins["logger"] {
		if logger, ok := plugin.(barbarian.LoggerPlugins); ok {
			logger.OnRequestStart(req)
		}
	}
}

func (c *Client) reportResponse(req *http.Request, res *http.Response) {
	for _, plugin := range c.plugins["logger"] {
		if logger, ok := plugin.(barbarian.LoggerPlugins); ok {
			logger.OnRequestEnd(req, res)
		}
	}
}

func (c *Client) reportError(req *http.Request, err error) {
	for _, plugin := range c.plugins["logger"] {
		if logger, ok := plugin.(barbarian.LoggerPlugins); ok {
			logger.OnRequestError(req, err)
		}
	}
}

func (c *Client) setRetrier() {
	for _, plugin := range c.plugins["retrier"] {
		if retrier, ok := plugin.(barbarian.RetryPlugins); ok {
			c.retrier = retrier
		}
	}
}
