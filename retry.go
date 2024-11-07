package barbarian

import (
	"time"
)

type Retriable interface {
	NextInterval(retry int) time.Duration
}

type RetriableFunc func(retry int) time.Duration

func (f RetriableFunc) NextInterval(retry int) time.Duration {
	return f(retry)
}

type retrier struct {
	backoff Backoff
}

func NewRetrier(backoff Backoff) Plugin {
	return &retrier{
		backoff: backoff,
	}
}

func NewRetrierFunc(f RetriableFunc) Retriable {
	return f
}

func (r *retrier) Type() string {
	return "retrier"
}

func (r *retrier) NextInterval(retry int) time.Duration {
	return r.backoff.Next(retry)
}

type noRetrier struct {
}

func NewNoRetrier() Retriable {
	return &noRetrier{}
}

func (r *noRetrier) NextInterval(retry int) time.Duration {
	return 0 * time.Millisecond
}
