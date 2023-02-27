package speakeasy_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/chromedp/cdproto/har"
	"github.com/gin-gonic/gin"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/mux"
	"github.com/labstack/echo/v4"
	"github.com/speakeasy-api/speakeasy-go-sdk"
	"github.com/speakeasy-api/speakeasy-schemas/grpc/go/registry/ingest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
	httptrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/net/http"
)

func TestMain(m *testing.M) {
	rand.Seed(time.Now().UnixNano())

	//nolint:tenv,nolintlint
	os.Setenv("SPEAKEASY_SERVER_SECURE", "false")
	gin.SetMode(gin.ReleaseMode)
	os.Exit(m.Run())
}

type header struct {
	Key    string   `json:"key"`
	Values []string `json:"values"`
}
type fields struct {
	MaxCaptureSize int `json:"max_capture_size,omitempty"`
}
type args struct {
	Method                   string            `json:"method"`
	URL                      string            `json:"url"`
	Headers                  []header          `json:"headers"`
	Body                     string            `json:"body"`
	RequestStartTime         time.Time         `json:"request_start_time"`
	ElapsedTime              int               `json:"elapsed_time"`
	ResponseStatus           int               `json:"response_status"`
	ResponseBody             string            `json:"response_body"`
	ResponseHeaders          []header          `json:"response_headers"`
	QueryStringMasks         map[string]string `json:"query_string_masks"`
	RequestHeaderMasks       map[string]string `json:"request_header_masks"`
	RequestCookieMasks       map[string]string `json:"request_cookie_masks"`
	RequestFieldMasksString  map[string]string `json:"request_field_masks_string"`
	RequestFieldMasksNumber  map[string]string `json:"request_field_masks_number"`
	ResponseHeaderMasks      map[string]string `json:"response_header_masks"`
	ResponseCookieMasks      map[string]string `json:"response_cookie_masks"`
	ResponseFieldMasksString map[string]string `json:"response_field_masks_string"`
	ResponseFieldMasksNumber map[string]string `json:"response_field_masks_number"`
}
type test struct {
	Name    string `json:"name"`
	Fields  fields `json:"fields"`
	Args    args   `json:"args"`
	WantHAR string `json:"want_har"`
}

const (
	testAPIKey    = "test"
	testApiID     = "testapi1"
	testVersionID = "v1.0.0"
)

func loadTestData(t *testing.T) []test {
	t.Helper()

	files, err := os.ReadDir("testdata")
	require.NoError(t, err)

	tests := []test{}
	for _, file := range files {
		if strings.HasSuffix(file.Name(), "_input.json") {
			baseName := strings.TrimSuffix(file.Name(), "_input.json")

			inputData, err := os.ReadFile("testdata/" + file.Name())
			require.NoError(t, err)

			tt := test{}
			err = json.Unmarshal(inputData, &tt)
			require.NoError(t, err)

			outputData, err := os.ReadFile("testdata/" + baseName + "_output.json")
			require.NoError(t, err)

			outputDataMinified := bytes.NewBuffer([]byte{})
			err = json.Compact(outputDataMinified, outputData)
			require.NoError(t, err)

			tt.WantHAR = outputDataMinified.String()

			tests = append(tests, tt)
		}
	}

	return tests
}

