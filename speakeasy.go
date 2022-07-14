package speakeasy

import (
	"errors"
	"net/http"
	"os"
)

// ErrAPIKeyMissing is returned when the API Key is not provided at configuration time.
var ErrAPIKeyMissing = errors.New("API key is required")

const (
	sdkName = "speakeasy-go-sdk"

	ingestAPI = "/rs/v1/ingest"
)

var (
	speakeasyVersion = "0.0.1"
	serverURL        = "https://api.speakeasyapi.dev"

	defaultInstance *speakeasy
)

// Config provides configuration for the Speakeasy SDK.
type Config struct {
	APIKey     string
	HTTPClient *http.Client
}

type speakeasy struct {
	config    Config
	serverURL string
}

// Configure allows you to configure the default instance of the Speakeasy SDK.
// Use this if you will use the same API Key for all connected APIs.
func Configure(config Config) {
	defaultInstance = New(config)
}

// New creates a new instance of the Speakeasy SDK.
// This allows you to create multiple instances of the SDK
// for specifying different API Keys for different APIs.
func New(config Config) *speakeasy {
	s := &speakeasy{}
	s.configure(config)

	return s
}

func (s *speakeasy) configure(config Config) {
	if config.APIKey == "" {
		panic(ErrAPIKeyMissing)
	}

	if config.HTTPClient == nil {
		config.HTTPClient = http.DefaultClient
	}

	configuredServerURL := serverURL

	envServerURL := os.Getenv("SPEAKEASY_SERVER_URL")
	if envServerURL != "" {
		configuredServerURL = envServerURL
	}

	s.serverURL = configuredServerURL

	s.config = config
}
