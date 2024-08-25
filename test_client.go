package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/dyaksa/barbarian/httpclient"
)

func main() {
	client := httpclient.NewClient(&httpclient.Config{
		Name:                         "test",
		ConsiderServerErrorAsFailure: true,
		ServerErrorThreshold:         500,
		Timeout:                      30 * time.Second,
		OnStateChange: func(name string, to, from httpclient.State) {
			fmt.Printf("State change from %s to %s\n", from, to)
		},
		ReadyToTrip: func(cunts httpclient.Counts) bool {
			return cunts.TotalFailures >= 2
		},
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

	fmt.Println("Server started at :8000")
}

func fetch(client *httpclient.Client) (*http.Response, error) {
	req, err := http.NewRequest("GET", "http://localhost:3001/test", nil)
	if err != nil {
		return nil, err
	}

	return client.Do(req)
}
