package speakeasy

import (
	"bytes"
	"io"
	"net/http"
)

type captureWriter struct {
	reqW         *requestWriter
	origResW     http.ResponseWriter
	resW         *responseWriter
	reqBuf       *bytes.Buffer
	resBuf       *bytes.Buffer
	reqValid     bool
	resValid     bool
	status       int
	responseSize int
	maxBuffer    int
}

func NewCaptureWriter(origResW http.ResponseWriter, maxBuffer int) *captureWriter {
	cw := &captureWriter{
		origResW:     origResW,
		reqBuf:       bytes.NewBuffer([]byte{}),
		resBuf:       bytes.NewBuffer([]byte{}),
		reqValid:     true,
		resValid:     true,
		status:       http.StatusOK,
		responseSize: 0,
		maxBuffer:    maxBuffer,
	}
	cw.reqW = &requestWriter{
		cw: cw,
	}
	cw.resW = &responseWriter{
		cw: cw,
	}

	return cw
}

func (c *captureWriter) GetRequestWriter() *requestWriter {
	return c.reqW
}

func (c *captureWriter) GetResponseWriter() *responseWriter {
	return c.resW
}

func (c *captureWriter) IsReqValid() bool {
	return c.reqValid
}

func (c *captureWriter) IsResValid() bool {
	return c.resValid
}

func (c *captureWriter) GetReqBuffer() *bytes.Buffer {
	return c.reqBuf
}

func (c *captureWriter) GetResBuffer() *bytes.Buffer {
	return c.resBuf
}

func (c *captureWriter) GetStatus() int {
	return c.status
}

func (c *captureWriter) GetResponseSize() int {
	return c.responseSize
}

func (c *captureWriter) writeReq(p []byte) (int, error) {
	// Check if we have exceeded the buffer size and if so drop rest of request
	if (c.reqBuf.Len() + c.resBuf.Len() + len(p)) > c.maxBuffer {
		c.reqValid = false
	} else if c.reqValid {
		_, err := c.reqBuf.Write(p)
		if err != nil {
			c.reqValid = false
		}
	}

	return len(p), nil
}

func (c *captureWriter) writeRes(p []byte) (int, error) {
	// Check if we have exceeded the buffer size and if so drop rest of response
	if (c.reqBuf.Len() + c.resBuf.Len() + len(p)) > c.maxBuffer {
		c.resValid = false
	} else if c.resValid {
		_, err := c.resBuf.Write(p)
		if err != nil {
			c.resValid = false
		}
	}

	n, err := c.origResW.Write(p)
	if err != nil {
		c.resValid = false
	}

	c.responseSize += n

	return n, err
}

func (c *captureWriter) writeHeader(statusCode int) {
	c.status = statusCode
	c.origResW.WriteHeader(statusCode)
}

type requestWriter struct {
	cw *captureWriter
}

var _ io.Writer = &requestWriter{}

func (r *requestWriter) Write(p []byte) (int, error) {
	return r.cw.writeReq(p)
}

type responseWriter struct {
	cw *captureWriter
}

var _ http.ResponseWriter = &responseWriter{}

func (r *responseWriter) Write(data []byte) (int, error) {
	return r.cw.writeRes(data)
}

func (r *responseWriter) WriteHeader(statusCode int) {
	r.cw.writeHeader(statusCode)
}

func (r *responseWriter) Header() http.Header {
	return r.cw.origResW.Header()
}
