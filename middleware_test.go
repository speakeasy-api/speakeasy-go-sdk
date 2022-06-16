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
}

func Test(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

func (s *TestSuite) SetupSubTest() {
	s.router = chi.NewRouter()
	s.testServer = httptest.NewServer(s.router)
	s.speakeasyMockMux = http.NewServeMux()
	s.speakeasyMockServer = httptest.NewServer(s.speakeasyMockMux)
	Configure(Configuration{ServerURL: s.speakeasyMockServer.URL, APIKey: "key", WorkspaceId: "workspace_id"})
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
	testCases := map[string]struct {
		requestJson        string
		responseJson       string
		status             int
		requestHeaderKey   string
		requestHeaderValue string
		respHeaderKey      string
		respHeaderValue    string
		speakeasyCalled    bool
	}{
		"happy-path": {

			requestJson:        `{"id":2}`,
			responseJson:       `{"id":2, "name":"test"}`,
			status:             http.StatusOK,
			requestHeaderKey:   "Req-K-200",
			requestHeaderValue: "Req-V-200",
			respHeaderKey:      "Resp-K-200",
			respHeaderValue:    "Resp-V-200",
			speakeasyCalled:    true,
		},
		"status-nok": {
			requestJson:        `{"id":3}`,
			responseJson:       `{"id":2, "name":"test", "errors":true}`,
			status:             http.StatusConflict,
			requestHeaderKey:   "Req-K-409",
			requestHeaderValue: "Req-V-409",
			respHeaderKey:      "Resp-K-409",
			respHeaderValue:    "Resp-V-409",
			speakeasyCalled:    true,
		},
		"req-not-json": {
			requestJson:     `{"id4`,
			responseJson:    `{}`,
			status:          http.StatusOK,
			speakeasyCalled: false,
		},
		"resp-not-json": {
			requestJson:     `{"id":5}`,
			responseJson:    `{"`,
			status:          http.StatusOK,
			speakeasyCalled: true,
		},
	}

	for tn, tc := range testCases {
		s.SetupSubTest()
		speakeasyCalled := false

		s.speakeasyMockMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			var speakeasyMetadata MetaData
			decoder := json.NewDecoder(r.Body)
			s.Require().NoError(decoder.Decode(&speakeasyMetadata))
			s.Require().Equal(getOsInfo(), speakeasyMetadata.Data.Server.Os)
			speakeasyCalled = true
		})

		s.router.With(Middleware).Get("/test", func(w http.ResponseWriter, r *http.Request) {
			s.Require().Equal(tc.requestHeaderValue, r.Header.Get(tc.requestHeaderKey))
			w.Header()[tc.respHeaderKey] = []string{tc.respHeaderValue}
			w.WriteHeader(tc.status)
			_, err := w.Write([]byte(tc.responseJson))
			if err != nil {
				return
			}
		})

		requestHeaders := map[string]string{}
		if len(tc.requestHeaderKey) > 0 {
			requestHeaders[tc.requestHeaderKey] = tc.requestHeaderValue
		}
		resp, respBody := s.testRequest(http.MethodGet, "/test", tc.requestJson, requestHeaders)
		s.Require().Equal(tc.responseJson, respBody, tn)
		s.Require().Equal(tc.status, resp.StatusCode, tn)
		s.Require().Equal(tc.respHeaderValue, resp.Header.Get(tc.respHeaderKey), tn)

		// wait on the async speakeasy call to finish
		time.Sleep(1 * time.Second)

		s.Require().Equal(tc.speakeasyCalled, speakeasyMuxCalled, tn)
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
