package client

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"

	"github.com/dyaksa/barbarian"
)

type File struct {
	Name      string
	ParamName string
	Reader    io.Reader
}

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

func SetFile(param, filePath string) barbarian.RequestOption {
	return func(r *http.Request) error {
		var formData url.Values
		formData.Set("@"+param, filePath)

		r.Body = io.NopCloser(bytes.NewBufferString(formData.Encode()))
		r.Header.Set("Content-Type", "multipart/form-data")
		return nil
	}
}

func SetFormData(data map[string]string) barbarian.RequestOption {
	return func(r *http.Request) error {
		var formData url.Values
		for key, value := range data {
			formData.Set(key, value)
		}

		r.Body = io.NopCloser(bytes.NewBufferString(formData.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		return nil
	}
}
