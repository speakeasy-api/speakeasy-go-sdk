package speakeasy

import (
	"context"
	"errors"
	"net"
	"os"
)

// ErrAPIKeyMissing is returned when the API Key is not provided at configuration time.
var ErrAPIKeyMissing = errors.New("API key is required")

const (
	sdkName = "speakeasy-go-sdk"
)

var (
	speakeasyVersion = "0.0.2" // TODO get this from CI
	serverURL        = "https://api.speakeasyapi.dev"

	defaultInstance *speakeasy
)

// Config provides configuration for the Speakeasy SDK.
type Config struct {
	APIKey     string
	GRPCDialer func() func(context.Context, string) (net.Conn, error)
}

type speakeasy struct {
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
func New(config Config) *speakeasy {
	s := &speakeasy{}
	s.configure(config)

	return s
}

func (s *speakeasy) configure(config Config) {
	if config.APIKey == "" {
		panic(ErrAPIKeyMissing)
	}

	configuredServerURL := serverURL

	envServerURL := os.Getenv("SPEAKEASY_SERVER_URL")
	if envServerURL != "" {
		configuredServerURL = envServerURL
	}

	secure := true
	envSecure := os.Getenv("SPEAKEASY_SERVER_SECURE")
	if envSecure == "false" {
		secure = false
	}

	s.serverURL = configuredServerURL
	s.secure = secure

	s.config = config
}
