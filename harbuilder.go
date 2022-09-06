package speakeasy

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strconv"
	"time"

	"github.com/chromedp/cdproto/har"
	"github.com/gorilla/handlers"
	"github.com/speakeasy-api/speakeasy-go-sdk/internal/bodymasking"
	"github.com/speakeasy-api/speakeasy-go-sdk/internal/log"
	"go.uber.org/zap"
)

type harBuilder struct{}

func (h *harBuilder) buildHarFile(ctx context.Context, cw *captureWriter, r *http.Request, startTime time.Time, c *controller) *har.HAR {
	resolvedURL := getResolvedURL(r, c)

	return &har.HAR{
		Log: &har.Log{
			Version: "1.2",
			Creator: &har.Creator{
				Name:    sdkName,
				Version: speakeasyVersion,
			},
			Comment: "request capture for " + resolvedURL.String(),
			Entries: []*har.Entry{
				{
					StartedDateTime: startTime.Format(time.RFC3339Nano),
					Time:            float64(timeSince(startTime).Milliseconds()),
					Request:         h.getHarRequest(ctx, cw, r, c, resolvedURL),
					Response:        h.getHarResponse(ctx, cw, r, startTime, c),
					Connection:      resolvedURL.Port(),
					ServerIPAddress: resolvedURL.Hostname(),
					Cache:           &har.Cache{},
					Timings: &har.Timings{
						Send:    -1,
						Wait:    -1,
						Receive: -1,
					},
				},
			},
		},
	}
}

func getResolvedURL(r *http.Request, c *controller) *url.URL {
	var url *url.URL

	// Taking advantage of Gorilla's ProxyHeaders parsing to resolve Forwarded headers
	handlers.ProxyHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		url = r.URL
	})).ServeHTTP(nil, r)

	queryParams := url.Query()
	for key, values := range queryParams {
		queryParams.Del(key)

		for _, value := range values {
			mask, ok := c.queryStringMasks[key]
			if ok {
				value = mask
			}

			queryParams.Add(key, value)
		}
	}
	url.RawQuery = queryParams.Encode()

	if url.IsAbs() {
		return url
	}

	if url.Scheme == "" {
		if r.TLS != nil {
			url.Scheme = "https"
		} else {
			url.Scheme = "http"
		}
	}

	if url.Host == "" {
		url.Host = r.Host
	}

	return url
}

//nolint:funlen
func (h *harBuilder) getHarRequest(ctx context.Context, cw *captureWriter, r *http.Request, c *controller, url *url.URL) *har.Request {
	reqHeaders := []*har.NameValuePair{}
	for key, headers := range r.Header {
		for _, headerValue := range headers {
			mask, ok := c.requestHeaderMasks[key]
			if ok {
				headerValue = mask
			}

			reqHeaders = append(reqHeaders, &har.NameValuePair{Name: key, Value: headerValue})
		}
	}

	sort.SliceStable(reqHeaders, func(i, j int) bool {
		return reqHeaders[i].Name < reqHeaders[j].Name
	})

	reqQueryParams := []*har.NameValuePair{}

	for key, values := range r.URL.Query() {
		for _, value := range values {
			mask, ok := c.queryStringMasks[key]
			if ok {
				value = mask
			}

			reqQueryParams = append(reqQueryParams, &har.NameValuePair{Name: key, Value: value})
		}
	}

	sort.SliceStable(reqQueryParams, func(i, j int) bool {
		return reqQueryParams[i].Name < reqQueryParams[j].Name
	})

	reqCookies := getHarCookies(r.Cookies(), time.Time{}, c.requestCookieMasks)

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

	postData := getPostData(r, cw, c, ctx)

	var bodySize int64 = -1
	if postData != nil {
		bodySize = r.ContentLength
	}

	return &har.Request{
		Method:      r.Method,
		URL:         url.String(),
		Headers:     reqHeaders,
		QueryString: reqQueryParams,
		BodySize:    bodySize,
		PostData:    postData,
		HTTPVersion: r.Proto,
		Cookies:     reqCookies,
		HeadersSize: int64(headerSize),
	}
}