func TestSpeakeasy_Middleware_Capture_Success(t *testing.T) {
	tests := loadTestData(t)
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			speakeasy.ExportSetMaxCaptureSize(tt.Fields.MaxCaptureSize)

			captured := false
			handled := false

			if tt.Args.RequestStartTime.IsZero() {
				speakeasy.ExportSetTimeNow(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))
			} else {
				speakeasy.ExportSetTimeNow(tt.Args.RequestStartTime)
			}
			if tt.Args.ElapsedTime == 0 {
				speakeasy.ExportSetTimeSince(1 * time.Millisecond)
			} else {
				speakeasy.ExportSetTimeSince(time.Duration(tt.Args.ElapsedTime) * time.Millisecond)
			}

			wg := &sync.WaitGroup{}
			wg.Add(1)

			sdkInstance := speakeasy.New(speakeasy.Config{
				APIKey:    testAPIKey,
				ApiID:     testApiID,
				VersionID: testVersionID,
				GRPCDialer: dialer(func(ctx context.Context, req *ingest.IngestRequest) {
					md, ok := metadata.FromIncomingContext(ctx)
					assert.True(t, ok)

					apiKeys := md.Get("x-api-key")
					assert.Contains(t, apiKeys, testAPIKey)

					assert.Equal(t, testApiID, req.ApiId)
					assert.Equal(t, testVersionID, req.VersionId)

					assert.JSONEq(t, tt.WantHAR, req.Har)
					captured = true
					wg.Done()
				}),
			})

			h := sdkInstance.Middleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				ctrl, _ := speakeasy.MiddlewareController(req)

				if tt.Args.QueryStringMasks != nil {
					for k, v := range tt.Args.QueryStringMasks {
						ctrl.Masking(speakeasy.WithQueryStringMask([]string{k}, v))
					}
				}

				if tt.Args.RequestHeaderMasks != nil {
					for k, v := range tt.Args.RequestHeaderMasks {
						ctrl.Masking(speakeasy.WithRequestHeaderMask([]string{k}, v))
					}
				}

				if tt.Args.RequestCookieMasks != nil {
					for k, v := range tt.Args.RequestCookieMasks {
						ctrl.Masking(speakeasy.WithRequestCookieMask([]string{k}, v))
					}
				}

				if tt.Args.RequestFieldMasksString != nil {
					for k, v := range tt.Args.RequestFieldMasksString {
						ctrl.Masking(speakeasy.WithRequestFieldMaskString([]string{k}, v))
					}
				}

				if tt.Args.RequestFieldMasksNumber != nil {
					for k, v := range tt.Args.RequestFieldMasksNumber {
						ctrl.Masking(speakeasy.WithRequestFieldMaskNumber([]string{k}, v))
					}
				}

				if tt.Args.ResponseHeaderMasks != nil {
					for k, v := range tt.Args.ResponseHeaderMasks {
						ctrl.Masking(speakeasy.WithResponseHeaderMask([]string{k}, v))
					}
				}

				if tt.Args.ResponseCookieMasks != nil {
					for k, v := range tt.Args.ResponseCookieMasks {
						ctrl.Masking(speakeasy.WithResponseCookieMask([]string{k}, v))
					}
				}

				if tt.Args.ResponseFieldMasksString != nil {
					for k, v := range tt.Args.ResponseFieldMasksString {
						ctrl.Masking(speakeasy.WithResponseFieldMaskString([]string{k}, v))
					}
				}

				if tt.Args.ResponseFieldMasksNumber != nil {
					for k, v := range tt.Args.ResponseFieldMasksNumber {
						ctrl.Masking(speakeasy.WithResponseFieldMaskNumber([]string{k}, v))
					}
				}

				for _, header := range tt.Args.ResponseHeaders {
					for _, val := range header.Values {
						w.Header().Add(header.Key, val)
					}
				}

				if req.Body != nil {
					data, err := io.ReadAll(req.Body)
					assert.NoError(t, err)
					assert.Equal(t, tt.Args.Body, string(data))
				}

				if tt.Args.ResponseStatus > 0 {
					w.WriteHeader(tt.Args.ResponseStatus)
				}

				if tt.Args.ResponseBody != "" {
					_, err := w.Write([]byte(tt.Args.ResponseBody))
					assert.NoError(t, err)
				}
				handled = true
			}))

			w := httptest.NewRecorder()

			var req *http.Request
			var err error
			if tt.Args.Body == "" {
				req, err = http.NewRequest(tt.Args.Method, tt.Args.URL, nil)
			} else {
				req, err = http.NewRequest(tt.Args.Method, tt.Args.URL, bytes.NewBuffer([]byte(tt.Args.Body)))
			}
			assert.NoError(t, err)

			for _, header := range tt.Args.Headers {
				for _, val := range header.Values {
					req.Header.Add(header.Key, val)
				}
			}

			h.ServeHTTP(w, req)

			wg.Wait()

			assert.True(t, handled, "middleware did not call handler")
			assert.True(t, captured, "middleware did not capture request")

			responseStatus := http.StatusOK
			if tt.Args.ResponseStatus > 0 {
				responseStatus = tt.Args.ResponseStatus
			}

			assert.Equal(t, responseStatus, w.Code)
		})
	}
}

