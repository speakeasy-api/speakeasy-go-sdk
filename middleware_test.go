package speakeasy_test

import (
	"bytes"
	"context"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/speakeasy-api/speakeasy-go-sdk"
	"github.com/speakeasy-api/speakeasy-schemas/grpc/go/registry/ingest"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

func TestMiddleware_Success(t *testing.T) {
	type args struct {
		method          string
		url             string
		headers         map[string][]string
		body            []byte
		responseStatus  int
		responseBody    []byte
		responseHeaders map[string][]string
	}
	tests := []struct {
		name    string
		args    args
		wantHAR string
	}{
		{
			name: "captures basic request and response",
			args: args{
				method:         http.MethodGet,
				url:            "http://test.com/test",
				responseStatus: http.StatusOK,
				responseBody:   []byte("test"),
			},
			wantHAR: `{"log":{"version":"1.2","creator":{"name":"speakeasy-go-sdk","version":"0.0.1"},"entries":[{"startedDateTime":"2020-01-01T00:00:00Z","time":1,"request":{"method":"GET","url":"http://test.com/test","httpVersion":"HTTP/1.1","cookies":[],"headers":[],"queryString":[],"postData":{"mimeType":"application/octet-stream","params":null,"text":""},"headersSize":-1,"bodySize":0},"response":{"status":200,"statusText":"OK","httpVersion":"HTTP/1.1","cookies":[],"headers":[],"content":{"size":4,"mimeType":"application/octet-stream","text":"test"},"redirectURL":"","headersSize":-1,"bodySize":4},"cache":null,"timings":null,"serverIPAddress":"test.com"}],"comment":"request capture for http://test.com/test"}}`,
		},
		{
			name: "captures basic request and no response body",
			args: args{
				method: http.MethodGet,
				url:    "http://test.com/test",
			},
			wantHAR: `{"log":{"version":"1.2","creator":{"name":"speakeasy-go-sdk","version":"0.0.1"},"entries":[{"startedDateTime":"2020-01-01T00:00:00Z","time":1,"request":{"method":"GET","url":"http://test.com/test","httpVersion":"HTTP/1.1","cookies":[],"headers":[],"queryString":[],"postData":{"mimeType":"application/octet-stream","params":null,"text":""},"headersSize":-1,"bodySize":0},"response":{"status":200,"statusText":"OK","httpVersion":"HTTP/1.1","cookies":[],"headers":[],"content":{"size":0,"mimeType":"application/octet-stream"},"redirectURL":"","headersSize":-1,"bodySize":0},"cache":null,"timings":null,"serverIPAddress":"test.com"}],"comment":"request capture for http://test.com/test"}}`,
		},
		{
			name: "captures basic request and response with no response header set",
			args: args{
				method:         http.MethodGet,
				url:            "http://test.com/test",
				responseStatus: -1,
				responseBody:   []byte("test"),
			},
			wantHAR: `{"log":{"version":"1.2","creator":{"name":"speakeasy-go-sdk","version":"0.0.1"},"entries":[{"startedDateTime":"2020-01-01T00:00:00Z","time":1,"request":{"method":"GET","url":"http://test.com/test","httpVersion":"HTTP/1.1","cookies":[],"headers":[],"queryString":[],"postData":{"mimeType":"application/octet-stream","params":null,"text":""},"headersSize":-1,"bodySize":0},"response":{"status":200,"statusText":"OK","httpVersion":"HTTP/1.1","cookies":[],"headers":[{"name":"Content-Type","value":"text/plain; charset=utf-8"}],"content":{"size":4,"mimeType":"text/plain; charset=utf-8","text":"test"},"redirectURL":"","headersSize":-1,"bodySize":4},"cache":null,"timings":null,"serverIPAddress":"test.com"}],"comment":"request capture for http://test.com/test"}}`,
		},
		{
			name: "captures basic request and response with different content types",
			args: args{
				method:          http.MethodGet,
				url:             "http://test.com/test",
				headers:         map[string][]string{"Content-Type": {"application/json"}},
				responseStatus:  -1,
				responseBody:    []byte("test"),
				responseHeaders: map[string][]string{"Content-Type": {"text/plain; charset=utf-8"}},
			},
			wantHAR: `{"log":{"version":"1.2","creator":{"name":"speakeasy-go-sdk","version":"0.0.1"},"entries":[{"startedDateTime":"2020-01-01T00:00:00Z","time":1,"request":{"method":"GET","url":"http://test.com/test","httpVersion":"HTTP/1.1","cookies":[],"headers":[{"name":"Content-Type","value":"application/json"}],"queryString":[],"postData":{"mimeType":"application/json","params":null,"text":""},"headersSize":-1,"bodySize":0},"response":{"status":200,"statusText":"OK","httpVersion":"HTTP/1.1","cookies":[],"headers":[{"name":"Content-Type","value":"text/plain; charset=utf-8"}],"content":{"size":4,"mimeType":"text/plain; charset=utf-8","text":"test"},"redirectURL":"","headersSize":-1,"bodySize":4},"cache":null,"timings":null,"serverIPAddress":"test.com"}],"comment":"request capture for http://test.com/test"}}`,
		},
		{
			name: "captures post request with body",
			args: args{
				method:          http.MethodPost,
				url:             "http://test.com/test",
				headers:         map[string][]string{"Content-Type": {"application/json"}},
				body:            []byte(`{test: "test"}`),
				responseStatus:  -1,
				responseBody:    []byte("test"),
				responseHeaders: map[string][]string{"Content-Type": {"text/plain; charset=utf-8"}},
			},
			wantHAR: `{"log":{"version":"1.2","creator":{"name":"speakeasy-go-sdk","version":"0.0.1"},"entries":[{"startedDateTime":"2020-01-01T00:00:00Z","time":1,"request":{"method":"POST","url":"http://test.com/test","httpVersion":"HTTP/1.1","cookies":[],"headers":[{"name":"Content-Type","value":"application/json"}],"queryString":[],"postData":{"mimeType":"application/json","params":null,"text":"{test: \"test\"}"},"headersSize":-1,"bodySize":14},"response":{"status":200,"statusText":"OK","httpVersion":"HTTP/1.1","cookies":[],"headers":[{"name":"Content-Type","value":"text/plain; charset=utf-8"}],"content":{"size":4,"mimeType":"text/plain; charset=utf-8","text":"test"},"redirectURL":"","headersSize":-1,"bodySize":4},"cache":null,"timings":null,"serverIPAddress":"test.com"}],"comment":"request capture for http://test.com/test"}}`,
		},
		{
			name: "captures query params",
			args: args{
				method:         http.MethodGet,
				url:            "http://test.com/test?param1=value1",
				responseStatus: http.StatusOK,
				responseBody:   []byte("test"),
			},
			wantHAR: `{"log":{"version":"1.2","creator":{"name":"speakeasy-go-sdk","version":"0.0.1"},"entries":[{"startedDateTime":"2020-01-01T00:00:00Z","time":1,"request":{"method":"GET","url":"http://test.com/test?param1=value1","httpVersion":"HTTP/1.1","cookies":[],"headers":[],"queryString":[{"name":"param1","value":"value1"}],"postData":{"mimeType":"application/octet-stream","params":null,"text":""},"headersSize":-1,"bodySize":0},"response":{"status":200,"statusText":"OK","httpVersion":"HTTP/1.1","cookies":[],"headers":[],"content":{"size":4,"mimeType":"application/octet-stream","text":"test"},"redirectURL":"","headersSize":-1,"bodySize":4},"cache":null,"timings":null,"serverIPAddress":"test.com"}],"comment":"request capture for http://test.com/test?param1=value1"}}`,
		},
		{
			name: "captures cookies",
			args: args{
				method:          http.MethodGet,
				url:             "http://test.com/test",
				headers:         map[string][]string{"Cookie": {"cookie1=value1; cookie2=value2"}},
				responseStatus:  http.StatusOK,
				responseBody:    []byte("test"),
				responseHeaders: map[string][]string{"Set-Cookie": {"cookie1=value1; cookie2=value2"}},
			},
			wantHAR: `{"log":{"version":"1.2","creator":{"name":"speakeasy-go-sdk","version":"0.0.1"},"entries":[{"startedDateTime":"2020-01-01T00:00:00Z","time":1,"request":{"method":"GET","url":"http://test.com/test","httpVersion":"HTTP/1.1","cookies":[{"name":"cookie1","value":"value1","expires":"0001-01-01T00:00:00Z"},{"name":"cookie2","value":"value2","expires":"0001-01-01T00:00:00Z"}],"headers":[],"queryString":[],"postData":{"mimeType":"application/octet-stream","params":null,"text":""},"headersSize":-1,"bodySize":0},"response":{"status":200,"statusText":"OK","httpVersion":"HTTP/1.1","cookies":[{"name":"cookie1","value":"value1","expires":"0001-01-01T00:00:00Z"},{"name":"cookie2","value":"value2","expires":"0001-01-01T00:00:00Z"}],"headers":[],"content":{"size":4,"mimeType":"application/octet-stream","text":"test"},"redirectURL":"","headersSize":-1,"bodySize":4},"cache":null,"timings":null,"serverIPAddress":"test.com"}],"comment":"request capture for http://test.com/test"}}`,
		},
		{
			name: "captures redirect",
			args: args{
				method:          http.MethodGet,
				url:             "http://test.com/test",
				responseStatus:  http.StatusOK,
				responseBody:    []byte("test"),
				responseHeaders: map[string][]string{"Location": {"http://test.com/test2"}},
			},
			wantHAR: `{"log":{"version":"1.2","creator":{"name":"speakeasy-go-sdk","version":"0.0.1"},"entries":[{"startedDateTime":"2020-01-01T00:00:00Z","time":1,"request":{"method":"GET","url":"http://test.com/test","httpVersion":"HTTP/1.1","cookies":[],"headers":[],"queryString":[],"postData":{"mimeType":"application/octet-stream","params":null,"text":""},"headersSize":-1,"bodySize":0},"response":{"status":200,"statusText":"OK","httpVersion":"HTTP/1.1","cookies":[],"headers":[{"name":"Location","value":"http://test.com/test2"}],"content":{"size":4,"mimeType":"application/octet-stream","text":"test"},"redirectURL":"http://test.com/test2","headersSize":-1,"bodySize":4},"cache":null,"timings":null,"serverIPAddress":"test.com"}],"comment":"request capture for http://test.com/test"}}`,
		},
		{
			name: "captures body size zero when cached",
			args: args{
				method:         http.MethodGet,
				url:            "http://test.com/test",
				responseStatus: http.StatusNotModified,
				responseBody:   []byte("test"),
			},
			wantHAR: `{"log":{"version":"1.2","creator":{"name":"speakeasy-go-sdk","version":"0.0.1"},"entries":[{"startedDateTime":"2020-01-01T00:00:00Z","time":1,"request":{"method":"GET","url":"http://test.com/test","httpVersion":"HTTP/1.1","cookies":[],"headers":[],"queryString":[],"postData":{"mimeType":"application/octet-stream","params":null,"text":""},"headersSize":-1,"bodySize":0},"response":{"status":304,"statusText":"Not Modified","httpVersion":"HTTP/1.1","cookies":[],"headers":[],"content":{"size":4,"mimeType":"application/octet-stream","text":"test"},"redirectURL":"","headersSize":-1,"bodySize":0},"cache":null,"timings":null,"serverIPAddress":"test.com"}],"comment":"request capture for http://test.com/test"}}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("SPEAKEASY_SERVER_SECURE", "false")

			captured := false
			handled := false

			speakeasy.ExportSetTimeNow(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))
			speakeasy.ExportSetTimeSince(1 * time.Second)

			wg := &sync.WaitGroup{}
			wg.Add(1)

			sdkInstance := speakeasy.New(speakeasy.Config{APIKey: "test", GRPCDialer: dialer(func(ctx context.Context, req *ingest.IngestRequest) {
				assert.Equal(t, tt.wantHAR, req.Har)
				captured = true
				wg.Done()
			})})

			h := sdkInstance.Middleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				for k, v := range tt.args.responseHeaders {
					for _, vv := range v {
						w.Header().Add(k, vv)
					}
				}

				if req.Body != nil {
					data, err := ioutil.ReadAll(req.Body)
					assert.NoError(t, err)
					assert.Equal(t, string(tt.args.body), string(data))
				}

				if tt.args.responseStatus > 0 {
					w.WriteHeader(tt.args.responseStatus)
				}

				if tt.args.responseBody != nil {
					_, err := w.Write(tt.args.responseBody)
					assert.NoError(t, err)
				}
				handled = true
			}))

			w := httptest.NewRecorder()

			var req *http.Request
			var err error
			if tt.args.body == nil {
				req, err = http.NewRequest(tt.args.method, tt.args.url, nil)
			} else {
				req, err = http.NewRequest(tt.args.method, tt.args.url, bytes.NewBuffer(tt.args.body))
			}
			assert.NoError(t, err)

			for k, v := range tt.args.headers {
				for _, vv := range v {
					req.Header.Add(k, vv)
				}
			}

			h.ServeHTTP(w, req)

			wg.Wait()

			assert.True(t, handled, "middleware did not call handler")
			assert.True(t, captured, "middleware did not capture request")

			responseStatus := http.StatusOK
			if tt.args.responseStatus > 0 {
				responseStatus = tt.args.responseStatus
			}

			assert.Equal(t, responseStatus, w.Code)
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
