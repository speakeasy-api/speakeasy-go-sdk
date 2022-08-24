package speakeasy

import (
	"context"
	"net/http"
)

const (
	DefaultStringMask = "__masked__"
	DefaultNumberMask = "-12321"
)

type MaskingOption func(c *controller)

// WithQueryStringMask will mask the specified query strings with an optional mask string.
// If no mask is provided, the value will be masked with the default mask.
// If a single mask is provided, it will be used for all query strings.
// If the number of masks provided is equal to the number of query strings, masks will be used in order.
// Otherwise, the masks will be used in order until it they are exhausted. If the masks are exhausted, the default mask will be used.
// (defaults to "__masked__").
func WithQueryStringMask(keys []string, masks ...string) MaskingOption {
	return func(c *controller) {
		for i, key := range keys {
			switch {
			case len(masks) == 1:
				c.queryStringMasks[key] = masks[0]
			case len(masks) > i:
				c.queryStringMasks[key] = masks[i]
			default:
				c.queryStringMasks[key] = DefaultStringMask
			}
		}
	}
}

// WithRequestHeaderMask will mask the specified request headers with an optional mask string.
// If no mask is provided, the value will be masked with the default mask.
// If a single mask is provided, it will be used for all headers.
// If the number of masks provided is equal to the number of headers, masks will be used in order.
// Otherwise, the masks will be used in order until it they are exhausted. If the masks are exhausted, the default mask will be used.
// (defaults to "__masked__").
func WithRequestHeaderMask(headers []string, masks ...string) MaskingOption {
	return func(c *controller) {
		for i, header := range headers {
			switch {
			case len(masks) == 1:
				c.requestHeaderMasks[header] = masks[0]
			case len(masks) > i:
				c.requestHeaderMasks[header] = masks[i]
			default:
				c.requestHeaderMasks[header] = DefaultStringMask
			}
		}
	}
}

// WithResponseHeaderMask will mask the specified response headers with an optional mask string.
// If no mask is provided, the value will be masked with the default mask.
// If a single mask is provided, it will be used for all headers.
// If the number of masks provided is equal to the number of headers, masks will be used in order.
// Otherwise, the masks will be used in order until it they are exhausted. If the masks are exhausted, the default mask will be used.
// (defaults to "__masked__").
func WithResponseHeaderMask(headers []string, masks ...string) MaskingOption {
	return func(c *controller) {
		for i, header := range headers {
			switch {
			case len(masks) == 1:
				c.responseHeaderMasks[header] = masks[0]
			case len(masks) > i:
				c.responseHeaderMasks[header] = masks[i]
			default:
				c.responseHeaderMasks[header] = DefaultStringMask
			}
		}
	}
}

// WithRequestCookieMask will mask the specified request cookies with an optional mask string.
// If no mask is provided, the value will be masked with the default mask.
// If a single mask is provided, it will be used for all cookies.
// If the number of masks provided is equal to the number of cookies, masks will be used in order.
// Otherwise, the masks will be used in order until it they are exhausted. If the masks are exhausted, the default mask will be used.
// (defaults to "__masked__").
func WithRequestCookieMask(cookies []string, masks ...string) MaskingOption {
	return func(c *controller) {
		for i, cookie := range cookies {
			switch {
			case len(masks) == 1:
				c.requestCookieMasks[cookie] = masks[0]
			case len(masks) > i:
				c.requestCookieMasks[cookie] = masks[i]
			default:
				c.requestCookieMasks[cookie] = DefaultStringMask
			}
		}
	}
}

// WithResponseCookieMask will mask the specified response cookies with an optional mask string.
// If no mask is provided, the value will be masked with the default mask.
// If a single mask is provided, it will be used for all cookies.
// If the number of masks provided is equal to the number of cookies, masks will be used in order.
// Otherwise, the masks will be used in order until it they are exhausted. If the masks are exhausted, the default mask will be used.
// (defaults to "__masked__").
func WithResponseCookieMask(cookies []string, masks ...string) MaskingOption {
	return func(c *controller) {
		for i, cookie := range cookies {
			switch {
			case len(masks) == 1:
				c.responseCookieMasks[cookie] = masks[0]
			case len(masks) > i:
				c.responseCookieMasks[cookie] = masks[i]
			default:
				c.responseCookieMasks[cookie] = DefaultStringMask
			}
		}
	}
}

// WithRequestFieldMaskString will mask the specified request body fields with an optional mask. Supports string fields only. Matches using regex.
// If no mask is provided, the value will be masked with the default mask.
// If a single mask is provided, it will be used for all fields.
// If the number of masks provided is equal to the number of fields, masks will be used in order.
// Otherwise, the masks will be used in order until it they are exhausted. If the masks are exhausted, the default mask will be used.
// (defaults to "__masked__").
func WithRequestFieldMaskString(fields []string, masks ...string) MaskingOption {
	return func(c *controller) {
		for i, field := range fields {
			switch {
			case len(masks) == 1:
				c.requestFieldMasksString[field] = masks[0]
			case len(masks) > i:
				c.requestFieldMasksString[field] = masks[i]
			default:
				c.requestFieldMasksString[field] = DefaultStringMask
			}
		}
	}
}