func TestSpeakeasy_Middleware_URL_Resolve_Success(t *testing.T) {
	type args struct {
		url     string
		headers map[string]string
		host    string
		https   bool
	}
	tests := []struct {
		name            string
		args            args
		wantResolvedURL string
	}{
		{
			name: "successfully resolves relative URL",
			args: args{
				url:  "/v1/users",
				host: "localhost:8080",
			},
			wantResolvedURL: "http://localhost:8080/v1/users",
		},
		{
			name: "successfully resolves relative HTTPS URL",
			args: args{
				url:   "/v1/users",
				host:  "localhost:8080",
				https: true,
			},
			wantResolvedURL: "https://localhost:8080/v1/users",
		},
		{
			name: "successfully resolves absolute URL",
			args: args{
				url: "https://localhost:8080/v1/users",
			},
			wantResolvedURL: "https://localhost:8080/v1/users",
		},
		{
			name: "successfully resolves relative URL behind proxy",
			args: args{
				url:  "/v1/users",
				host: "localhost:8080",
				headers: map[string]string{
					"X-Forwarded-Host":  "dev.speakeasyapi.dev",
					"X-Forwarded-Proto": "https",
				},
			},
			wantResolvedURL: "https://dev.speakeasyapi.dev/v1/users",
		},
		{
			name: "successfully resolves absolute URL behind proxy",
			args: args{
				url: "http://10.0.0.1:8080/v1/users",
				headers: map[string]string{
					"X-Forwarded-Host":  "dev.speakeasyapi.dev",
					"X-Forwarded-Proto": "https",
				},
			},
			wantResolvedURL: "https://dev.speakeasyapi.dev/v1/users",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			captured := false
			handled := false

			speakeasy.ExportSetTimeNow(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))
			speakeasy.ExportSetTimeSince(1 * time.Millisecond)

			wg := &sync.WaitGroup{}
			wg.Add(1)

			sdkInstance := speakeasy.New(speakeasy.Config{
				APIKey:    testAPIKey,
				ApiID:     testApiID,
				VersionID: testVersionID,
				GRPCDialer: dialer(func(ctx context.Context, req *ingest.IngestRequest) {
					var h har.HAR

					err := json.Unmarshal([]byte(req.GetHar()), &h)
					require.NoError(t, err)

					assert.Equal(t, tt.wantResolvedURL, h.Log.Entries[0].Request.URL)
					captured = true
					wg.Done()
				}),
			})

			r := mux.NewRouter()
			r.Use(sdkInstance.Middleware)

			r.Methods(http.MethodGet).Path("/v1/users").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.WriteHeader(http.StatusOK)
				handled = true
			})

			w := httptest.NewRecorder()

			req, err := http.NewRequest(http.MethodGet, tt.args.url, nil)
			assert.NoError(t, err)

			if tt.args.host != "" {
				req.Host = tt.args.host
			}
			for k, v := range tt.args.headers {
				req.Header.Add(k, v)
			}
			if tt.args.https {
				req.TLS = &tls.ConnectionState{}
			}

			r.ServeHTTP(w, req)

			wg.Wait()

			assert.True(t, handled, "middleware did not call handler")
			assert.True(t, captured, "middleware did not capture request")
			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

func TestSpeakeasy_Middleware_GorillaMux_PathHint_Success(t *testing.T) {
	type args struct {
		path    string
		url     string
		devHint string
	}
	tests := []struct {
		name         string
		args         args
		wantPathHint string
	}{
		{
			name: "captures simple path hint from mux",
			args: args{
				path: "/user/{id}",
				url:  "http://test.com/user/1",
			},
			wantPathHint: "/user/{id}",
		},
		{
			name: "captures more complex path hint from mux",
			args: args{
				path: "/user/{id:[0-9]+}/account/{accountID}",
				url:  "http://test.com/user/1/account/abcdefg",
			},
			wantPathHint: "/user/{id}/account/{accountID}",
		},
		{
			name: "path hint is overridden by dev hint",
			args: args{
				path:    "/user/{id:[0-9]+}/account/{accountID}",
				url:     "http://test.com/user/1/account/abcdefg",
				devHint: "/user/{id}/account/{accountID}",
			},
			wantPathHint: "/user/{id}/account/{accountID}",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			captured := false
			handled := false

			speakeasy.ExportSetTimeNow(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))
			speakeasy.ExportSetTimeSince(1 * time.Millisecond)

			wg := &sync.WaitGroup{}
			wg.Add(1)

			sdkInstance := speakeasy.New(speakeasy.Config{
				APIKey:    testAPIKey,
				ApiID:     testApiID,
				VersionID: testVersionID,
				GRPCDialer: dialer(func(ctx context.Context, req *ingest.IngestRequest) {
					assert.Equal(t, tt.wantPathHint, req.PathHint)
					captured = true
					wg.Done()
				}),
			})

			r := mux.NewRouter()
			r.Use(sdkInstance.Middleware)

			r.Methods(http.MethodGet).Path(tt.args.path).HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if tt.args.devHint != "" {
					ctrl, _ := speakeasy.MiddlewareController(req)
					require.NotNil(t, ctrl)
					ctrl.PathHint(tt.args.devHint)
				}

				w.WriteHeader(http.StatusOK)
				handled = true
			})

			w := httptest.NewRecorder()

			req, err := http.NewRequest(http.MethodGet, tt.args.url, nil)
			assert.NoError(t, err)

			r.ServeHTTP(w, req)

			wg.Wait()

			assert.True(t, handled, "middleware did not call handler")
			assert.True(t, captured, "middleware did not capture request")
			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

