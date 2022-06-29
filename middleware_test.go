package speakeasy

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi"
	"github.com/speakeasy-api/speakeasy-go-sdk/internal/models"
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

func (s *TestSuite) SetupSubTest(wantConfErr error, schemaPath string) error {
	s.router = chi.NewRouter()
	s.testServer = httptest.NewServer(s.router)
	s.speakeasyMockMux = http.NewServeMux()
	s.speakeasyMockServer = httptest.NewServer(s.speakeasyMockMux)
	var err error
	s.speakeasyApp, err = Configure(Configuration{ServerURL: s.speakeasyMockServer.URL, APIKey: "key", SchemaFilePath: schemaPath, ApiStatsIntervalSeconds: 1})
	if wantConfErr != nil {
		s.Require().ErrorContains(err, wantConfErr.Error())
		return err
	} else {
		s.Require().NoError(err)
	}
	return nil
}

func (s *TestSuite) TearDownSubTest() {
	s.testServer.Close()
	s.speakeasyMockServer.Close()
	s.speakeasyApp.CancelApiStats()
	// wait on the goroutine sending speakeasy stats to terminate
	time.Sleep(1 * time.Second)
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
		requestJson, responseJson, requestHeaderKey, requestHeaderValue, apiPath, respHeaderKey, respHeaderValue, schemaPath string
		status, numRequests                                                                                                  int
	}

	tests := []struct {
		name         string
		args         args
		wantApiStats *ApiStats
		wantApi      *models.Api
		wantSchema   *models.Schema
		wantConfErr  error
	}{
		{
			name: "happy-path",
			args: args{
				numRequests:        5,
				requestJson:        `{"id":2}`,
				responseJson:       `{"id":2, "name":"test"}`,
				status:             http.StatusOK,
				requestHeaderKey:   "Req-K-200",
				requestHeaderValue: "Req-V-200",
				apiPath:            "/test",
				respHeaderKey:      "Resp-K-200",
				respHeaderValue:    "Resp-V-200",
				schemaPath:         "./test_fixtures/valid_openapi_schema.yml",
			},
			wantApiStats: &ApiStats{NumCalls: 5, NumErrors: 0, NumUniqueCustomers: 0},
			wantSchema:   &models.Schema{VersionId: "1.0.0", Filename: "valid_openapi_schema.yml"},
			wantApi:      &models.Api{Method: "GET", Path: "/test", DisplayName: "testRequestsv1", Description: "Test API Requests"},
		},
		{
			name: "status-nok",
			args: args{
				numRequests:        5,
				requestJson:        `{"id":3}`,
				responseJson:       `{"id":2, "name":"test", "errors":true}`,
				status:             http.StatusConflict,
				requestHeaderKey:   "Req-K-409",
				requestHeaderValue: "Req-V-409",
				apiPath:            "/test",
				respHeaderKey:      "Resp-K-409",
				respHeaderValue:    "Resp-V-409",
				schemaPath:         "./test_fixtures/valid_openapi_schema.yml",
			},
			wantApiStats: &ApiStats{NumCalls: 5, NumErrors: 5, NumUniqueCustomers: 0},
			wantSchema:   &models.Schema{VersionId: "1.0.0", Filename: "valid_openapi_schema.yml"},
			wantApi:      &models.Api{Method: "GET", Path: "/test", DisplayName: "testRequestsv1", Description: "Test API Requests"},
		},
		{
			name: "req-not-json",
			args: args{
				requestJson:  `{"id4`,
				apiPath:      "/test",
				responseJson: `{}`,
				status:       http.StatusOK,
				schemaPath:   "./test_fixtures/valid_openapi_schema.yml",
			},
			wantApiStats: &ApiStats{NumCalls: 0, NumErrors: 0, NumUniqueCustomers: 0},
			wantSchema:   &models.Schema{VersionId: "1.0.0", Filename: "valid_openapi_schema.yml"},
			wantApi:      &models.Api{Method: "GET", Path: "/test", DisplayName: "testRequestsv1", Description: "Test API Requests"},
		},
		{
			name: "valid-schema-wrong-path",
			args: args{
				numRequests:        5,
				requestJson:        `{"id":2}`,
				responseJson:       `{"id":2, "name":"test"}`,
				status:             http.StatusOK,
				requestHeaderKey:   "Req-K-200",
				requestHeaderValue: "Req-V-200",
				apiPath:            "/wrong",
				respHeaderKey:      "Resp-K-200",
				respHeaderValue:    "Resp-V-200",
				schemaPath:         "./test_fixtures/valid_openapi_schema_wrong_path.yml",
			},
			wantApiStats: &ApiStats{NumCalls: 0, NumErrors: 0, NumUniqueCustomers: 0},
			wantSchema:   &models.Schema{VersionId: "1.0.0", Filename: "valid_openapi_schema_wrong_path.yml"},
			wantApi:      &models.Api{Method: "GET", Path: "/wrong", DisplayName: "testRequestsv1", Description: "Test API Requests"},
		},
		{
			name: "invalid-schema",
			args: args{
				requestJson:        `{"id":2}`,
				responseJson:       `{"id":2, "name":"test"}`,
				status:             http.StatusOK,
				requestHeaderKey:   "Req-K-200",
				requestHeaderValue: "Req-V-200",
				apiPath:            "/test",
				respHeaderKey:      "Resp-K-200",
				respHeaderValue:    "Resp-V-200",
				schemaPath:         "./test_fixtures/invalid_openapi_schema.yml",
			},
			wantConfErr: errors.New("value of openapi must be a non-empty string"),
			wantSchema:  &models.Schema{VersionId: "1.0.0", Filename: "invalid_openapi_schema.yml"},
		},
	}

	for _, tt := range tests {
		err := s.SetupSubTest(tt.wantConfErr, tt.args.schemaPath)
		if err != nil {
			return
		}
		speakeasyCalled := false

		s.speakeasyMockMux.HandleFunc("/rs/v1/metrics", func(w http.ResponseWriter, r *http.Request) {
			var apiData ApiData
			decoder := json.NewDecoder(r.Body)
			s.Require().NoError(decoder.Decode(&apiData))
			if tt.wantApiStats != nil {
				s.Require().Equal(s.speakeasyApp.APIKey, apiData.ApiKey)
				apiId := s.speakeasyApp.ApiByPath[tt.args.apiPath].ID
				s.Require().Equal(tt.wantApiStats, apiData.Handlers.ApiStatsById[apiId])
			}
			speakeasyCalled = true
		})

		registerCalled := false

		s.speakeasyMockMux.HandleFunc("/rs/v1/apis/", func(w http.ResponseWriter, r *http.Request) {
			var api models.Api
			var schema models.Schema
			var decoder *json.Decoder
			if !strings.Contains(r.RequestURI, "schemas") {
				decoder = json.NewDecoder(r.Body)
				s.Require().NoError(decoder.Decode(&api))
				fmt.Print("done")
				if tt.wantApi != nil {
					s.Require().True(testApiEqual(*tt.wantApi, api))
				}
			} else {
				s.Require().NoError(r.ParseMultipartForm(32 << 20))
				schemaJSON := r.FormValue("schema")
				s.Require().NotEmpty(schemaJSON)
				json.Unmarshal([]byte(schemaJSON), &schema)
				s.Require().NotEmpty(schema)
				if tt.wantSchema != nil {
					s.Require().True(testSchemaEqual(*tt.wantSchema, schema))
				}
			}
			registerCalled = true
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
		var resp *http.Response
		var respBody string
		for i := 1; i <= tt.args.numRequests; i++ {
			resp, respBody = s.testRequest(http.MethodGet, "/test", tt.args.requestJson, requestHeaders)
			s.Require().Equal(tt.args.responseJson, respBody, tt.name)
			s.Require().Equal(tt.args.status, resp.StatusCode, tt.name)
			s.Require().Equal(tt.args.respHeaderValue, resp.Header.Get(tt.args.respHeaderKey), tt.name)
		}

		// wait on the async speakeasy call to finish
		time.Sleep(3 * time.Second)

		s.Require().Equal(true, speakeasyCalled, tt.name)
		s.Require().Equal(true, registerCalled, tt.name)
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

func testApiEqual(a1, a2 models.Api) bool {
	return a1.Method == a2.Method && a1.Path == a2.Path && a1.DisplayName == a2.DisplayName && a1.Description == a2.Description
}

func testSchemaEqual(s1, s2 models.Schema) bool {
	return s1.VersionId == s2.VersionId && s1.Filename == s2.Filename
}
