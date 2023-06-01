package speakeasy

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"regexp"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi-validator/paths"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/speakeasy-api/speakeasy-schemas/grpc/go/registry/embedaccesstoken"
)

var (
	// ErrAPIKeyMissing is returned when the API Key is not provided at configuration time.
	ErrAPIKeyMissing = errors.New("API key is required")
	// ErrAPIIdMissing is returned when the Api ID is not provided at configuration time.
	ErrApiIDMissing = errors.New("ApiID is required")
	// ErrApiIDMalformed is returned when the Api ID is invalid.
	ErrApiIDMalformed = errors.New("ApiID is malformed")
	// ErrVersionIDMissing is returned when the Version ID is not provided at configuration time.
	ErrVersionIDMissing = errors.New("VersionID is required")
	// ErrVersionIDMalformed is returned when the Version ID is invalid.
	ErrVersionIDMalformed = errors.New("VersionID is malformed")
)

const (
	sdkName = "speakeasy-go-sdk"
)

var (
	speakeasyVersion = "1.5.0" // TODO get this from CI
	serverURL        = "grpc.prod.speakeasyapi.dev:443"

	defaultInstance *Speakeasy
)

const (
	maxIDSize          = 128
	validCharsRegexStr = `[^a-zA-Z0-9.\-_~]`
)

var validCharsRegex = regexp.MustCompile(validCharsRegexStr)

// Config provides configuration for the Speakeasy SDK.
type Config struct {
	// APIKey is the API Key obtained from the Speakeasy platform for capturing requests to a particular workspace.
	APIKey string
	// ApiID is the ID of the Api to associate any requests captured by this instance of the SDK to.
	ApiID string
	// VersionID is the ID of the Api Version to associate any requests captured by this instance of the SDK to.
	VersionID       string
	OpenAPIDocument []byte
	GRPCDialer      func() func(context.Context, string) (net.Conn, error)
}

// Speakeasy is the concrete type for the Speakeasy SDK.
// Don't instantiate this directly, use Configure() or New() instead.
type Speakeasy struct {
	config     Config
	harBuilder harBuilder
	grpcClient *GRPCClient
	doc        *libopenapi.DocumentModel[v3.Document]
}

// Configure allows you to configure the default instance of the Speakeasy SDK.
// Use this if you will use the same API Key for all connected APIs.
func Configure(config Config) {
	globalInstance := New(config)
	defaultInstance = globalInstance
}

// New creates a new instance of the Speakeasy SDK.
// This allows you to create multiple instances of the SDK
// for specifying different API Keys for different APIs.
func New(config Config) *Speakeasy {
	s := &Speakeasy{}
	s.configure(config)
	return s
}

func GetEmbedAccessToken(ctx context.Context, req *embedaccesstoken.EmbedAccessTokenRequest) (string, error) {
	return defaultInstance.GetEmbedAccessToken(ctx, req)
}

func Close() error {
	return defaultInstance.Close()
}

func (s *Speakeasy) GetEmbedAccessToken(ctx context.Context, req *embedaccesstoken.EmbedAccessTokenRequest) (string, error) {
	return s.grpcClient.GetEmbedAccessToken(ctx, req)
}

func (s *Speakeasy) Close() error {
	return s.grpcClient.conn.Close()
}

func (s *Speakeasy) MatchOpenAPIPath(r *http.Request) string {
	if s.doc != nil {
		_, _, pathHint := paths.FindPath(r, &s.doc.Model)
		if pathHint != "" {
			return pathHint
		}
	}

	return ""
}

func (s *Speakeasy) configure(cfg Config) {
	mustValidateConfig(cfg)

	// The below environment variables allow the overriding of the location of the ingest server.
	// Useful for testing or on-premise deployments.

	// SPEAKEASY_SERVER_URL allows the overriding of the endpoint to send the ingest request to.
	configuredServerURL := serverURL
	envServerURL := os.Getenv("SPEAKEASY_SERVER_URL")
	if envServerURL != "" {
		configuredServerURL = envServerURL
	}

	// SPEAKEASY_SERVER_SECURE allows the need for TLS connections to be disabled.
	secure := true
	envSecure := os.Getenv("SPEAKEASY_SERVER_SECURE")
	if envSecure == "false" {
		secure = false
	}

	s.config = cfg

	grpcClient, err := newGRPCClient(context.Background(), s.config.APIKey, configuredServerURL, secure, s.config.GRPCDialer)
	s.grpcClient = grpcClient
	if err != nil {
		panic(err)
	}

	if len(s.config.OpenAPIDocument) > 0 {
		doc, err := libopenapi.NewDocument(s.config.OpenAPIDocument)
		if err != nil {
			panic(fmt.Errorf("failed to parse OpenAPI document: %w", err))
		}

		v3Doc, errs := doc.BuildV3Model()
		if len(errs) > 0 {
			panic(fmt.Sprintf("failed to build OpenAPI v3 model: %v", errs))
		}

		s.doc = v3Doc
	}
}

func mustValidateConfig(cfg Config) {
	if cfg.APIKey == "" {
		panic(ErrAPIKeyMissing)
	}

	if cfg.ApiID == "" {
		panic(ErrApiIDMissing)
	}

	if len(cfg.ApiID) > maxIDSize {
		panic(fmt.Errorf("ApiID is too long. Max length is %d: %w", maxIDSize, ErrApiIDMalformed))
	}

	if validCharsRegex.MatchString(cfg.ApiID) {
		panic(fmt.Errorf("ApiID contains invalid characters %s: %w", validCharsRegexStr, ErrApiIDMalformed))
	}

	if cfg.VersionID == "" {
		panic(ErrVersionIDMissing)
	}

	if len(cfg.VersionID) > maxIDSize {
		panic(fmt.Errorf("VersionID is too long. Max length is %d: %w", maxIDSize, ErrVersionIDMalformed))
	}

	if validCharsRegex.MatchString(cfg.VersionID) {
		panic(fmt.Errorf("VersionID contains invalid characters %s: %w", validCharsRegexStr, ErrVersionIDMalformed))
	}
}
