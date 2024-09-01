package barbarian

import (
	"net/http"
	"time"
)

type LoggerPlugins interface {
	Plugin
	OnRequestStart(req *http.Request)
	OnRequestEnd(req *http.Request, res *http.Response)
	OnRequestError(req *http.Request, err error)
}

type RetryPlugins interface {
	Plugin
	NextInterval(retry int) time.Duration
}

type Plugin interface {
	Type() string
}