// WithRequestFieldMaskNumber will mask the specified request body fields with an optional mask. Supports number fields only. Matches using regex.
// If no mask is provided, the value will be masked with the default mask.
// If a single mask is provided, it will be used for all fields.
// If the number of masks provided is equal to the number of fields, masks will be used in order.
// Otherwise, the masks will be used in order until it they are exhausted. If the masks are exhausted, the default mask will be used.
// (defaults to "-12321").
func WithRequestFieldMaskNumber(fields []string, masks ...string) MaskingOption {
	return func(c *controller) {
		for i, field := range fields {
			switch {
			case len(masks) == 1:
				c.requestFieldMasksNumber[field] = masks[0]
			case len(masks) > i:
				c.requestFieldMasksNumber[field] = masks[i]
			default:
				c.requestFieldMasksNumber[field] = DefaultNumberMask
			}
		}
	}
}

// WithResponseFieldMaskString will mask the specified response body with an optional mask. Supports string only. Matches using regex.
// If no mask is provided, the value will be masked with the default mask.
// If a single mask is provided, it will be used for all fields.
// If the number of masks provided is equal to the number of fields, masks will be used in order.
// Otherwise, the masks will be used in order until it they are exhausted. If the masks are exhausted, the default mask will be used.
// (defaults to "__masked__").
func WithResponseFieldMaskString(fields []string, masks ...string) MaskingOption {
	return func(c *controller) {
		for i, field := range fields {
			switch {
			case len(masks) == 1:
				c.responseFieldMasksString[field] = masks[0]
			case len(masks) > i:
				c.responseFieldMasksString[field] = masks[i]
			default:
				c.responseFieldMasksString[field] = DefaultStringMask
			}
		}
	}
}

// WithResponseFieldMaskNumber will mask the specified response body with an optional mask. Supports number fields only. Matches using regex.
// If no mask is provided, the value will be masked with the default mask.
// If a single mask is provided, it will be used for all fields.
// If the number of masks provided is equal to the number of fields, masks will be used in order.
// Otherwise, the masks will be used in order until it they are exhausted. If the masks are exhausted, the default mask will be used.
// (defaults to "-12321").
func WithResponseFieldMaskNumber(fields []string, masks ...string) MaskingOption {
	return func(c *controller) {
		for i, field := range fields {
			switch {
			case len(masks) == 1:
				c.responseFieldMasksNumber[field] = masks[0]
			case len(masks) > i:
				c.responseFieldMasksNumber[field] = masks[i]
			default:
				c.responseFieldMasksNumber[field] = DefaultNumberMask
			}
		}
	}
}

type contextKey int

const (
	controllerKey contextKey = iota
)

type controller struct {
	pathHint                 string
	customerID               string
	queryStringMasks         map[string]string
	requestHeaderMasks       map[string]string
	requestCookieMasks       map[string]string
	requestFieldMasksString  map[string]string
	requestFieldMasksNumber  map[string]string
	responseHeaderMasks      map[string]string
	responseCookieMasks      map[string]string
	responseFieldMasksString map[string]string
	responseFieldMasksNumber map[string]string
	sdkInstance              *Speakeasy
}

// MiddlewareController will return the speakeasy middleware controller from the current request,
// if the current request is monitored by the speakeasy middleware.
func MiddlewareController(r *http.Request) *controller {
	c, _ := r.Context().Value(controllerKey).(*controller)
	return c
}

// PathHint will allow you to provide a path hint for the current request.
func (c *controller) PathHint(pathHint string) {
	c.pathHint = pathHint
}

// CustomerID will allow you to associate a customer ID with the current request.
func (c *controller) CustomerID(customerID string) {
	c.customerID = customerID
}

func (c *controller) Masking(opts ...MaskingOption) {
	for _, opt := range opts {
		opt(c)
	}
}

func (c *controller) GetSDKInstance() *Speakeasy {
	return c.sdkInstance
}

func contextWithController(ctx context.Context, sdk *Speakeasy) (context.Context, *controller) {
	c := &controller{
		queryStringMasks:         make(map[string]string),
		requestHeaderMasks:       make(map[string]string),
		requestCookieMasks:       make(map[string]string),
		requestFieldMasksString:  make(map[string]string),
		requestFieldMasksNumber:  make(map[string]string),
		responseHeaderMasks:      make(map[string]string),
		responseCookieMasks:      make(map[string]string),
		responseFieldMasksString: make(map[string]string),
		responseFieldMasksNumber: make(map[string]string),
		sdkInstance:              sdk,
	}
	return context.WithValue(ctx, controllerKey, c), c
}