func TestSpeakeasy_Middleware_Chi_PathHint_Success(t *testing.T) {
	type args struct {
		path string
		url  string
	}
	tests := []struct {
		name         string
		args         args
		wantPathHint string
	}{
		{
			name: "captures simple path hint from chi",
			args: args{
				path: "/user/{id}",
				url:  "http://test.com/user/1",
			},
			wantPathHint: "/user/{id}",
		},
		{
			name: "captures complex path hint from chi",
			args: args{
				path: "/user/{id}/account/*",
				url:  "http://test.com/user/abcdefg/account/1",
			},
			wantPathHint: "/user/{id}/account/{wildcard}",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			captured := false
			handled := false

			speakeasy.ExportSetTimeNow(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))
			speakeasy.ExportSetTimeSince(1 * time.Millisecond)

			wg := &sync.WaitGroup{}
			wg.Add(1)

			sdkInstance := speakeasy.New(speakeasy.Config{
				APIKey:    testAPIKey,
				ApiID:     testApiID,
				VersionID: testVersionID,
				GRPCDialer: dialer(func(ctx context.Context, req *ingest.IngestRequest) {
					assert.Equal(t, tt.wantPathHint, req.PathHint)
					captured = true
					wg.Done()
				}),
			})

			r := chi.NewRouter()
			r.Use(sdkInstance.Middleware)

			r.Get(tt.args.path, func(w http.ResponseWriter, req *http.Request) {
				w.WriteHeader(http.StatusOK)
				handled = true
			})

			w := httptest.NewRecorder()

			req, err := http.NewRequest(http.MethodGet, tt.args.url, nil)
			assert.NoError(t, err)

			r.ServeHTTP(w, req)

			wg.Wait()

			assert.True(t, handled, "middleware did not call handler")
			assert.True(t, captured, "middleware did not capture request")
			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

func TestSpeakeasy_Middleware_ServerMux_PathHint_Success(t *testing.T) {
	type args struct {
		path string
		url  string
	}
	tests := []struct {
		name         string
		args         args
		wantPathHint string
	}{
		{
			name: "captures simple path hint from DefaultServerMux",
			args: args{
				path: "/user",
				url:  "http://test.com/user",
			},
			wantPathHint: "/user",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			captured := false
			handled := false

			speakeasy.ExportSetTimeNow(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))
			speakeasy.ExportSetTimeSince(1 * time.Millisecond)

			wg := &sync.WaitGroup{}
			wg.Add(1)

			sdkInstance := speakeasy.New(speakeasy.Config{
				APIKey:    testAPIKey,
				ApiID:     testApiID,
				VersionID: testVersionID,
				GRPCDialer: dialer(func(ctx context.Context, req *ingest.IngestRequest) {
					assert.Equal(t, tt.wantPathHint, req.PathHint)
					captured = true
					wg.Done()
				}),
			})

			r := http.DefaultServeMux

			r.Handle(tt.args.path, sdkInstance.Middleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.WriteHeader(http.StatusOK)
				handled = true
			})))

			w := httptest.NewRecorder()

			req, err := http.NewRequest(http.MethodGet, tt.args.url, nil)
			assert.NoError(t, err)

			r.ServeHTTP(w, req)

			wg.Wait()

			assert.True(t, handled, "middleware did not call handler")
			assert.True(t, captured, "middleware did not capture request")
			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

func TestSpeakeasy_GinMiddleware_Success(t *testing.T) {
	tests := loadTestData(t)
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			speakeasy.ExportSetMaxCaptureSize(tt.Fields.MaxCaptureSize)

			captured := false
			handled := false

			if tt.Args.RequestStartTime.IsZero() {
				speakeasy.ExportSetTimeNow(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))
			} else {
				speakeasy.ExportSetTimeNow(tt.Args.RequestStartTime)
			}
			if tt.Args.ElapsedTime == 0 {
				speakeasy.ExportSetTimeSince(1 * time.Millisecond)
			} else {
				speakeasy.ExportSetTimeSince(time.Duration(tt.Args.ElapsedTime) * time.Millisecond)
			}

			wg := &sync.WaitGroup{}
			wg.Add(1)

			sdkInstance := speakeasy.New(speakeasy.Config{
				APIKey:    testAPIKey,
				ApiID:     testApiID,
				VersionID: testVersionID,
				GRPCDialer: dialer(func(ctx context.Context, req *ingest.IngestRequest) {
					assert.JSONEq(t, tt.WantHAR, req.Har)
					captured = true
					wg.Done()
				}),
			})

			r := gin.Default()
			r.Use(sdkInstance.GinMiddleware)

			r.Any("/*path", func(ctx *gin.Context) {
				ctrl, _ := speakeasy.MiddlewareController(ctx.Request)

				if tt.Args.QueryStringMasks != nil {
					for k, v := range tt.Args.QueryStringMasks {
						ctrl.Masking(speakeasy.WithQueryStringMask([]string{k}, v))
					}
				}

				if tt.Args.RequestHeaderMasks != nil {
					for k, v := range tt.Args.RequestHeaderMasks {
						ctrl.Masking(speakeasy.WithRequestHeaderMask([]string{k}, v))
					}
				}

				if tt.Args.RequestCookieMasks != nil {
					for k, v := range tt.Args.RequestCookieMasks {
						ctrl.Masking(speakeasy.WithRequestCookieMask([]string{k}, v))
					}
				}

				if tt.Args.RequestFieldMasksString != nil {
					for k, v := range tt.Args.RequestFieldMasksString {
						ctrl.Masking(speakeasy.WithRequestFieldMaskString([]string{k}, v))
					}
				}

				if tt.Args.RequestFieldMasksNumber != nil {
					for k, v := range tt.Args.RequestFieldMasksNumber {
						ctrl.Masking(speakeasy.WithRequestFieldMaskNumber([]string{k}, v))
					}
				}

				if tt.Args.ResponseHeaderMasks != nil {
					for k, v := range tt.Args.ResponseHeaderMasks {
						ctrl.Masking(speakeasy.WithResponseHeaderMask([]string{k}, v))
					}
				}

				if tt.Args.ResponseCookieMasks != nil {
					for k, v := range tt.Args.ResponseCookieMasks {
						ctrl.Masking(speakeasy.WithResponseCookieMask([]string{k}, v))
					}
				}

				if tt.Args.ResponseFieldMasksString != nil {
					for k, v := range tt.Args.ResponseFieldMasksString {
						ctrl.Masking(speakeasy.WithResponseFieldMaskString([]string{k}, v))
					}
				}

				if tt.Args.ResponseFieldMasksNumber != nil {
					for k, v := range tt.Args.ResponseFieldMasksNumber {
						ctrl.Masking(speakeasy.WithResponseFieldMaskNumber([]string{k}, v))
					}
				}

				for _, header := range tt.Args.ResponseHeaders {
					for _, val := range header.Values {
						ctx.Writer.Header().Add(header.Key, val)
					}
				}

				if ctx.Request.Body != nil {
					data, err := io.ReadAll(ctx.Request.Body)
					assert.NoError(t, err)
					assert.Equal(t, tt.Args.Body, string(data))
				}

				if tt.Args.ResponseStatus > 0 {
					ctx.Writer.WriteHeader(tt.Args.ResponseStatus)
				}

				if tt.Args.ResponseBody != "" {
					_, err := ctx.Writer.Write([]byte(tt.Args.ResponseBody))
					assert.NoError(t, err)
				}
				handled = true
			})

			w := httptest.NewRecorder()

			var req *http.Request
			var err error
			if tt.Args.Body == "" {
				req, err = http.NewRequest(tt.Args.Method, tt.Args.URL, nil)
			} else {
				req, err = http.NewRequest(tt.Args.Method, tt.Args.URL, bytes.NewBuffer([]byte(tt.Args.Body)))
			}
			assert.NoError(t, err)

			for _, header := range tt.Args.Headers {
				for _, val := range header.Values {
					req.Header.Add(header.Key, val)
				}
			}

			r.ServeHTTP(w, req)

			wg.Wait()

			assert.True(t, handled, "middleware did not call handler")
			assert.True(t, captured, "middleware did not capture request")

			responseStatus := http.StatusOK
			if tt.Args.ResponseStatus > 0 {
				responseStatus = tt.Args.ResponseStatus
			}

			assert.Equal(t, responseStatus, w.Code)
		})
	}
}

