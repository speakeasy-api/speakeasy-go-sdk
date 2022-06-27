package speakeasy

import (
	"context"
	"sync"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/google/uuid"
	"github.com/jinzhu/copier"
	"github.com/speakeasy-api/speakeasy-go-sdk/internal/log"
	"go.uber.org/zap"
)

var Config internalConfiguration

const defaultServerURL = "localhost:3000" // TODO: Inject appropriate Speakeasy Registry API endpoint either through env vars

// Configuration sets up and customizes communication with the Speakeasy Registry API
type Configuration struct {
	APIKey         string
	ServerURL      string
	SchemaFilePath string
}

// internalConfiguration is used for communication with Speakeasy Registry API
type internalConfiguration struct {
	Configuration
	apiServerId  uuid.UUID
	serverInfo   ServerInfo
	languageInfo LanguageInfo
}

type SpeakeasyApp struct {
	Lock     sync.RWMutex
	ApiStats map[string]ApiStats
	Schema   *openapi3.T
}

func Configure(config Configuration) (*SpeakeasyApp, error) {
	defer dontPanic(context.Background())
	if config.ServerURL != "" {
		Config.ServerURL = config.ServerURL
	} else {
		Config.ServerURL = defaultServerURL
	}
	if config.APIKey != "" {
		Config.APIKey = config.APIKey
	}
	if config.SchemaFilePath != "" {
		Config.SchemaFilePath = config.SchemaFilePath
	}

	Config.apiServerId = uuid.New()
	Config.serverInfo = getServerInfo()
	Config.languageInfo = getLanguageInfo()

	app := &SpeakeasyApp{Lock: sync.RWMutex{}, ApiStats: make(map[string]ApiStats), Schema: &openapi3.T{}}
	ctx := log.WithFields(context.Background(), zap.String("schema_file_path", Config.SchemaFilePath))
	// Populate map with all schema paths
	err := app.setApiStatsFromSchema(ctx, Config.SchemaFilePath)
	if err != nil {
		log.From(ctx).Error("failing speakeasy configuration; unable to load OpenAPI schema", zap.Error(err))
		return nil, err
	}
	// Call goroutine to send Api stats to Speakeasy
	go app.sendApiStatsToSpeakeasy(app.ApiStats, 5)

	return app, nil
}

func (app SpeakeasyApp) setApiStatsFromSchema(ctx context.Context, schemaFilePath string) error {
	err := app.loadOpenApiSchema(ctx, schemaFilePath)
	if err != nil {
		return err
	}
	var apiStats = make(map[string]ApiStats)
	for path := range app.Schema.Paths {
		// TODO register api here and get ApiId in lieu of path below
		apiStats[path] = ApiStats{NumCalls: 0, NumErrors: 0, NumUniqueCustomers: 0}
	}
	app.ApiStats = apiStats
	return nil
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
