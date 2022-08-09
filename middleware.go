package speakeasy

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/mux"
	"github.com/labstack/echo/v4"
)

// Middleware setups up the default SDK instance to start capturing requests from routers that support http.Handlers.
// Currently only gorilla/mux, go-chi/chi routers and the http.DefaultServerMux are supported for automatically
// capturing path hints. Otherwise path hints can be supplied by a handler through the speakeasy MiddlewareController.
func Middleware(next http.Handler) http.Handler {
	return defaultInstance.Middleware(next)
}

// Middleware setups the current instance of the SDK to start capturing requests from routers that support http.Handlers.
// Currently only gorilla/mux, go-chi/chi routers and the http.DefaultServerMux are supported for automatically
// capturing path hints. Otherwise path hints can be supplied by a handler through the speakeasy MiddlewareController.
func (s *Speakeasy) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.handleRequestResponse(w, r, next.ServeHTTP, func(r *http.Request) string {
			var pathHint string

			// First check gorilla/mux for a path hint
			route := mux.CurrentRoute(r)
			if route != nil {
				pathHint, _ = route.GetPathTemplate()
				if pathHint != "" {
					return pathHint
				}
			}

			// Check chi router for a path hint
			routeContext := chi.RouteContext(r.Context())
			if routeContext != nil {
				pathHint = routeContext.RoutePattern()
				if pathHint != "" {
					return pathHint
				}
			}

			// lastly check the default server mux for a path hint
			_, pathHint = http.DefaultServeMux.Handler(r)

			return pathHint
		})
	})
}

// GinMiddleware setups up the default SDK instance to start capturing requests from the gin http framework.
func GinMiddleware(c *gin.Context) {
	defaultInstance.GinMiddleware(c)
}

// GinMiddleware setups the current instance of the SDK to start capturing requests from the gin http framework.
func (s *Speakeasy) GinMiddleware(c *gin.Context) {
	s.handleRequestResponse(c.Writer, c.Request, func(w http.ResponseWriter, r *http.Request) {
		c.Writer = &ginResponseWriter{c.Writer, w}
		c.Request = r

		c.Next()
	}, func(c *gin.Context) func(r *http.Request) string {
		return func(r *http.Request) string {
			return c.FullPath()
		}
	}(c))
}

// EchoMiddleware setups up the default SDK instance to start capturing requests from the echo http framework.
func EchoMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return defaultInstance.EchoMiddleware(next)
}

// EchoMiddleware setups the current instance of the SDK to start capturing requests from the echo http framework.
func (s *Speakeasy) EchoMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		return s.handleRequestResponseError(c.Response(), c.Request(), func(w http.ResponseWriter, r *http.Request) error {
			c.SetResponse(echo.NewResponse(w, c.Echo()))
			c.SetRequest(r)

			return next(c)
		}, func(r *http.Request) string {
			return c.Path()
		})
	}
}

type ginResponseWriter struct {
	gin.ResponseWriter
	writer http.ResponseWriter
}

var _ gin.ResponseWriter = &ginResponseWriter{}

func (g *ginResponseWriter) Write(data []byte) (int, error) {
	return g.writer.Write(data)
}

func (g *ginResponseWriter) WriteString(s string) (int, error) {
	return g.writer.Write([]byte(s))
}

func (g *ginResponseWriter) WriteHeader(statusCode int) {
	g.writer.WriteHeader(statusCode)
}
