package speakeasy

import (
	"context"
	"net/http"
)

type contextKey int

const (
	controllerKey contextKey = iota
)

type controller struct {
	pathHint   string
	customerID string
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

func contextWithController(ctx context.Context) (context.Context, *controller) {
	c := &controller{}
	return context.WithValue(ctx, controllerKey, c), c
}
