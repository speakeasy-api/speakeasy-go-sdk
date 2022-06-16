# speakeasy-go-sdk
Go SDK for parsing API requests and responses for any API framework using native http library

## Installation

```shell
go get github.com/speakeasy-api/speakeasy-go-sdk
```

Speakeasy uses [Go Modules](https://github.com/golang/go/wiki/Modules) to manage dependencies.


## Basic configuration

Configure Speakeasy at the start of your `main()` function:

```go
import "github.com/speakeasy-api/speakeasy-go-sdk"

func main() {
	speakeasy.Configure(speakeasy.Configuration{
		APIKey:     "YOUR API KEY HERE",     // retrieve from future Speakeasy dev dashboard
		ProjectID:  "YOUR WORKSPACE ID HERE" // workspace id
		KeysToMask: []string{"password"},    // optional, mask fields you don't want sent to Speakeasy
		ServerURL:  "localhost://3000",      // optional, don't use default server URL
	}

    // rest of your program.
}

```


After that, just use the middleware with any of your handlers:
 ```go
mux := http.NewServeMux()
mux.Handle("/", speakeasy.Middleware(yourHandler))
```
