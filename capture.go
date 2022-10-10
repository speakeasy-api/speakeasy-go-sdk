package speakeasy

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/speakeasy-api/speakeasy-go-sdk/internal/log"
	"github.com/speakeasy-api/speakeasy-go-sdk/internal/pathhints"
	"github.com/speakeasy-api/speakeasy-schemas/grpc/go/registry/ingest"
	"go.uber.org/zap"
)

var maxCaptureSize = 1 * 1024 * 1024

var timeNow = func() time.Time {
	return time.Now()
}

var timeSince = func(t time.Time) time.Duration {
	return time.Since(t)
}

type handlerFunc func(http.ResponseWriter, *http.Request) error

func (s *Speakeasy) handleRequestResponse(w http.ResponseWriter, r *http.Request, next http.HandlerFunc, capturePathHint func(r *http.Request) string) {
	err := s.handleRequestResponseError(w, r, func(w http.ResponseWriter, r *http.Request) error {
		next.ServeHTTP(w, r)
		return nil
	}, capturePathHint)
	if err != nil {
		log.Logger().Error("speakeasy-sdk: unexpected error from non-error handlerFunc", zap.Error(err))
	}
}

func (s *Speakeasy) handleRequestResponseError(w http.ResponseWriter, r *http.Request, next handlerFunc, capturePathHint func(r *http.Request) string) error {
	//nolint:ifshort
	startTime := timeNow()

	cw := NewCaptureWriter(w, maxCaptureSize)

	if r.Body != nil {
		// We need to duplicate the request body, because it should be consumed by the next handler first before we can read it
		// (as io.Reader is a stream and can only be read once) but we are potentially storing a large request (such as a file upload)
		// in memory, so we may need to allow the middleware to be configured to not read the body or have a max size
		tee := io.TeeReader(r.Body, cw.GetRequestWriter())
		r.Body = ioutil.NopCloser(tee)
	}

	ctx, c := contextWithController(r.Context(), s)
	r = r.WithContext(ctx)

	err := next(cw.GetResponseWriter(), r)

	pathHint := capturePathHint(r)
	pathHint = pathhints.NormalizePathHint(pathHint)

	// if developer has provided a path hint use it, otherwise use the pathHint from the request
	if c.pathHint != "" {
		pathHint = c.pathHint
	}

	// Used for load testing: set this to true and the capture GRPC call is invoked inline.
	// This will cause the endpoint latency to be added to the GRPC request/response latency
	if os.Getenv("SPEAKEASY_SDK_CAPTURE_INLINE") == "true" {
		s.captureRequestResponse(cw, r, startTime, pathHint, c)
	} else {
		go s.captureRequestResponse(cw, r, startTime, pathHint, c)
	}
	return err
}

//nolint:nolintlint,contextcheck
func (s *Speakeasy) captureRequestResponse(cw *captureWriter, r *http.Request, startTime time.Time, pathHint string, c *controller) {
	var ctx context.Context = valueOnlyContext{r.Context()}

	if cw.IsReqValid() && cw.GetReqBuffer().Len() == 0 && r.Body != nil {
		// Read the body just in case it was not read in the handler
		//nolint: errcheck
		io.Copy(ioutil.Discard, r.Body)
	}

	harData, err := json.Marshal(s.harBuilder.buildHarFile(ctx, cw, r, startTime, c))
	if err != nil {
		log.From(ctx).Error("speakeasy-sdk: failed to create har file", zap.Error(err))
		return
	}

	s.grpcClient.SendToIngest(ctx, &ingest.IngestRequest{
		Har:        string(harData),
		PathHint:   pathHint,
		ApiId:      s.config.ApiID,
		VersionId:  s.config.VersionID,
		CustomerId: c.customerID,
		//nolint:nosnakecase
		MaskingMetadata: &ingest.IngestRequest_MaskingMetadata{
			QueryStringMasks:         c.queryStringMasks,
			RequestHeaderMasks:       c.requestHeaderMasks,
			RequestCookieMasks:       c.requestCookieMasks,
			RequestFieldMasksString:  c.requestFieldMasksString,
			RequestFieldMasksNumber:  c.requestFieldMasksNumber,
			ResponseHeaderMasks:      c.responseHeaderMasks,
			ResponseCookieMasks:      c.responseCookieMasks,
			ResponseFieldMasksString: c.responseFieldMasksString,
			ResponseFieldMasksNumber: c.responseFieldMasksNumber,
		},
	})
}

// This allows us to not be affected by context cancellation of the request that spawned our request capture while still retaining any context values.
//
//nolint:containedctx
type valueOnlyContext struct{ context.Context }

//nolint:nonamedreturns
func (valueOnlyContext) Deadline() (deadline time.Time, ok bool) { return }
func (valueOnlyContext) Done() <-chan struct{}                   { return nil }
func (valueOnlyContext) Err() error                              { return nil }
