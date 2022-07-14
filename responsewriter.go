package speakeasy

import (
	"bytes"
	"net/http"

	"github.com/speakeasy-api/speakeasy-go-sdk/internal/log"
	"go.uber.org/zap"
)

type speakeasyResponseWriter struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
	valid  bool
}

var _ http.ResponseWriter = &speakeasyResponseWriter{}

func newResponseWriter(w http.ResponseWriter) *speakeasyResponseWriter {
	return &speakeasyResponseWriter{
		ResponseWriter: w,
		status:         http.StatusOK,
		valid:          true,
	}
}

func (r *speakeasyResponseWriter) Write(data []byte) (int, error) {
	if _, err := r.body.Write(data); err != nil {
		log.Logger().Error("failed to record response body", zap.Error(err))
		r.valid = false
	}

	return r.ResponseWriter.Write(data)
}

func (r *speakeasyResponseWriter) WriteHeader(statusCode int) {
	r.status = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}
