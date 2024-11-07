package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/dyaksa/barbarian"
	"io"
	"net/http"
	"time"

	"github.com/dyaksa/barbarian/client"
)

func main() {
	client := client.NewClient(&client.Config{
		Name:                         "test",
		ConsiderServerErrorAsFailure: true,
		ServerErrorThreshold:         500,
		ReadyToTrip: func(cunts client.Counts) bool {
			return cunts.TotalFailures > 2
		},
		Timeout: 30 * time.Second,
	})

	logger := barbarian.NewLogger(nil, nil)
	client.AddPlugin(logger)

	//retrier := plugins.NewRetrier(plugins.NewConstantBackoff(3*time.Second, 1))
	//client.AddPlugin(retrier)

	// client.FallbackFunc(func() (*http.Response, error) {
	// 	httpClient := &http.Client{}
	// 	payload, err := json.Marshal(map[string]string{"nama": "John Doe"})
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	req, err := http.NewRequest(http.MethodGet, "https://webhook.site/339df973-287a-4cf1-b1b0-c79913986ad8", bytes.NewBuffer(payload))
	// 	if err != nil {
	// 		return nil, err
	// 	}

	// 	resp, err := httpClient.Do(req)
	// 	if err != nil {
	// 		return nil, err
	// 	}

	// 	if resp.StatusCode != http.StatusOK {
	// 		return nil, fmt.Errorf("error status code %d ", resp.StatusCode)
	// 	}

	// 	return resp, nil
	// })

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

type Body struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func fetch(client *client.Client) (*http.Response, error) {

	var body = Body{
		Name:  "John Doe",
		Email: "johndoe@gmail.com",
	}

	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, "https://webhook.site/e4381a42-1983-4884-bed4-1ca307685357", bytes.NewBuffer(b))
	if err != nil {
		return nil, err
	}

	return client.Do(req)
}
