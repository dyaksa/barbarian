package httpclient

import (
	"errors"
	"sync"
)

type MultiError struct {
	mutex sync.Mutex
	errs  []error
}

func (m *MultiError) Push(errString string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.errs = append(m.errs, errors.New(errString))
}

func (m *MultiError) HasError() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if len(m.errs) == 0 {
		return nil
	}

	return m
}

func (m *MultiError) Error() string {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.errs[len(m.errs)-1].Error()
}
