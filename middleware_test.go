package speakeasy

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi"
	"github.com/stretchr/testify/suite"
)

type TestSuite struct {
	suite.Suite

	testServer *httptest.Server
	router     *chi.Mux

	speakeasyMockMux    *http.ServeMux
	speakeasyMockServer *httptest.Server

	speakeasyApp *SpeakeasyApp
}

func Test(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

func (s *TestSuite) SetupSubTest() {
	s.router = chi.NewRouter()
	s.testServer = httptest.NewServer(s.router)
	s.speakeasyMockMux = http.NewServeMux()
	s.speakeasyMockServer = httptest.NewServer(s.speakeasyMockMux)
	var err error
	s.speakeasyApp, err = Configure(Configuration{ServerURL: s.speakeasyMockServer.URL, APIKey: "key", SchemaFilePath: "./test_fixtures/valid_openapi_schema.yml"})
	s.Require().NoError(err)
}

func (s *TestSuite) TearDownSubTest() {
	s.testServer.Close()
	s.speakeasyMockServer.Close()
}

func (s *TestSuite) Test_JsonFormat() {
	content, err := ioutil.ReadFile("sample.json")
	s.Require().NoError(err)
	var speakeasyMetadata MetaData
	err = json.Unmarshal(content, &speakeasyMetadata)
	s.Require().NoError(err)

}

func (s *TestSuite) Test_Middleware() {
	type args struct {
		requestJson, responseJson, requestHeaderKey, requestHeaderValue, respHeaderKey, respHeaderValue string
		status                                                                                          int
	}

	tests := []struct {
		name                string
		args                args
		wantSpeakeasyCalled bool
		wantApiData         *ApiData
	}{
		{
			name: "happy-path",
			args: args{
				requestJson:        `{"id":2}`,
				responseJson:       `{"id":2, "name":"test"}`,
				status:             http.StatusOK,
				requestHeaderKey:   "Req-K-200",
				requestHeaderValue: "Req-V-200",
				respHeaderKey:      "Resp-K-200",
				respHeaderValue:    "Resp-V-200",
			},
			wantSpeakeasyCalled: true,
			wantApiData: &ApiData{
				ApiKey:      "key",
				ApiServerId: "3453",
				Handlers:    []HandlerInfo{{Path: "/test", ApiStats: ApiStats{NumCalls: 0, NumErrors: 0, NumUniqueCustomers: 0}}},
			},
		},
		{
			name: "status-nok",
			args: args{
				requestJson:        `{"id":3}`,
				responseJson:       `{"id":2, "name":"test", "errors":true}`,
				status:             http.StatusConflict,
				requestHeaderKey:   "Req-K-409",
				requestHeaderValue: "Req-V-409",
				respHeaderKey:      "Resp-K-409",
				respHeaderValue:    "Resp-V-409",
			},
			wantSpeakeasyCalled: true,
			wantApiData: &ApiData{
				ApiKey:      "key",
				ApiServerId: "3453",
				Handlers:    []HandlerInfo{{Path: "/test", ApiStats: ApiStats{NumCalls: 0, NumErrors: 0, NumUniqueCustomers: 0}}},
			},
		},
		{
			name: "req-not-json",
			args: args{
				requestJson:  `{"id4`,
				responseJson: `{}`,
				status:       http.StatusOK,
			},
			wantSpeakeasyCalled: false,
		},
		{
			name: "resp-not-json",
			args: args{
				requestJson:  `{"id":5}`,
				responseJson: `{"`,
				status:       http.StatusOK,
			},
			wantSpeakeasyCalled: true,
		},
	}

	for _, tt := range tests {
		s.SetupSubTest()
		speakeasyCalled := false

		s.speakeasyMockMux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
			var speakeasyApiData ApiData
			decoder := json.NewDecoder(r.Body)
			s.Require().NoError(decoder.Decode(&speakeasyApiData))
			s.Require().Equal(speakeasyApiData, tt.wantApiData)
			speakeasyCalled = true
		})

		s.router.With(s.speakeasyApp.Middleware).Get("/test", func(w http.ResponseWriter, r *http.Request) {
			s.Require().Equal(tt.args.requestHeaderValue, r.Header.Get(tt.args.requestHeaderKey))
			w.Header()[tt.args.respHeaderKey] = []string{tt.args.respHeaderValue}
			w.WriteHeader(tt.args.status)
			_, err := w.Write([]byte(tt.args.responseJson))
			if err != nil {
				return
			}
		})

		requestHeaders := map[string]string{}
		if len(tt.args.requestHeaderKey) > 0 {
			requestHeaders[tt.args.requestHeaderKey] = tt.args.requestHeaderValue
		}
		resp, respBody := s.testRequest(http.MethodGet, "/test", tt.args.requestJson, requestHeaders)
		s.Require().Equal(tt.args.responseJson, respBody, tt.name)
		s.Require().Equal(tt.args.status, resp.StatusCode, tt.name)
		s.Require().Equal(tt.args.respHeaderValue, resp.Header.Get(tt.args.respHeaderKey), tt.name)

		// wait on the async speakeasy call to finish
		time.Sleep(1 * time.Second)

		s.Require().Equal(tt.wantSpeakeasyCalled, speakeasyCalled, tt.name)
		s.TearDownSubTest()
	}
}

func (s *TestSuite) testRequest(method, path, body string, headers map[string]string) (*http.Response, string) {
	req, err := http.NewRequest(method, s.testServer.URL+path, bytes.NewBuffer([]byte(body)))
	s.Require().NoError(err)

	for k, v := range headers {
		req.Header.Add(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	s.Require().NoError(err)

	respBody, err := ioutil.ReadAll(resp.Body)
	s.Require().NoError(err)
	defer resp.Body.Close()

	return resp, string(respBody)
}
