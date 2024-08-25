package httpclient

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

type RequestOption func(*http.Request) error

func WithAuthorization(key, value string) RequestOption {
	return func(req *http.Request) error {
		req.Header.Set(key, value)
		return nil
	}
}

func BodyJSON(body interface{}) RequestOption {
	return func(r *http.Request) error {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return err
		}

		r.Body = io.NopCloser(bytes.NewBuffer(jsonBody))
		return nil
	}
}
