package httpclient

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/dyaksa/barbarian"
)

func WithHeaders(headers map[string]string) barbarian.RequestOption {
	return func(req *http.Request) error {
		for key, value := range headers {
			req.Header.Set(key, value)
		}
		return nil
	}
}

func WithBasicAuth(username, password string) barbarian.RequestOption {
	return func(req *http.Request) error {
		req.SetBasicAuth(username, password)
		return nil
	}
}

func WithBearerToken(token string) barbarian.RequestOption {
	return func(req *http.Request) error {
		req.Header.Set("Authorization", "Bearer "+token)
		return nil
	}
}

func BodyJSON(body interface{}) barbarian.RequestOption {
	return func(r *http.Request) error {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return err
		}

		r.Body = io.NopCloser(bytes.NewBuffer(jsonBody))
		return nil
	}
}
