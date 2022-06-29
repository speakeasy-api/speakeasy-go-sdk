package speakeasy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/speakeasy-api/speakeasy-go-sdk/internal/log"
	"github.com/speakeasy-api/speakeasy-go-sdk/internal/models"
	"go.uber.org/zap"
)

const (
	timeoutDuration = 2 * time.Second
)

// Disable sending request/response info for every call to Speakeasy for now
// func sendToSpeakeasy(speakeasyInfo MetaData) {
// 	bytesRepresentation, err := json.Marshal(speakeasyInfo)
// 	if err != nil {
// 		return
// 	}

// 	req, err := http.NewRequest(http.MethodPost, Config.ServerURL, bytes.NewBuffer(bytesRepresentation))
// 	if err != nil {
// 		return
// 	}
// 	// Set the content type from the writer, it includes necessary boundary as well
// 	req.Header.Set("Content-Type", "application/json")
// 	req.Header.Set("x-api-key", Config.APIKey)

// 	// Do the request
// 	client := &http.Client{Timeout: timeoutDuration}
// 	_, _ = client.Do(req)
// }

func (app SpeakeasyApp) sendApiStatsToSpeakeasy(ctx context.Context, apiStatsById map[uint]*ApiStats, ticker *time.Ticker) {
	for {
		select {
		case <-ctx.Done():
			return
		case t := <-ticker.C:
			ctx := log.WithFields(ctx, zap.Time("timestamp", t))
			handlerInfo := HandlerInfo{ApiStatsById: apiStatsById}

			// Convert map state to ApiData
			apiData := &ApiData{ApiKey: app.APIKey, ApiServerId: app.apiServerId.String(), Handlers: handlerInfo}
			bytesRepresentation, err := json.Marshal(apiData)
			if err != nil {
				log.From(ctx).Error("failed to encode ApiData", zap.Error(err))
				return
			}
			metricsEndpoint := app.ServerURL + "/rs/v1/metrics"
			req, err := http.NewRequest(http.MethodPost, metricsEndpoint, bytes.NewBuffer(bytesRepresentation))
			if err != nil {
				log.From(ctx).Error("failed to create http request for Speakeasy metrics endpoint", zap.String("req_path", metricsEndpoint), zap.Error(err))
				return
			}
			// Set the content type from the writer, it includes necessary boundary as well
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("x-api-key", app.APIKey)

			// Do the request
			client := &http.Client{Timeout: timeoutDuration}
			startTime := time.Now()
			_, err = client.Do(req)
			if err != nil {
				log.From(ctx).Error("failed to get valid response for http request", zap.Time("start_time", startTime), zap.String("method", req.Method), zap.String("request_uri", req.RequestURI), zap.Duration("request_duration", time.Since(startTime)))
			}
		}
	}
}

func (app SpeakeasyApp) registerApi(api models.Api) {
	ctx := log.WithFields(context.Background(), zap.Any("api", api))

	bytesRepresentation, err := json.Marshal(api)
	if err != nil {
		log.From(ctx).Error("failed to encode Api", zap.Error(err))
		return
	}
	apiId := strconv.FormatUint(uint64(api.ID), 10)
	apiEndpoint := app.ServerURL + fmt.Sprintf("/rs/v1/apis/%s", apiId)
	req, err := http.NewRequest(http.MethodPost, apiEndpoint, bytes.NewBuffer(bytesRepresentation))
	if err != nil {
		log.From(ctx).Error("failed to create http request for Speakeasy api endpoint", zap.String("req_path", apiEndpoint), zap.Error(err))
		return
	}

	// Set the content type from the writer, it includes necessary boundary as well
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", app.APIKey)

	// Do the request
	client := &http.Client{Timeout: timeoutDuration}
	startTime := time.Now()
	_, err = client.Do(req)
	if err != nil {
		log.From(ctx).Error("failed to get valid response for http request", zap.Time("start_time", startTime), zap.String("method", req.Method), zap.String("request_uri", req.RequestURI), zap.Duration("request_duration", time.Since(startTime)))
	}
}

func (app SpeakeasyApp) registerSchema(schema models.Schema, mimeType string) {
	ctx := log.WithFields(context.Background(), zap.Any("schema", schema))

	bytesRepresentation, err := json.Marshal(schema)
	if err != nil {
		log.From(ctx).Error("failed to encode Schema", zap.Error(err))
		return
	}
	schemaId := strconv.FormatUint(uint64(schema.ID), 10)
	schemaEndpoint := app.ServerURL + fmt.Sprintf("/rs/v1/apis/%s/versions/%s/schemas/%s", schema.ApiId, schema.VersionId, schemaId)
	// TODO: add mimetype and stuff here
	buf := bytes.NewBuffer([]byte{})
	mw := multipart.NewWriter(buf)

	sw, err := mw.CreateFormField("schema")
	if err != nil {
		log.From(ctx).Error("failed to create schema writer", zap.Error(err))
		return
	}
	_, err = sw.Write(bytesRepresentation)
	if err != nil {
		log.From(ctx).Error("failed to write schema in to request body", zap.Error(err))
		return
	}

	mtw, err := mw.CreateFormField("mime_type")
	if err != nil {
		log.From(ctx).Error("failed to create mime_type writer", zap.Error(err))
		return
	}
	_, err = mtw.Write([]byte(mimeType))
	if err != nil {
		log.From(ctx).Error("failed to write mime_type in to request body", zap.Error(err))
		return
	}

	file, err := os.Open(app.SchemaFilePath)
	if err != nil {
		log.From(ctx).Error("failed to open schema filepath", zap.Error(err), zap.String("schema_filepath", app.SchemaFilePath))
		return
	}
	fw, err := mw.CreateFormFile("file", schema.Filename)
	if err != nil {
		log.From(ctx).Error("failed to create filewriter", zap.Error(err))
		return
	}
	_, err = io.Copy(fw, file)
	if err != nil {
		log.From(ctx).Error("failed to copy schema file contents to writer", zap.Error(err), zap.String("schema_filepath", app.SchemaFilePath))
		return
	}

	err = mw.Close()
	if err != nil {
		log.From(ctx).Error("failed to close multipart writer", zap.Error(err))
		return
	}

	err = file.Close()
	if err != nil {
		log.From(ctx).Error("failed to close schema file", zap.Error(err), zap.String("schema_filepath", app.SchemaFilePath))
		return
	}

	req, err := http.NewRequest(http.MethodPost, schemaEndpoint, buf)
	if err != nil {
		log.From(ctx).Error("failed to create http request for Speakeasy schema endpoint", zap.String("req_path", schemaEndpoint), zap.Error(err))
		return
	}

	// Set the content type from the writer, it includes necessary boundary as well
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("x-api-key", app.APIKey)

	// Do the request
	client := &http.Client{Timeout: timeoutDuration}
	startTime := time.Now()
	_, err = client.Do(req)
	if err != nil {
		log.From(ctx).Error("failed to get valid response for http request", zap.Time("start_time", startTime), zap.String("method", req.Method), zap.String("request_uri", req.RequestURI), zap.Duration("request_duration", time.Since(startTime)))
	}
}
