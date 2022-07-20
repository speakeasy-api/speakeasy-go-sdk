# speakeasy-go-sdk
The Speakeasy Go SDK for evaluating API requests/responses. Compatible with any API framework implemented on top of Go's native http library.

## Installation
> Speakeasy uses [Go Modules](https://github.com/golang/go/wiki/Modules) to manage dependencies.

```shell
go get github.com/speakeasy-api/speakeasy-go-sdk
```

## Minimum configuration
Configure Speakeasy at the start of your `main()` function:

```go
import "github.com/speakeasy-api/speakeasy-go-sdk"

func main() {
	speakeasy.Configure(speakeasy.Configuration {
		APIKey:     "YOUR API KEY HERE",     // retrieve from future Speakeasy dev dashboard
	})
	// rest of your program.
	mux := http.NewServeMux()
	mux.Handle("/", speakeasy.Middleware(yourHandler))	
}
```
## Optional Arguments
The only required argument to the Speakeasy configuration is your API key. There are some optional parameters which you can choose to include. The full list is found below:
```go
speakeasy.Configure(speakeasy.Configuration {
		APIKey:   "YOUR API KEY HERE",  // Initialize with value from Speakeasy dev dashboard
		PathHints:  "REGEX EXPRESSION",  //Regex expression for discovering new API endpoints
	})
```
