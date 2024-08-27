package barbarian

import (
	"context"
	"net/http"
)

type RequestOption func(*http.Request) error

type Client interface {
	Do(req *http.Request) (res *http.Response, err error)
	Get(ctx context.Context, path string, options ...RequestOption) (res *http.Response, err error)
	Post(ctx context.Context, path string, options ...RequestOption) (res *http.Response, err error)
	Put(ctx context.Context, path string, options ...RequestOption) (res *http.Response, err error)
	Patch(ctx context.Context, path string, options ...RequestOption) (res *http.Response, err error)
	Delete(ctx context.Context, path string, options ...RequestOption) (res *http.Response, err error)
	AddPlugin(plugins ...Plugins)
	Fallback(f func() (*http.Response, error))
}
