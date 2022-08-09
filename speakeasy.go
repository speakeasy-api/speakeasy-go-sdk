package speakeasy

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"regexp"
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
	speakeasyVersion = "1.1.0" // TODO get this from CI
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
	VersionID  string
	GRPCDialer func() func(context.Context, string) (net.Conn, error)
}

// Speakeasy is the concrete type for the Speakeasy SDK.
// Don't instantiate this directly, use Configure() or New() instead.
type Speakeasy struct {
	config    Config
	serverURL string
	secure    bool
}

// Configure allows you to configure the default instance of the Speakeasy SDK.
// Use this if you will use the same API Key for all connected APIs.
func Configure(config Config) {
	defaultInstance = New(config)
}

// New creates a new instance of the Speakeasy SDK.
// This allows you to create multiple instances of the SDK
// for specifying different API Keys for different APIs.
func New(config Config) *Speakeasy {
	s := &Speakeasy{}
	s.configure(config)

	return s
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

	s.serverURL = configuredServerURL
	s.secure = secure

	s.config = cfg
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