//nolint:funlen
func (h *harBuilder) getHarResponse(ctx context.Context, cw *captureWriter, r *http.Request, startTime time.Time, c *controller) *har.Response {
	resHeaders := []*har.NameValuePair{}

	cookieParser := http.Response{Header: http.Header{}}

	for key, values := range cw.origResW.Header() {
		for _, value := range values {
			if key == "Set-Cookie" {
				cookieParser.Header.Add(key, value)
			}

			mask, ok := c.responseHeaderMasks[key]
			if ok {
				value = mask
			}

			resHeaders = append(resHeaders, &har.NameValuePair{Name: key, Value: value})
		}
	}

	sort.SliceStable(resHeaders, func(i, j int) bool {
		return resHeaders[i].Name < resHeaders[j].Name
	})

	resCookies := getHarCookies(cookieParser.Cookies(), startTime, c.responseCookieMasks)

	resContentType := cw.origResW.Header().Get("Content-Type")
	if resContentType == "" {
		resContentType = "application/octet-stream" // default http content type
	}

	bodyText := ""
	var bodySize int64 = -1
	contentBodySize := -1
	//nolint:nestif
	if cw.GetStatus() == http.StatusNotModified {
		bodySize = 0
	} else {
		if !cw.IsResValid() {
			bodyText = "--dropped--"
		} else {
			bodyText = cw.GetResBuffer().String()
			if cw.GetResBuffer().Len() > 0 {
				contentBodySize = cw.GetResBuffer().Len()
			}
		}

		contentLength := cw.origResW.Header().Get("Content-Length")
		if contentLength != "" {
			var err error
			//nolint:gomnd
			bodySize, err = strconv.ParseInt(contentLength, 10, 64)
			if err != nil {
				bodySize = -1
			}
		}
	}

	b := bytes.NewBuffer([]byte{})
	headerSize := -1
	if err := cw.origResW.Header().Write(b); err != nil {
		log.From(ctx).Error("speakeasy-sdk: failed to read length of response headers", zap.Error(err))
	} else {
		headerSize = b.Len()
	}

	if len(bodyText) > 0 {
		maskedBody, err := bodymasking.MaskBodyRegex(bodyText, resContentType, c.responseFieldMasksString, c.responseFieldMasksNumber)
		if err != nil {
			log.From(ctx).Error("speakeasy-sdk: failed to mask response body", zap.Error(err))
		} else {
			bodyText = maskedBody
		}
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
		BodySize:    bodySize,
	}
}

func getPostData(r *http.Request, cw *captureWriter, c *controller, ctx context.Context) *har.PostData {
	bodyText := "--dropped--"
	if cw.IsReqValid() {
		bodyText = cw.GetReqBuffer().String()
	}

	var postData *har.PostData
	if len(bodyText) == 0 {
		return nil
	}

	reqContentType := r.Header.Get("Content-Type")
	if reqContentType == "" {
		reqContentType = http.DetectContentType(cw.GetReqBuffer().Bytes())
		if reqContentType == "" {
			// default http content type
			reqContentType = "application/octet-stream"
		}
	}

	maskedBody, err := bodymasking.MaskBodyRegex(bodyText, reqContentType, c.responseFieldMasksString, c.responseFieldMasksNumber)
	if err != nil {
		log.From(ctx).Error("speakeasy-sdk: failed to mask request body", zap.Error(err))
	} else {
		bodyText = maskedBody
	}

	postData = &har.PostData{
		MimeType: reqContentType,
		Params:   []*har.Param{},
		Text:     bodyText,
	}
	return postData
}

func getHarCookies(cookies []*http.Cookie, startTime time.Time, masks map[string]string) []*har.Cookie {
	harCookies := []*har.Cookie{}
	for _, cookie := range cookies {
		mask, ok := masks[cookie.Name]
		if ok {
			cookie.Value = mask
		}

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