func TestSpeakeasy_GinMiddleware_PathHint_Success(t *testing.T) {
	type args struct {
		path    string
		url     string
		devHint string
	}
	tests := []struct {
		name         string
		args         args
		wantPathHint string
	}{
		{
			name: "captures simple path hint from gin",
			args: args{
				path: "/user",
				url:  "http://test.com/user",
			},
			wantPathHint: "/user",
		},
		{
			name: "captures more complex path hint from gin",
			args: args{
				path: "/user/:id/*action",
				url:  "http://test.com/user/1/send",
			},
			wantPathHint: "/user/{id}/{action}",
		},
		{
			name: "path hint is overridden by dev hint",
			args: args{
				path:    "/user/:id/*action",
				url:     "http://test.com/user/1/sent",
				devHint: "/user/{id}/{action}",
			},
			wantPathHint: "/user/{id}/{action}",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			captured := false
			handled := false

			speakeasy.ExportSetTimeNow(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))
			speakeasy.ExportSetTimeSince(1 * time.Millisecond)

			wg := &sync.WaitGroup{}
			wg.Add(1)

			sdkInstance := speakeasy.New(speakeasy.Config{
				APIKey:    testAPIKey,
				ApiID:     testApiID,
				VersionID: testVersionID,
				GRPCDialer: dialer(func(ctx context.Context, req *ingest.IngestRequest) {
					assert.Equal(t, tt.wantPathHint, req.PathHint)
					captured = true
					wg.Done()
				}),
			})

			r := gin.Default()
			r.Use(sdkInstance.GinMiddleware)

			r.Handle(http.MethodGet, tt.args.path, func(ctx *gin.Context) {
				if tt.args.devHint != "" {
					ctrl, _ := speakeasy.MiddlewareController(ctx.Request)
					require.NotNil(t, ctrl)
					ctrl.PathHint(tt.args.devHint)
				}
				ctx.Writer.WriteHeader(http.StatusOK)
				handled = true
			})

			w := httptest.NewRecorder()

			req, err := http.NewRequest(http.MethodGet, tt.args.url, nil)
			assert.NoError(t, err)

			r.ServeHTTP(w, req)

			wg.Wait()

			assert.True(t, handled, "middleware did not call handler")
			assert.True(t, captured, "middleware did not capture request")

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

func TestSpeakeasy_EchoMiddleware_Success(t *testing.T) {
	tests := loadTestData(t)
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			speakeasy.ExportSetMaxCaptureSize(tt.Fields.MaxCaptureSize)

			captured := false
			handled := false

			if tt.Args.RequestStartTime.IsZero() {
				speakeasy.ExportSetTimeNow(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))
			} else {
				speakeasy.ExportSetTimeNow(tt.Args.RequestStartTime)
			}
			if tt.Args.ElapsedTime == 0 {
				speakeasy.ExportSetTimeSince(1 * time.Millisecond)
			} else {
				speakeasy.ExportSetTimeSince(time.Duration(tt.Args.ElapsedTime) * time.Millisecond)
			}

			wg := &sync.WaitGroup{}
			wg.Add(1)

			sdkInstance := speakeasy.New(speakeasy.Config{
				APIKey:    testAPIKey,
				ApiID:     testApiID,
				VersionID: testVersionID,
				GRPCDialer: dialer(func(ctx context.Context, req *ingest.IngestRequest) {
					assert.JSONEq(t, tt.WantHAR, req.Har)
					captured = true
					wg.Done()
				}),
			})

			r := echo.New()
			r.Use(sdkInstance.EchoMiddleware)

			r.Any("/*", func(c echo.Context) error {
				ctrl, _ := speakeasy.MiddlewareController(c.Request())

				if tt.Args.QueryStringMasks != nil {
					for k, v := range tt.Args.QueryStringMasks {
						ctrl.Masking(speakeasy.WithQueryStringMask([]string{k}, v))
					}
				}

				if tt.Args.RequestHeaderMasks != nil {
					for k, v := range tt.Args.RequestHeaderMasks {
						ctrl.Masking(speakeasy.WithRequestHeaderMask([]string{k}, v))
					}
				}

				if tt.Args.RequestCookieMasks != nil {
					for k, v := range tt.Args.RequestCookieMasks {
						ctrl.Masking(speakeasy.WithRequestCookieMask([]string{k}, v))
					}
				}

				if tt.Args.RequestFieldMasksString != nil {
					for k, v := range tt.Args.RequestFieldMasksString {
						ctrl.Masking(speakeasy.WithRequestFieldMaskString([]string{k}, v))
					}
				}

				if tt.Args.RequestFieldMasksNumber != nil {
					for k, v := range tt.Args.RequestFieldMasksNumber {
						ctrl.Masking(speakeasy.WithRequestFieldMaskNumber([]string{k}, v))
					}
				}

				if tt.Args.ResponseHeaderMasks != nil {
					for k, v := range tt.Args.ResponseHeaderMasks {
						ctrl.Masking(speakeasy.WithResponseHeaderMask([]string{k}, v))
					}
				}

				if tt.Args.ResponseCookieMasks != nil {
					for k, v := range tt.Args.ResponseCookieMasks {
						ctrl.Masking(speakeasy.WithResponseCookieMask([]string{k}, v))
					}
				}

				if tt.Args.ResponseFieldMasksString != nil {
					for k, v := range tt.Args.ResponseFieldMasksString {
						ctrl.Masking(speakeasy.WithResponseFieldMaskString([]string{k}, v))
					}
				}

				if tt.Args.ResponseFieldMasksNumber != nil {
					for k, v := range tt.Args.ResponseFieldMasksNumber {
						ctrl.Masking(speakeasy.WithResponseFieldMaskNumber([]string{k}, v))
					}
				}

				for _, header := range tt.Args.ResponseHeaders {
					for _, val := range header.Values {
						c.Response().Header().Add(header.Key, val)
					}
				}

				if c.Request().Body != nil {
					data, err := io.ReadAll(c.Request().Body)
					assert.NoError(t, err)
					assert.Equal(t, tt.Args.Body, string(data))
				}

				if tt.Args.ResponseStatus > 0 {
					c.Response().WriteHeader(tt.Args.ResponseStatus)
				}

				if tt.Args.ResponseBody != "" {
					_, err := c.Response().Write([]byte(tt.Args.ResponseBody))
					assert.NoError(t, err)
				}
				handled = true

				return nil
			})

			w := httptest.NewRecorder()

			var req *http.Request
			var err error
			if tt.Args.Body == "" {
				req, err = http.NewRequest(tt.Args.Method, tt.Args.URL, nil)
			} else {
				req, err = http.NewRequest(tt.Args.Method, tt.Args.URL, bytes.NewBuffer([]byte(tt.Args.Body)))
			}
			assert.NoError(t, err)

			for _, header := range tt.Args.Headers {
				for _, val := range header.Values {
					req.Header.Add(header.Key, val)
				}
			}

			r.ServeHTTP(w, req)

			wg.Wait()

			assert.True(t, handled, "middleware did not call handler")
			assert.True(t, captured, "middleware did not capture request")

			responseStatus := http.StatusOK
			if tt.Args.ResponseStatus > 0 {
				responseStatus = tt.Args.ResponseStatus
			}

			assert.Equal(t, responseStatus, w.Code)
		})
	}
}

