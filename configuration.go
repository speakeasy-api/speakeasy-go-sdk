package speakeasy

import (
	"context"
	"hash/fnv"
	"net/http"
	urlPath "path"
	"strconv"
	"sync"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/google/uuid"
	"github.com/jinzhu/copier"
	"github.com/speakeasy-api/speakeasy-go-sdk/internal/log"
	"github.com/speakeasy-api/speakeasy-go-sdk/internal/models"
	"github.com/speakeasy-api/speakeasy-schemas/pkg/metrics"
	"go.uber.org/zap"
)

var Config internalConfiguration

const defaultServerURL = "localhost:3000" // TODO: Inject appropriate Speakeasy Registry API endpoint either through env vars
const defaultApiStatsIntervalSeconds = 300

// Configuration sets up and customizes communication with the Speakeasy Registry API
type Configuration struct {
	APIKey                  string
	WorkspaceId             string
	ServerURL               string
	SchemaFilePath          string
	ApiStatsIntervalSeconds int
}

// internalConfiguration is used for communication with Speakeasy Registry API
type internalConfiguration struct {
	serverInfo   ServerInfo
	languageInfo LanguageInfo
}

type SpeakeasyApp struct {
	Configuration
	apiServerId    uuid.UUID
	CancelApiStats context.CancelFunc
	Lock           sync.RWMutex
	ApiStatsById   map[string]*metrics.ApiStats
	ApiByPath      map[string]models.Api
	Schema         *openapi3.T
}

func Configure(config Configuration) (*SpeakeasyApp, error) {
	defer dontPanic(context.Background())
	app := &SpeakeasyApp{Lock: sync.RWMutex{}, ApiStatsById: make(map[string]*metrics.ApiStats), ApiByPath: make(map[string]models.Api), Schema: &openapi3.T{}}
	if config.ServerURL != "" {
		app.ServerURL = config.ServerURL
	} else {
		app.ServerURL = defaultServerURL
	}
	if config.ApiStatsIntervalSeconds != 0 {
		app.ApiStatsIntervalSeconds = config.ApiStatsIntervalSeconds
	} else {
		app.ApiStatsIntervalSeconds = defaultApiStatsIntervalSeconds
	}
	app.APIKey = config.APIKey
	app.WorkspaceId = config.WorkspaceId
	app.SchemaFilePath = config.SchemaFilePath

	app.apiServerId = uuid.New()
	Config.serverInfo = getServerInfo()
	Config.languageInfo = getLanguageInfo()

	ctx := log.WithFields(context.Background(), zap.String("schema_file_path", app.SchemaFilePath))

	// Populate map with all schema paths
	err := app.registerApiAndSetStats(ctx, app.SchemaFilePath)
	if err != nil {
		log.From(ctx).Error("failing speakeasy configuration; unable to load OpenAPI schema", zap.Error(err))
		return nil, err
	}
	// Call goroutine to send Api stats to Speakeasy
	ticker := time.NewTicker(time.Duration(app.ApiStatsIntervalSeconds) * time.Second)

	var cancelCtx context.Context
	cancelCtx, app.CancelApiStats = context.WithCancel(context.Background())
	go app.sendApiStatsToSpeakeasy(cancelCtx, app.ApiStatsById, ticker)

	return app, nil
}

func (app SpeakeasyApp) registerApiAndSetStats(ctx context.Context, schemaFilePath string) error {
	err := app.loadOpenApiSchema(ctx, schemaFilePath)
	if err != nil {
		return err
	}
	for path, pathItem := range app.Schema.Paths {
		// register api
		method, op := methodAndOpFromPathItem(ctx, path, pathItem)
		apiId := hash(app.WorkspaceId + method + path)
		api := models.Api{ID: apiId, WorkspaceId: app.WorkspaceId, Method: method, Path: path, DisplayName: op.OperationID, Description: op.Summary}
		go app.registerApi(api)
		apiIdStr := strconv.FormatUint(uint64(apiId), 10)
		app.ApiStatsById[apiIdStr] = &metrics.ApiStats{NumCalls: 0, NumErrors: 0}
		app.ApiByPath[path] = api

		// register schema
		schema := models.Schema{ID: hash(apiIdStr), ApiId: apiIdStr, VersionId: app.Schema.Info.Version, Filename: urlPath.Base(schemaFilePath), Description: app.Schema.Info.Description}
		mimeType := "application/json"
		go app.registerSchema(schema, mimeType)
	}
	return nil
}

func methodAndOpFromPathItem(ctx context.Context, path string, pathItem *openapi3.PathItem) (string, *openapi3.Operation) {
	if pathItem.Get != nil {
		return http.MethodGet, pathItem.Get
	} else if pathItem.Post != nil {
		return http.MethodPost, pathItem.Post
	} else if pathItem.Connect != nil {
		return http.MethodConnect, pathItem.Connect
	} else if pathItem.Put != nil {
		return http.MethodPut, pathItem.Put
	} else if pathItem.Patch != nil {
		return http.MethodPatch, pathItem.Patch
	} else if pathItem.Delete != nil {
		return http.MethodDelete, pathItem.Delete
	} else if pathItem.Head != nil {
		return http.MethodHead, pathItem.Head
	} else if pathItem.Options != nil {
		return http.MethodOptions, pathItem.Options
	} else if pathItem.Trace != nil {
		return http.MethodTrace, pathItem.Trace
	} else {
		log.From(ctx).Error("supported HTTP method not found in schema's path item", zap.String("path", path), zap.Any("path_item", pathItem))
	}
	return "", nil
}

func (app SpeakeasyApp) loadOpenApiSchema(ctx context.Context, schemaFilePath string) error {
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true
	schema, err := loader.LoadFromFile(schemaFilePath)
	if err != nil {
		log.From(ctx).Error("failed to load OpenAPI schema from file", zap.Error(err))
		return err
	}
	err = schema.Validate(loader.Context)
	if err != nil {
		log.From(ctx).Error("not a valid OpenAPI schema", zap.Error(err))
		return err
	}
	copier.Copy(app.Schema, schema)
	return nil
}

func hash(s string) uint {
	h := fnv.New32a()
	h.Write([]byte(s))
	return uint(h.Sum32())
}
