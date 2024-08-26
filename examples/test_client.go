package main

import (
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
		BaseUrl:                      "http://localhost:3001",
		ConsiderServerErrorAsFailure: true,
		ServerErrorThreshold:         500,
		ReadyToTrip: func(cunts httpclient.Counts) bool {
			return cunts.TotalFailures > 2
		},
		Timeout: 30 * time.Second,
	})

	logger := plugins.NewLogger(nil, nil)

	client.AddPlugin(logger)

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

	fmt.Println("Server started at :8000")
}

func fetch(client *httpclient.Client) (*http.Response, error) {
	req, err := http.NewRequest("GET", "http://localhost:3001/test", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "apiKey 123456")

	return client.Do(req)
}