func TestSpeakeasy_EchoMiddleware_PathHint_Success(t *testing.T) {
	type args struct {
		path    string
		url     string
		devHint string
	}
	tests := []struct {
		name         string
		args         args
		wantPathHint string
	}{
		{
			name: "captures simple path hint from echo",
			args: args{
				path: "/user",
				url:  "http://test.com/user",
			},
			wantPathHint: "/user",
		},
		{
			name: "captures more complex path hint from echo",
			args: args{
				path: "/user/:id/*",
				url:  "http://test.com/user/1/send",
			},
			wantPathHint: "/user/{id}/{wildcard}",
		},
		{
			name: "path hint is overridden by dev hint",
			args: args{
				path:    "/user/:id/*action",
				url:     "http://test.com/user/1/sent",
				devHint: "/user/{id}/{action}",
			},
			wantPathHint: "/user/{id}/{action}",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			captured := false
			handled := false

			speakeasy.ExportSetTimeNow(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))
			speakeasy.ExportSetTimeSince(1 * time.Millisecond)

			wg := &sync.WaitGroup{}
			wg.Add(1)

			sdkInstance := speakeasy.New(speakeasy.Config{
				APIKey:    testAPIKey,
				ApiID:     testApiID,
				VersionID: testVersionID,
				GRPCDialer: dialer(func(ctx context.Context, req *ingest.IngestRequest) {
					assert.Equal(t, tt.wantPathHint, req.PathHint)
					captured = true
					wg.Done()
				}),
			})

			r := echo.New()
			r.Use(sdkInstance.EchoMiddleware)
			r.Match([]string{http.MethodGet}, tt.args.path, func(c echo.Context) error {
				if tt.args.devHint != "" {
					ctrl, _ := speakeasy.MiddlewareController(c.Request())
					require.NotNil(t, ctrl)
					ctrl.PathHint(tt.args.devHint)
				}
				c.Response().Writer.WriteHeader(http.StatusOK)
				handled = true

				return nil
			})

			w := httptest.NewRecorder()

			req, err := http.NewRequest(http.MethodGet, tt.args.url, nil)
			assert.NoError(t, err)

			r.ServeHTTP(w, req)

			wg.Wait()

			assert.True(t, handled, "middleware did not call handler")
			assert.True(t, captured, "middleware did not capture request")

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

func TestSpeakeasy_Middleware_Capture_CustomerID_Success(t *testing.T) {
	type args struct {
		url        string
		customerID string
	}
	tests := []struct {
		name           string
		args           args
		wantCustomerID string
	}{
		{
			name: "captures simple path hint from mux",
			args: args{
				url:        "http://test.com/user/1",
				customerID: "a-customers-id",
			},
			wantCustomerID: "a-customers-id",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			captured := false
			handled := false

			speakeasy.ExportSetTimeNow(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))
			speakeasy.ExportSetTimeSince(1 * time.Millisecond)

			wg := &sync.WaitGroup{}
			wg.Add(1)

			sdkInstance := speakeasy.New(speakeasy.Config{
				APIKey:    testAPIKey,
				ApiID:     testApiID,
				VersionID: testVersionID,
				GRPCDialer: dialer(func(ctx context.Context, req *ingest.IngestRequest) {
					assert.Equal(t, tt.wantCustomerID, req.CustomerId)
					captured = true
					wg.Done()
				}),
			})

			w := httptest.NewRecorder()

			req, err := http.NewRequest(http.MethodGet, tt.args.url, nil)
			assert.NoError(t, err)

			sdkInstance.Middleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				ctrl, _ := speakeasy.MiddlewareController(req)
				require.NotNil(t, ctrl)
				ctrl.CustomerID(tt.args.customerID)

				w.WriteHeader(http.StatusOK)
				handled = true
			})).ServeHTTP(w, req)

			wg.Wait()

			assert.True(t, handled, "middleware did not call handler")
			assert.True(t, captured, "middleware did not capture request")
			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

func TestSpeakeasy_Middleware_DataDogHttpTraceServerMux_PathHint_Success(t *testing.T) {
	type args struct {
		path string
		url  string
	}
	tests := []struct {
		name         string
		args         args
		wantPathHint string
	}{
		{
			name: "captures simple path hint from DefaultServerMux",
			args: args{
				path: "/user",
				url:  "http://test.com/user",
			},
			wantPathHint: "/user",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			captured := false
			handled := false

			speakeasy.ExportSetTimeNow(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))
			speakeasy.ExportSetTimeSince(1 * time.Millisecond)

			wg := &sync.WaitGroup{}
			wg.Add(1)

			sdkInstance := speakeasy.New(speakeasy.Config{
				APIKey:    testAPIKey,
				ApiID:     testApiID,
				VersionID: testVersionID,
				GRPCDialer: dialer(func(ctx context.Context, req *ingest.IngestRequest) {
					assert.Equal(t, tt.wantPathHint, req.PathHint)
					captured = true
					wg.Done()
				}),
			})

			r := httptrace.NewServeMux()

			r.Handle(tt.args.path, sdkInstance.MiddlewareWithMux(r, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.WriteHeader(http.StatusOK)
				handled = true
			})))

			w := httptest.NewRecorder()

			req, err := http.NewRequest(http.MethodGet, tt.args.url, nil)
			assert.NoError(t, err)

			r.ServeHTTP(w, req)

			wg.Wait()

			assert.True(t, handled, "middleware did not call handler")
			assert.True(t, captured, "middleware did not capture request")
			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

func dialer(handlerFunc func(ctx context.Context, req *ingest.IngestRequest)) func() func(context.Context, string) (net.Conn, error) {
	return func() func(context.Context, string) (net.Conn, error) {
		listener := bufconn.Listen(1024 * 1024)

		server := grpc.NewServer()

		ingest.RegisterIngestServiceServer(server, &mockIngestServer{
			handlerFunc: handlerFunc,
		})

		go func() {
			if err := server.Serve(listener); err != nil {
				log.Fatal(err)
			}
		}()

		return func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}
	}
}

type mockIngestServer struct {
	ingest.UnimplementedIngestServiceServer
	handlerFunc func(ctx context.Context, req *ingest.IngestRequest)
}

func (m *mockIngestServer) Ingest(ctx context.Context, req *ingest.IngestRequest) (*ingest.IngestResponse, error) {
	m.handlerFunc(ctx, req)

	return &ingest.IngestResponse{}, nil
}
