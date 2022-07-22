package speakeasy

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/chromedp/cdproto/har"
	"github.com/speakeasy-api/speakeasy-go-sdk/internal/log"
	"github.com/speakeasy-api/speakeasy-schemas/grpc/go/registry/ingest"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

var timeNow = func() time.Time {
	return time.Now()
}

var timeSince = func(t time.Time) time.Duration {
	return time.Since(t)
}

type handlerFunc func(http.ResponseWriter, *http.Request) error

func (s *speakeasy) handleRequestResponse(w http.ResponseWriter, r *http.Request, next http.HandlerFunc, capturePathHint func(r *http.Request) string) {
	err := s.handleRequestResponseError(w, r, func(w http.ResponseWriter, r *http.Request) error {
		next.ServeHTTP(w, r)
		return nil
	}, capturePathHint)
	if err != nil {
		log.Logger().Error("speakeasy-sdk: unexpected error from non-error handlerFunc", zap.Error(err))
	}
}

func (s *speakeasy) handleRequestResponseError(w http.ResponseWriter, r *http.Request, next handlerFunc, capturePathHint func(r *http.Request) string) error {
	startTime := timeNow()

	responseBuf := &bytes.Buffer{}
	if r.Body != nil {
		// We need to duplicate the request body, because it should be consumed by the next handler first before we can read it
		// (as io.Reader is a stream and can only be read once) but we are potentially storing a large request (such as a file upload)
		// in memory, so we may need to allow the middleware to be configured to not read the body or have a max size
		tee := io.TeeReader(r.Body, responseBuf)
		r.Body = ioutil.NopCloser(tee)
	}

	swr := newResponseWriter(w)

	ctx, c := contextWithController(r.Context())
	r = r.WithContext(ctx)

	err := next(swr, r)

	pathHint := capturePathHint(r)

	// if developer has provided a path hint use it, otherwise use the pathHint from the request
	if c.pathHint != "" {
		pathHint = c.pathHint
	}

	// Assuming response is done
	go s.captureRequestResponse(swr, responseBuf, r, startTime, pathHint)

	return err
}

func (s *speakeasy) captureRequestResponse(swr *speakeasyResponseWriter, resBuf *bytes.Buffer, r *http.Request, startTime time.Time, pathHint string) {
	var ctx context.Context = valueOnlyContext{r.Context()}

	if !swr.valid {
		log.From(ctx).Error("speakeasy-sdk: failed to capture request response")
		return
	}

	if resBuf.Len() == 0 && r.Body != nil {
		// Read the body just in case it was not read in the handler
		if _, err := io.Copy(ioutil.Discard, r.Body); err != nil {
			log.From(ctx).Error("speakeasy-sdk: failed to read request body", zap.Error(err))
		}
	}

	opts := []grpc.DialOption{}

	if s.secure {
		// nolint: gosec
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	if s.config.GRPCDialer != nil {
		opts = append(opts, grpc.WithContextDialer(s.config.GRPCDialer()))
	}

	conn, err := grpc.DialContext(ctx, s.serverURL, opts...)
	if err != nil {
		log.From(ctx).Error("speakeasy-sdk: failed to create grpc connection", zap.Error(err))
		return
	}
	defer conn.Close()

	harData, err := json.Marshal(s.buildHarFile(swr, resBuf.String(), r, startTime))
	if err != nil {
		log.From(ctx).Error("speakeasy-sdk: failed to ingest request body", zap.Error(err))
		return
	}

	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("x-api-key", s.config.APIKey))

	_, err = ingest.NewIngestServiceClient(conn).Ingest(ctx, &ingest.IngestRequest{
		Har:      string(harData),
		PathHint: pathHint,
	})
	if err != nil {
		log.From(ctx).Error("speakeasy-sdk: failed to send ingest request", zap.Error(err))
		return
	}
}

func (s *speakeasy) buildHarFile(swr *speakeasyResponseWriter, resBody string, r *http.Request, startTime time.Time) *har.HAR {
	return &har.HAR{
		Log: &har.Log{
			Version: "1.2",
			Creator: &har.Creator{
				Name:    sdkName,
				Version: speakeasyVersion,
			},
			Comment: "request capture for " + r.URL.String(),
			Entries: []*har.Entry{
				{
					StartedDateTime: startTime.Format(time.RFC3339),
					Time:            timeSince(startTime).Seconds(),
					Request:         s.getHarRequest(r, resBody),
					Response:        s.getHarResponse(swr, r),
					Connection:      r.URL.Port(),
					ServerIPAddress: r.URL.Hostname(),
				},
			},
		},
	}
}

