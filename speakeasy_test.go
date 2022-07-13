package speakeasy_test

import (
	"errors"
	"net/http"
	"net/url"
	"os"
	"testing"

	"github.com/speakeasy-api/speakeasy-go-sdk"
	"github.com/stretchr/testify/assert"
)

func TestConfigure_Success(t *testing.T) {
	type fields struct {
		envServerURL string
	}
	type args struct {
		config speakeasy.Config
	}
	tests := []struct {
		name           string
		fields         fields
		args           args
		wantServerURL  string
		wantHTTPClient *http.Client
	}{
		{
			name: "successfully configures default instance with defaults",
			args: args{
				config: speakeasy.Config{
					APIKey: "12345",
				},
			},
			wantServerURL:  speakeasy.ExportServerURL,
			wantHTTPClient: http.DefaultClient,
		},
		{
			name: "successfully configures default instance with overrides from environment",
			fields: fields{
				envServerURL: "https://testapi.speakeasyapi.dev",
			},
			args: args{
				config: speakeasy.Config{
					APIKey: "12345",
				},
			},
			wantServerURL:  "https://testapi.speakeasyapi.dev",
			wantHTTPClient: http.DefaultClient,
		},
		{
			name: "successfully configures default instance with custom HTTPClient",
			fields: fields{
				envServerURL: "https://testapi.speakeasyapi.dev",
			},
			args: args{
				config: speakeasy.Config{
					APIKey: "12345",
				},
			},
			wantServerURL:  "https://testapi.speakeasyapi.dev",
			wantHTTPClient: &http.Client{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			speakeasy.ExportResetSpeakeasyDefaultInstance()

			if tt.fields.envServerURL != "" {
				os.Setenv("SPEAKEASY_SERVER_URL", tt.fields.envServerURL)
			}

			speakeasy.Configure(tt.args.config)

			defaultInstance := speakeasy.ExportGetSpeakeasyDefaultInstance()

			assert.NotNil(t, defaultInstance)

			config := defaultInstance.ExportGetSpeakeasyConfig()

			assert.Equal(t, tt.args.config.APIKey, config.APIKey)
			assert.Equal(t, tt.wantHTTPClient, config.HTTPClient)
			assert.Equal(t, tt.wantServerURL, defaultInstance.ExportGetSpeakeasyServerURL())
		})
	}
}

func TestConfigure_Error(t *testing.T) {
	type fields struct {
		envServerURL string
	}
	type args struct {
		config speakeasy.Config
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr error
	}{
		{
			name: "panics with missing APIKey",
			args: args{
				config: speakeasy.Config{},
			},
			wantErr: errors.New("API key is required"),
		},
		{
			name: "panics with invalid server URL",
			fields: fields{
				envServerURL: "hi/there?",
			},
			args: args{
				config: speakeasy.Config{
					APIKey: "12345",
				},
			},
			wantErr: &url.Error{Op: "parse", URL: "hi/there?", Err: errors.New("invalid URI for request")},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.PanicsWithError(t, tt.wantErr.Error(), func() {
				if tt.fields.envServerURL != "" {
					os.Setenv("SPEAKEASY_SERVER_URL", tt.fields.envServerURL)
				}

				speakeasy.Configure(tt.args.config)
			})
		})
	}
}
