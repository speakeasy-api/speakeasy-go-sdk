# speakeasy-go-sdk

![180100416-b66263e6-1607-4465-b45d-0e298a67c397](https://user-images.githubusercontent.com/68016351/181640742-31ab234a-3b39-432e-b899-21037596b360.png)

Speakeasy is your API Platform team as a service. Use our drop in SDK to manage all your API Operations including embeds for request logs and usage dashboards, test case generation from traffic, and understanding API drift.

The Speakeasy Go SDK for evaluating API requests/responses. Compatible with any API framework implemented on top of Go's native http library. 

## Requirements

Supported routers: 

* gorilla/mux
* go-chi/chi
* http.DefaultServerMux

We also support custom HTTP frameworks: 

* gin-gonic/gin
* labstack/echo

## Usage

> Speakeasy uses [Go Modules](https://github.com/golang/go/wiki/Modules) to manage dependencies.

```shell
go get github.com/speakeasy-api/speakeasy-go-sdk
```

### Minimum configuration

[Sign up for free on our platform](https://www.speakeasyapi.dev/). After you've created a workspace and generated an API key enable Speakeasy in your API as follows:

Configure Speakeasy at the start of your `main()` function:

```go
import "github.com/speakeasy-api/speakeasy-go-sdk"

func main() {
	// Configure the Global SDK
	speakeasy.Configure(speakeasy.Config {
		APIKey:		"YOUR API KEY HERE",	// retrieve from Speakeasy API dashboard.
		ApiID:		"YOUR API ID HERE", 	// enter a name that you'd like to associate captured requests with.
        // This name will show up in the Speakeasy dashboard. e.g. "PetStore" might be a good ApiID for a Pet Store's API.
        // No spaces allowed.
		VersionID:	"YOUR VERSION ID HERE",	// enter a version that you would like to associate captured requests with.
        // The combination of ApiID (name) and VersionID will uniquely identify your requests in the Speakeasy Dashboard.
        // e.g. "v1.0.0". You can have multiple versions for the same ApiID (if running multiple versions of your API)
	})

    // Associate the SDK's middleware with your router
	r := mux.NewRouter()
	r.Use(speakeasy.Middleware)
}
```

Build and deploy your app and that's it. Your API is being tracked in the Speakeasy workspace you just created
and will be visible on the dashboard next time you log in. Visit our [docs site](https://docs.speakeasyapi.dev/) to
learn more.

### Advanced configuration

The Speakeasy SDK provides both a global and per Api configuration option. If you want to use the SDK to track multiple Apis or Versions from the same service you can configure individual instances of the SDK, like so:

```go
import "github.com/speakeasy-api/speakeasy-go-sdk"

func main() {
	r := mux.NewRouter()

	// Configure a new instance of the SDK for the store API
	storeSDKInstance := speakeasy.New(speakeasy.Config {
		APIKey:		"YOUR API KEY HERE",	// retrieve from Speakeasy API dashboard.
		ApiID:		"store_api", 	   		// this is an ID you provide that you would like to associate captured requests with.
		VersionID:	"1.0.0",				// this is a Version you provide that you would like to associate captured requests with.
	})

	// Configure a new instance of the SDK for the product API
	productSDKInstance := speakeasy.New(speakeasy.Config {
		APIKey:		"YOUR API KEY HERE",	// retrieve from Speakeasy API dashboard.
		ApiID:		"product_api", 			// this is an ID you provide that you would like to associate captured requests with.
		VersionID:	"1.0.0",				// this is a Version you provide that you would like to associate captured requests with.
	})

    // The different instances of the SDK (with differnt IDs or even versions assigned) can be used to associate requests with different APIs and Versions.
	s := r.PathPrefix("/store").Subrouter()
	r.Use(storeSDKInstance.Middleware)

	s := r.PathPrefix("/products").Subrouter()
	r.Use(productSDKInstance.Middleware)
}
```

This allows multiple instances of the SDK to be associated with different routers or routes within your service.

### On-Premise Configuration

The SDK provides a way to redirect the requests it captures to an on-premise deployment of the Speakeasy Platform. This is done through the use of environment variables listed below. These are to be set in the environment of your services that have integrated the SDK:

* `SPEAKEASY_SERVER_URL` - The url of the on-premise Speakeasy Platform's GRPC Endpoint. By default this is `grpc.prod.speakeasyapi.dev:443`.
* `SPEAKEASY_SERVER_SECURE` - Whether or not to use TLS for the on-premise Speakeasy Platform. By default this is `true` set to `SPEAKEASY_SERVER_SECURE="false"` if you are using an insecure connection.

## Request Matching

The Speakeasy SDK out of the box will do its best to match requests to your provided OpenAPI Schema. It does this by extracting the path template used by one of the supported routers or frameworks above for each request captured and attempting to match it to the paths defined in the OpenAPI Schema, for example:

```go
r := mux.NewRouter()
r.Use(sdkInstance.Middleware)
r.HandleFunc("/v1/users/{id}", MyHandler) // The path template "/v1/users/{id}" is captured automatically by the SDK
```

This isn't always successful or even possible, meaning requests received by Speakeasy will be marked as `unmatched`, and potentially not associated with your Api, Version or ApiEndpoints in the Speakeasy Dashboard.

To help the SDK in these situations you can provide path hints per request handler that match the paths in your OpenAPI Schema:

```go
func MyHandler(w http.ResponseWriter, r *http.Request) {
	// Provide a path hint for the request using the OpenAPI Path Templating format: https://swagger.io/specification/#path-templating-matching
	ctrl := speakeasy.MiddlewareController(req)
	ctrl.PathHint("/v1/users/{id}")
	
	// the rest of your handlers code
}
```

Notes:  
Wildcard path matching in Echo & Chi will end up with a OpenAPI path paramater called {wildcard} which will only match single level values represented by the wildcard. This is a restriction of the OpenAPI spec ([Detail Here](https://github.com/OAI/OpenAPI-Specification/issues/892#issuecomment-281449239)). For example: 

`chi template: /user/{id}/path/* => openapi template: /user/{id}/path/{wildcard}`

And in the above example a path like `/user/1/path/some/sub/path` won't match but `/user/1/path/somesubpathstring` will, as `/` characters are not matched in path paramters by the OpenAPI spec.

## Capturing Customer IDs

To help associate requests with customers/users of your APIs you can provide a customer ID per request handler:

```go
func MyHandler(w http.ResponseWriter, r *http.Request) {
	ctrl := speakeasy.MiddlewareController(req)
	ctrl.CustomerID("a-customers-id") // This customer ID will be used to associate this instance of a request with your customers/users
	
	// the rest of your handlers code
}
```

Note: This is not required, but is highly recommended. By setting a customer ID you can easily associate requests with your customers/users in the Speakeasy Dashboard, powering filters in the [Request Viewer](https://docs.speakeasyapi.dev/speakeasy-user-guide/request-viewer).

## Masking sensitive data

Speakeasy can mask sensitive data in the query string parameters, headers, cookies and request/response bodies captured by the SDK. This is useful for maintaining sensitive data isolation, and retaining control over the data that is captured.

Using the `Advanced Configuration` section above you can completely ignore certain routes by not assigning the middleware to their router, causing the SDK to not capture any requests to that router.

But if you would like to be more selective you can mask certain sensitive data using our middleware controller allowing you to mask fields as needed in different handlers:

```go
func MyHandler(w http.ResponseWriter, r *http.Request) {
	ctrl := speakeasy.MiddlewareController(req)
	ctrl.Masking(speakeasy.WithRequestHeaderMask("Authorization")) // Mask the Authorization header in the request
	
	// the rest of your handlers code
}
```

The `Masking` function takes a number of different options to mask sensitive data in the request:

* `speakeasy.WithQueryStringMask` - **WithQueryStringMask** will mask the specified query strings with an optional mask string.
* `speakeasy.WithRequestHeaderMask` - **WithRequestHeaderMask** will mask the specified request headers with an optional mask string.
* `speakeasy.WithResponseHeaderMask` - **WithResponseHeaderMask** will mask the specified response headers with an optional mask string.
* `speakeasy.WithRequestCookieMask` - **WithRequestCookieMask** will mask the specified request cookies with an optional mask string.
* `speakeasy.WithResponseCookieMask` - **WithResponseCookieMask** will mask the specified response cookies with an optional mask string.
* `speakeasy.WithRequestFieldMaskString` - **WithRequestFieldMaskString** will mask the specified request body fields with an optional mask. Supports string fields only. Matches using regex.
* `speakeasy.WithRequestFieldMaskNumber` - **WithRequestFieldMaskNumber** will mask the specified request body fields with an optional mask. Supports number fields only. Matches using regex.
* `speakeasy.WithResponseFieldMaskString` - **WithResponseFieldMaskString** will mask the specified response body fields with an optional mask. Supports string fields only. Matches using regex.
* `speakeasy.WithResponseFieldMaskNumber` - **WithResponseFieldMaskNumber** will mask the specified response body fields with an optional mask. Supports number fields only. Matches using regex.

Masking can also be done more globally on all routes or a selection of routes by taking advantage of middleware. Here is an example:

```go
speakeasy.Configure(speakeasy.Config {
	APIKey:		"YOUR API KEY HERE",	// retrieve from Speakeasy API dashboard.
	ApiID:		"YOUR API ID HERE", 	// this is an ID you provide that you would like to associate captured requests with.
	VersionID:	"YOUR VERSION ID HERE",	// this is a Version you provide that you would like to associate captured requests with.
})

r := mux.NewRouter()
r.Use(speakeasy.Middleware)
r.Use(func (next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mask the Authorization header in the request for all requests served by this middleware
		ctrl := speakeasy.MiddlewareController(req)
		ctrl.Masking(speakeasy.WithRequestHeaderMask("Authorization"))
	})
})
```

## Embedded Request Viewer Access Tokens

The Speakeasy SDK can generate access tokens for the [Embedded Request Viewer](https://docs.speakeasyapi.dev/speakeasy-user-guide/request-viewer/embedded-request-viewer) that can be used to view requests captured by the SDK.

For documentation on how to configure filters, find that [HERE](https://docs.speakeasyapi.dev/speakeasy-user-guide/request-viewer/embedded-request-viewer).

Below are some examples on how to generate access tokens:

```go
import "github.com/speakeasy-api/speakeasy-schemas/grpc/go/registry/embedaccesstoken"

ctx := context.Background()

// If the SDK is configured as a global instance, an access token can be generated using the `GenerateAccessToken` function on the speakeasy package.
accessToken, err := speakeasy.GetEmbedAccessToken(ctx, &embedaccesstoken.EmbedAccessTokenRequest{
	Filters: []*embedaccesstoken.EmbedAccessTokenRequest_Filter{
		{
			Key:   "customer_id",
			Operator: "=",
			Value: "a-customer-id",
		},
	},
})

// If you have followed the `Advanced Configuration` section above you can also generate an access token using the `GenerateAccessToken` function on the sdk instance.
accessToken, err := storeSDKInstance.GetEmbedAccessToken(ctx, &embedaccesstoken.EmbedAccessTokenRequest{
	Filters: []*embedaccesstoken.EmbedAccessTokenRequest_Filter{
		{
			Key:   "customer_id",
			Operator: "=",
			Value: "a-customer-id",
		},
	},
})

// Or finally if you have a handler that you would like to generate an access token from, you can get the SDK instance for that handler from the middleware controller and use the `GetEmbedAccessToken` function it.
func MyHandler(w http.ResponseWriter, r *http.Request) {
	ctrl := speakeasy.MiddlewareController(req)
	accessToken, err := ctrl.GetSDKInstance().GetEmbedAccessToken(ctx, &embedaccesstoken.EmbedAccessTokenRequest{
		Filters: []*embedaccesstoken.EmbedAccessTokenRequest_Filter{
			{
				Key:   "customer_id",
				Operator: "=",
				Value: "a-customer-id",
			},
		},
	})
	
	// the rest of your handlers code
}
```
