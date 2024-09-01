package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/dyaksa/barbarian/httpclient"
	"github.com/dyaksa/barbarian/plugins"
)

func main() {
	client := httpclient.NewClient(&httpclient.Config{
		Name:                         "test",
		BaseUrl:                      "https://webhook.site",
		ConsiderServerErrorAsFailure: true,
		ServerErrorThreshold:         500,
		ReadyToTrip: func(cunts httpclient.Counts) bool {
			return cunts.TotalFailures > 2
		},
		Timeout: 30 * time.Second,
	})

	logger := plugins.NewLogger(nil, nil)
	client.AddPlugin(logger)

	// retrier := plugins.NewRetrier(plugins.NewConstantBackoff(1*time.Second, 1))
	// client.AddPlugin(retrier)

	client.FallbackFunc(func() (*http.Response, error) {
		httpClient := &http.Client{}
		payload, err := json.Marshal(map[string]string{"nama": "John Doe"})
		if err != nil {
			return nil, err
		}
		req, err := http.NewRequest(http.MethodGet, "https://webhook.site/339df973-287a-4cf1-b1b0-c79913986ad8", bytes.NewBuffer(payload))
		if err != nil {
			return nil, err
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("error status code %d ", resp.StatusCode)
		}

		return resp, nil
	})

	http.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		resp, err := fetch(client)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			respErr := map[string]interface{}{
				"error": err.Error(),
			}
			b, _ := json.Marshal(respErr)
			w.Write(b)
			return
		}

		defer resp.Body.Close()

		b, err := io.ReadAll(resp.Body)
		if err != nil {
			w.Write([]byte(err.Error()))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(b)
	})

	if err := http.ListenAndServe(":9000", nil); err != nil {
		fmt.Println("Error:", err)
	}
}

func fetch(client *httpclient.Client) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, "http://localhost:3001/test", nil)
	if err != nil {
		return nil, err
	}

	return client.Do(req)
}
