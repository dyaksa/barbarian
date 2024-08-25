package barbarian

import "net/http"

type Plugins interface {
	OnRequestStart(req *http.Request)
	OnRequestEnd(req *http.Request, res *http.Response)
	OnRequestError(req *http.Request, err error)
}
