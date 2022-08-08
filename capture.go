package speakeasy

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/chromedp/cdproto/har"
	"github.com/speakeasy-api/speakeasy-go-sdk/internal/log"
	"github.com/speakeasy-api/speakeasy-go-sdk/internal/pathhints"
	"github.com/speakeasy-api/speakeasy-schemas/grpc/go/registry/ingest"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
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
	startTime := timeNow()

	cw := NewCaptureWriter(w, maxCaptureSize)

	if r.Body != nil {
		// We need to duplicate the request body, because it should be consumed by the next handler first before we can read it
		// (as io.Reader is a stream and can only be read once) but we are potentially storing a large request (such as a file upload)
		// in memory, so we may need to allow the middleware to be configured to not read the body or have a max size
		tee := io.TeeReader(r.Body, cw.GetRequestWriter())
		r.Body = ioutil.NopCloser(tee)
	}

	ctx, c := contextWithController(r.Context())
	r = r.WithContext(ctx)

	err := next(cw.GetResponseWriter(), r)

	pathHint := capturePathHint(r)
	pathHint = pathhints.NormalizePathHint(pathHint)

	// if developer has provided a path hint use it, otherwise use the pathHint from the request
	if c.pathHint != "" {
		pathHint = c.pathHint
	}

	// Assuming response is done
	go s.captureRequestResponse(cw, r, startTime, pathHint)

	return err
}

func (s *Speakeasy) captureRequestResponse(cw *captureWriter, r *http.Request, startTime time.Time, pathHint string) {
	var ctx context.Context = valueOnlyContext{r.Context()}

	if cw.IsReqValid() && cw.GetReqBuffer().Len() == 0 && r.Body != nil {
		// Read the body just in case it was not read in the handler
		//nolint: errcheck
		io.Copy(ioutil.Discard, r.Body)
	}

	opts := []grpc.DialOption{}

	if s.secure {
		//nolint: gosec
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

	harData, err := json.Marshal(s.buildHarFile(ctx, cw, r, startTime))
	if err != nil {
		log.From(ctx).Error("speakeasy-sdk: failed to ingest request body", zap.Error(err))
		return
	}

	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("x-api-key", s.config.APIKey))

	_, err = ingest.NewIngestServiceClient(conn).Ingest(ctx, &ingest.IngestRequest{
		Har:       string(harData),
		PathHint:  pathHint,
		ApiId:     s.config.ApiID,
		VersionId: s.config.VersionID,
	})
	if err != nil {
		log.From(ctx).Error("speakeasy-sdk: failed to send ingest request", zap.Error(err))
		return
	}
}

func (s *Speakeasy) buildHarFile(ctx context.Context, cw *captureWriter, r *http.Request, startTime time.Time) *har.HAR {
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
					StartedDateTime: startTime.Format(time.RFC3339Nano),
					Time:            float64(timeSince(startTime).Milliseconds()),
					Request:         s.getHarRequest(ctx, cw, r),
					Response:        s.getHarResponse(ctx, cw, r, startTime),
					Connection:      r.URL.Port(),
					ServerIPAddress: r.URL.Hostname(),
				},
			},
		},
	}
}

//nolint:cyclop,funlen
func (s *Speakeasy) getHarRequest(ctx context.Context, cw *captureWriter, r *http.Request) *har.Request {
	reqHeaders := []*har.NameValuePair{}
	for k, v := range r.Header {
		for _, vv := range v {
			reqHeaders = append(reqHeaders, &har.NameValuePair{Name: k, Value: vv})
		}
	}

	reqQueryParams := []*har.NameValuePair{}

	for k, v := range r.URL.Query() {
		for _, vv := range v {
			reqQueryParams = append(reqQueryParams, &har.NameValuePair{Name: k, Value: vv})
		}
	}

	reqCookies := getHarCookies(r.Cookies(), time.Time{})
	bodyText := "--dropped--"
	if cw.IsReqValid() {
		bodyText = cw.GetReqBuffer().String()
	}

	hw := httptest.NewRecorder()

	for k, vv := range r.Header {
		for _, v := range vv {
			hw.Header().Set(k, v)
		}
	}

	b := bytes.NewBuffer([]byte{})
	headerSize := -1
	if err := hw.Header().Write(b); err != nil {
		log.From(ctx).Error("speakeasy-sdk: failed to read length of request headers", zap.Error(err))
	} else {
		headerSize = b.Len()
	}

	var postData *har.PostData
	if len(bodyText) > 0 {
		reqContentType := r.Header.Get("Content-Type")
		if reqContentType == "" {
			reqContentType = http.DetectContentType(cw.GetReqBuffer().Bytes())
			if reqContentType == "" {
				reqContentType = "application/octet-stream" // default http content type
			}
		}

		postData = &har.PostData{
			MimeType: reqContentType,
			Text:     bodyText,
			Params:   []*har.Param{}, // We don't parse the body here to populate this
		}
	}

	return &har.Request{
		Method:      r.Method,
		URL:         r.URL.String(),
		Headers:     reqHeaders,
		QueryString: reqQueryParams,
		BodySize:    r.ContentLength,
		PostData:    postData,
		HTTPVersion: r.Proto,
		Cookies:     reqCookies,
		HeadersSize: int64(headerSize),
	}
}

