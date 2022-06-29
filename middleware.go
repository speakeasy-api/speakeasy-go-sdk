package speakeasy

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"
	"github.com/speakeasy-api/speakeasy-go-sdk/internal/log"
	"go.uber.org/zap"
)

const (
	speakasyVersion = 0.1
	sdkName         = "go"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (rec *statusRecorder) WriteHeader(code int) {
	rec.status = code
	rec.ResponseWriter.WriteHeader(code)
}

func (app SpeakeasyApp) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		_, errReqInfo := getRequestInfo(r, startTime)

		// intercept the status code
		rec := statusRecorder{w, 200}

		next.ServeHTTP(&rec, r)

		ctx := log.WithFields(r.Context(), zap.Time("start_time", startTime), zap.String("method", r.Method), zap.String("request_uri", r.RequestURI), zap.Duration("request_duration", time.Since(startTime)))

		if !errors.Is(errReqInfo, ErrNotJson) {

			router, err := gorillamux.NewRouter(app.Schema)
			if err != nil {
				log.From(ctx).Error("failed to create router for OpenAPI schema", zap.Error(err))
				return
			}
			route, _, err := router.FindRoute(r)
			if !errors.Is(err, routers.ErrPathNotFound) {
				app.updateApiStatsByResponseStatus(route.Path, rec.status)
			} else {
				log.From(ctx).Error("failed to find schema path for request in router", zap.Error(err))
			}
		} else {
			log.From(ctx).Error("malformed request", zap.Error(errReqInfo))
		}
	})
}

func (app SpeakeasyApp) updateApiStatsByResponseStatus(path string, status int) {
	// TODO: Update number of unique customers here as well
	app.Lock.Lock()
	defer app.Lock.Unlock()

	apiId := app.ApiByPath[path].ID
	stats := app.ApiStatsById[apiId]
	stats.NumCalls += 1
	if status < 200 || status >= 300 {
		stats.NumErrors += 1
	}
	app.ApiStatsById[apiId] = stats
}

// If anything happens to go wrong inside one of speakasy-go-sdk internals, recover from panic and continue
func dontPanic(ctx context.Context) {
	if err := recover(); err != nil {
		log.From(ctx).Error(fmt.Sprintf("speakeasy sdk panic %s", err))
	}
}