func (s *speakeasy) getHarRequest(r *http.Request, resBody string) *har.Request {
	reqHeaders := []*har.NameValuePair{}
	for k, v := range r.Header {
		if k != "Cookie" {
			for _, vv := range v {
				reqHeaders = append(reqHeaders, &har.NameValuePair{Name: k, Value: vv})
			}
		}
	}

	reqQueryParams := []*har.NameValuePair{}

	for k, v := range r.URL.Query() {
		for _, vv := range v {
			reqQueryParams = append(reqQueryParams, &har.NameValuePair{Name: k, Value: vv})
		}
	}

	reqCookies := []*har.Cookie{}

	for _, cookie := range r.Cookies() {
		reqCookies = append(reqCookies, &har.Cookie{
			Name:     cookie.Name,
			Value:    cookie.Value,
			Path:     cookie.Path,
			Domain:   cookie.Domain,
			Expires:  cookie.Expires.Format(time.RFC3339),
			Secure:   cookie.Secure,
			HTTPOnly: cookie.HttpOnly,
		})
	}

	reqContentType := r.Header.Get("Content-Type")
	if reqContentType == "" {
		reqContentType = "application/octet-stream" // default http content type
	}

	return &har.Request{
		Method:      r.Method,
		URL:         r.URL.String(),
		Headers:     reqHeaders,
		QueryString: reqQueryParams,
		BodySize:    r.ContentLength,
		PostData: &har.PostData{
			MimeType: reqContentType,
			Text:     resBody,
			Params:   nil, // We don't parse the body here to populate this
		},
		HTTPVersion: r.Proto,
		Cookies:     reqCookies,
		HeadersSize: -1, // TODO do we need to calculate this? If so we can get it with r.Header.Write to a bytes.Buffer and read size
	}
}

func (s *speakeasy) getHarResponse(swr *speakeasyResponseWriter, r *http.Request) *har.Response {
	resHeaders := []*har.NameValuePair{}
	cookieParser := http.Request{
		Header: http.Header{},
	}

	for k, v := range swr.Header() {
		for _, vv := range v {
			if k == "Set-Cookie" {
				cookieParser.Header.Add("Cookie", vv)
			} else {
				resHeaders = append(resHeaders, &har.NameValuePair{Name: k, Value: vv})
			}
		}
	}

	resCookies := []*har.Cookie{}
	for _, cookie := range cookieParser.Cookies() {
		resCookies = append(resCookies, &har.Cookie{
			Name:     cookie.Name,
			Value:    cookie.Value,
			Path:     cookie.Path,
			Domain:   cookie.Domain,
			Expires:  cookie.Expires.Format(time.RFC3339),
			Secure:   cookie.Secure,
			HTTPOnly: cookie.HttpOnly,
		})
	}

	resContentType := swr.Header().Get("Content-Type")
	if resContentType == "" {
		resContentType = "application/octet-stream" // default http content type
	}

	resBodySize := int64(swr.body.Len())
	if swr.status == http.StatusNotModified {
		resBodySize = 0
	}

	return &har.Response{
		Status:      int64(swr.status),
		StatusText:  http.StatusText(swr.status),
		HTTPVersion: r.Proto,
		Headers:     resHeaders,
		Cookies:     resCookies,
		Content: &har.Content{ // we are assuming we are getting the raw response here, so if we are put in the chain such that compression or encoding happens then the response text will be unreadable
			Size:     int64(swr.body.Len()),
			MimeType: resContentType,
			Text:     swr.body.String(),
		},
		RedirectURL: swr.Header().Get("Location"),
		HeadersSize: -1,
		BodySize:    resBodySize,
	}
}

// This allows us to not be affected by context cancellation of the request that spawned our request capture while still retaining any context values.
// nolint:containedctx
type valueOnlyContext struct{ context.Context }

// nolint
func (valueOnlyContext) Deadline() (deadline time.Time, ok bool) { return }
func (valueOnlyContext) Done() <-chan struct{}                   { return nil }
func (valueOnlyContext) Err() error                              { return nil }