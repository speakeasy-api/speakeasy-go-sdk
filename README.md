# speakeasy-go-sdk

![180100416-b66263e6-1607-4465-b45d-0e298a67c397](https://user-images.githubusercontent.com/68016351/181640742-31ab234a-3b39-432e-b899-21037596b360.png)

Speakeasy is your API Platform team as a service. Use our drop in SDK to manage all your API Operations including embeds for request logs and usage dashboards, test case generation from traffic, and understanding API drift.

The Speakeasy Go SDK for evaluating API requests/responses. Compatible with any API framework implemented on top of Go's native http library. 

## Requirements

Supported frameworks: 

* gorilla/mux
* go-chi/chi
* http.DefaultServerMux

We also support custom Http frameworks: 

* gin-gonic/gin
* labstack/echo

## Usage

> Speakeasy uses [Go Modules](https://github.com/golang/go/wiki/Modules) to manage dependencies.

```shell
go get github.com/speakeasy-api/speakeasy-go-sdk
```

## Minimum configuration

[Sign up for free on our platform](https://www.speakeasyapi.dev/). After you've created a workspace and generated an API key enable Speakeasy in your API as follows:

Configure Speakeasy at the start of your `main()` function with just 2 lines of code:

```go
import "github.com/speakeasy-api/speakeasy-go-sdk"

func main() {
	speakeasy.Configure(speakeasy.Configuration {
		APIKey:     "YOUR API KEY HERE",     // retrieve from Speakeasy API dashboard
	})
	// rest of your program.
}
```

Build and deploy your app and that's it. Your API is being tracked in the Speakeasy workspace you just created
and will be visible on the dashboard next time you log in. Visit our [docs site](https://docs.speakeasyapi.dev/) to
learn more.


## Optional Arguments

Coming soon !
```