func (s *Speakeasy) getHarResponse(ctx context.Context, cw *captureWriter, r *http.Request, startTime time.Time) *har.Response {
	resHeaders := []*har.NameValuePair{}

	cookieParser := http.Response{Header: http.Header{}}

	for k, v := range cw.origResW.Header() {
		for _, vv := range v {
			if k == "Set-Cookie" {
				cookieParser.Header.Add(k, vv)
			}

			resHeaders = append(resHeaders, &har.NameValuePair{Name: k, Value: vv})
		}
	}

	resCookies := getHarCookies(cookieParser.Cookies(), startTime)

	resContentType := cw.origResW.Header().Get("Content-Type")
	if resContentType == "" {
		resContentType = "application/octet-stream" // default http content type
	}

	bodyText := "--dropped--"
	contentBodySize := 0
	if cw.IsResValid() {
		bodyText = cw.GetResBuffer().String()
		contentBodySize = cw.GetResBuffer().Len()
	}

	bodySize := cw.GetResponseSize()
	if cw.GetStatus() == http.StatusNotModified {
		bodySize = 0
	}

	b := bytes.NewBuffer([]byte{})
	headerSize := -1
	if err := cw.origResW.Header().Write(b); err != nil {
		log.From(ctx).Error("speakeasy-sdk: failed to read length of response headers", zap.Error(err))
	} else {
		headerSize = b.Len()
	}

	return &har.Response{
		Status:      int64(cw.status),
		StatusText:  http.StatusText(cw.status),
		HTTPVersion: r.Proto,
		Headers:     resHeaders,
		Cookies:     resCookies,
		Content: &har.Content{ // we are assuming we are getting the raw response here, so if we are put in the chain such that compression or encoding happens then the response text will be unreadable
			Size:     int64(contentBodySize),
			MimeType: resContentType,
			Text:     bodyText,
		},
		RedirectURL: cw.origResW.Header().Get("Location"),
		HeadersSize: int64(headerSize),
		BodySize:    int64(bodySize),
	}
}

func getHarCookies(cookies []*http.Cookie, startTime time.Time) []*har.Cookie {
	harCookies := []*har.Cookie{}
	for _, cookie := range cookies {
		harCookie := &har.Cookie{
			Name:     cookie.Name,
			Value:    cookie.Value,
			Path:     cookie.Path,
			Domain:   cookie.Domain,
			Secure:   cookie.Secure,
			HTTPOnly: cookie.HttpOnly,
		}

		if cookie.MaxAge != 0 {
			harCookie.Expires = startTime.Add(time.Duration(cookie.MaxAge) * time.Second).Format(time.RFC3339)
		} else if (cookie.Expires != time.Time{}) {
			harCookie.Expires = cookie.Expires.Format(time.RFC3339)
		}

		harCookies = append(harCookies, harCookie)
	}

	return harCookies
}

// This allows us to not be affected by context cancellation of the request that spawned our request capture while still retaining any context values.
//
//nolint:containedctx
type valueOnlyContext struct{ context.Context }

//nolint:nonamedreturns
func (valueOnlyContext) Deadline() (deadline time.Time, ok bool) { return }
func (valueOnlyContext) Done() <-chan struct{}                   { return nil }
func (valueOnlyContext) Err() error                              { return nil }
