package barbarian

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type ctxKey string

const reqTime ctxKey = "request_time"

type logger struct {
	out    io.Writer
	errOut io.Writer
}

func NewLogger(out io.Writer, errOut io.Writer) LoggerPlugins {
	if out == nil {
		out = os.Stdout
	}

	if errOut == nil {
		errOut = os.Stderr
	}

	return &logger{
		out:    out,
		errOut: errOut,
	}
}

func (l *logger) Type() string {
	return "logger"
}

func (l *logger) OnRequestStart(req *http.Request) {
	ctx := context.WithValue(req.Context(), reqTime, time.Now())
	*req = *(req.WithContext(ctx))
}

func (l *logger) OnRequestEnd(req *http.Request, res *http.Response) {
	reqDuration := getRequestDuration(req.Context()) / time.Millisecond
	method := req.Method
	url := req.URL.String()
	statusCode := res.StatusCode
	fmt.Fprintf(l.out, "%s %s %s %d [%dms]\n", time.Now().Format("02/Jan/2006 03:04:05"), method, url, statusCode, reqDuration)
}

func (l *logger) OnRequestError(req *http.Request, err error) {
	reqDuration := getRequestDuration(req.Context()) / time.Millisecond
	method := req.Method
	url := req.URL.String()
	fmt.Fprintf(l.errOut, "%s %s %s [%dms] ERROR: %v\n", time.Now().Format("02/Jan/2006 03:04:05"), method, url, reqDuration, err)
}

func getRequestDuration(ctx context.Context) time.Duration {
	now := time.Now()
	start := ctx.Value(reqTime)
	if start == nil {
		return 0
	}
	startTime, ok := start.(time.Time)
	if !ok {
		return 0
	}
	return now.Sub(startTime)
}
