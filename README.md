# Barbarian - Simple HTTP Client For Go

Barbarian is an simple HTTP client that helps your application with circuit breaker

All HTTP methods are exposed as a fluent interface.

## Installation

```
go get -u github.com/dyaksa/barbarian
```

## Usage

### Importing the package

This package can be used by adding the following import statement to your `.go` files.

```go
import "github.com/dyaksa/barbarian/httpclient"
```

### Making a simple `GET` request

The below example will print the contents:

```go
// Create a new HTTP client with a default timeout
client := httpclient.NewClient(&httpclient.Config{
	BaseUrl:     "http://localhost:3001", // baseurl
	HTTPTimeout: 30 * time.Second, // http timeout
})

// Use the clients GET method to create and execute the request
res, err := client.Get(context.Background(), "/test")
if err != nil {
    panic(err)
}
// Barbarian returns the standard *http.Response object
body, err := ioutil.ReadAll(res.Body)
fmt.Println(string(body))
```

You can also use the `*http.Request` object with the `http.Do` interface :

```go
client := httpclient.NewClient(&httpclient.Config{
	HTTPTimeout: 30 * time.Second, // http timeout
})

// Create an http.Request instance
req, _ := http.NewRequest(http.MethodGet, "http://google.com", nil)
// Call the `Do` method, which has a similar interface to the `http.Do` method
res, err := client.Do(req)
if err != nil {
	panic(err)
}

body, err := ioutil.ReadAll(res.Body)
fmt.Println(string(body))
```

You can configure `CircuitBreaker` by the struct `Settings`:

```go
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

req, _ := http.NewRequest(http.MethodGet, "http://google.com", nil)

if err != nil {
    panic(err)
}

// Call the `Do` method, which has a similar interface to the `http.Do` method
res, err := client.Do(req)
if err != nil {
	panic(err)
}

body, err := ioutil.ReadAll(res.Body)
fmt.Println(string(body))
```

- `Name` is the name of the `CircuitBreaker`.

- `ConsiderServerErrorAsFailure` Determines whether server errors should trigger the circuit breaker.

- `ServerErrorThreshold` Specifies the HTTP status code threshold at which the circuit breaker should open.

- `MaxRequests` is the maximum number of requests allowed to pass through
  when the `CircuitBreaker` is half-open.
  If `MaxRequests` is 0, `CircuitBreaker` allows only 1 request.

- `Interval` is the cyclic period of the closed state
  for `CircuitBreaker` to clear the internal `Counts`, described later in this section.
  If `Interval` is 0, `CircuitBreaker` doesn't clear the internal `Counts` during the closed state.

- `Timeout` is the period of the open state,
  after which the state of `CircuitBreaker` becomes half-open.
  If `Timeout` is 0, the timeout value of `CircuitBreaker` is set to 60 seconds.

- `ReadyToTrip` is called with a copy of `Counts` whenever a request fails in the closed state.
  If `ReadyToTrip` returns true, `CircuitBreaker` will be placed into the open state.
  If `ReadyToTrip` is `nil`, default `ReadyToTrip` is used.
  Default `ReadyToTrip` returns true when the number of consecutive failures is more than 5.

- `OnStateChange` is called whenever the state of `CircuitBreaker` changes.

- `IsSuccessful` is called with the error returned from a request.
  If `IsSuccessful` returns true, the error is counted as a success.
  Otherwise the error is counted as a failure.
  If `IsSuccessful` is nil, default `IsSuccessful` is used, which returns false for all non-nil errors.

You can call options `GET` by the method `Options`:

```go
client := httpclient.NewClient(&httpclient.Config{
	Name:    "http client",
	BaseUrl: "http://localhost:3001",
})

headers := map[string]string{
	"Content-Type":  "application/json",
	"Authorization": "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
}

res, err :=  client.Get(context.Background(), "/test",
	httpclient.WithHeaders(headers),
	httpclient.BodyJSON(map[string]interface{}{"name": "John Doe"}),
)

if err != nil {
	panic(err)
}

body, err := ioutil.ReadAll(res.Body)
fmt.Println(string(body))
```

## Plugins

To add a plugin to an existing client, use the `AddPlugin` method of the client.

An example, with the [logger plugin](/plugins/logger.go):

```go
// import "github.com/dyaksa/barbarian/plugins"
client := httpclient.NewClient(&httpclient.Config{
	Name:       "http client",
    BaseUrl:    "http://localhost:3001",
    HTTPTimout: 30 * time.Second
})

logger := plugins.NewLogger(nil, nil)
client.AddPlugin(logger)

res, err := client.Get(context.Background(), "/test", nil)
if err != nil {
    panic(err)
}

// This will log:
//23/Aug/2024 12:48:04 GET http://localhost:3001 200 [412ms]
// to STDOUT
```

A plugin is an interface whose methods get called during key events in a requests lifecycle:

- `OnRequestStart` is called just before the request is made
- `OnRequestEnd` is called once the request has successfully executed
- `OnError` is called is the request failed

Each method is called with the request object as an argument, with `OnRequestEnd`, and `OnError` additionally being called with the response and error instances respectively.
For a simple example on how to write plugins, look at the [logger plugin](/plugins/logger.go).

## License

Copyright 2024

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
