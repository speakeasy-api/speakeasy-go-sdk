package speakeasy

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/speakeasy-api/speakeasy-go-sdk/internal/log"
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

func (app SpeakeasyApp) sendApiStatsToSpeakeasy(apiStats map[string]ApiStats, intervalMinutes time.Duration) {
	for range time.Tick(intervalMinutes * time.Minute) {
		tick := time.Now()
		ctx := log.WithFields(context.Background(), zap.Time("timestamp", tick))
		handlerInfoList := app.getHandlerInfo(apiStats)

		// Convert map state to ApiData
		apiData := &ApiData{ApiKey: Config.APIKey, ApiServerId: Config.apiServerId.String(), Handlers: handlerInfoList}
		bytesRepresentation, err := json.Marshal(apiData)
		if err != nil {
			log.From(ctx).Error("failed to encode ApiData", zap.Error(err))
			return
		}

		metricsEndpoint := Config.ServerURL + "/metrics"
		req, err := http.NewRequest(http.MethodPost, metricsEndpoint, bytes.NewBuffer(bytesRepresentation))
		if err != nil {
			log.From(ctx).Error("failed to create http request for Speakeasy metrics endpoint", zap.String("req_path", metricsEndpoint), zap.Error(err))
			return
		}
		// Set the content type from the writer, it includes necessary boundary as well
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", Config.APIKey)

		// Do the request
		client := &http.Client{Timeout: timeoutDuration}
		startTime := time.Now()
		_, err = client.Do(req)
		if err != nil {
			log.From(ctx).Error("failed to get valid response for http request", zap.Time("start_time", startTime), zap.String("method", req.Method), zap.String("request_uri", req.RequestURI), zap.Duration("request_duration", time.Since(startTime)))
		}
	}
}

func (app SpeakeasyApp) getHandlerInfo(apiStats map[string]ApiStats) []HandlerInfo {
	app.Lock.RLock()
	defer app.Lock.RUnlock()
	var HandlerInfoList []HandlerInfo
	for path, stats := range apiStats {
		HandlerInfoList = append(HandlerInfoList, HandlerInfo{Path: path, ApiStats: stats})
	}
	return HandlerInfoList
}
