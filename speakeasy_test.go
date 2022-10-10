package speakeasy_test

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/speakeasy-api/speakeasy-go-sdk"
	"github.com/stretchr/testify/assert"
)

func TestConfigure_Success(t *testing.T) {
	type fields struct {
		envServerURL string
		envSecure    *bool
	}
	type args struct {
		config speakeasy.Config
	}
	tests := []struct {
		name          string
		fields        fields
		args          args
		wantServerURL string
		wantSecure    bool
	}{
		{
			name: "successfully configures default instance with defaults",
			args: args{
				config: speakeasy.Config{
					APIKey:    "12345",
					ApiID:     "testapi1",
					VersionID: "v1.0.0",
				},
			},
			wantServerURL: speakeasy.ExportServerURL,
			wantSecure:    true,
		},
		{
			name: "successfully configures default instance with overrides from environment",
			fields: fields{
				envServerURL: "https://testapi.speakeasyapi.dev",
				envSecure:    pointer.ToBool(false),
			},
			args: args{
				config: speakeasy.Config{
					APIKey:    "12345",
					ApiID:     "testapi1",
					VersionID: "testversion1",
				},
			},
			wantServerURL: "https://testapi.speakeasyapi.dev",
			wantSecure:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Unsetenv("SPEAKEASY_SERVER_URL")
			os.Unsetenv("SPEAKEASY_SERVER_SECURE")

			if tt.fields.envServerURL != "" {
				os.Setenv("SPEAKEASY_SERVER_URL", tt.fields.envServerURL)
			}

			if tt.fields.envSecure != nil {
				os.Setenv("SPEAKEASY_SERVER_SECURE", strconv.FormatBool(*tt.fields.envSecure))
			}

			sdkInstance := speakeasy.New(tt.args.config)
			assert.NotNil(t, sdkInstance)

			config := sdkInstance.ExportGetSpeakeasyConfig()

			assert.Equal(t, tt.args.config.APIKey, config.APIKey)
			assert.Equal(t, tt.wantServerURL, sdkInstance.ExportGetSpeakeasyServerURL())
			assert.Equal(t, tt.wantSecure, sdkInstance.ExportGetSpeakeasyServerSecure())
		})
	}
}

func TestConfigure_Error(t *testing.T) {
	type args struct {
		config speakeasy.Config
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr string
	}{
		{
			name: "panics with missing APIKey",
			args: args{
				config: speakeasy.Config{},
			},
			wantErr: speakeasy.ErrAPIKeyMissing.Error(),
		},
		{
			name: "panics with missing ApiID",
			args: args{
				config: speakeasy.Config{
					APIKey: "12345",
				},
			},
			wantErr: speakeasy.ErrApiIDMissing.Error(),
		},
		{
			name: "panics with too long ApiID",
			args: args{
				config: speakeasy.Config{
					APIKey: "12345",
					ApiID:  randStringRunes(speakeasy.ExportMaxIDSize + 1),
				},
			},
			wantErr: fmt.Sprintf("ApiID is too long. Max length is %d: ApiID is malformed", speakeasy.ExportMaxIDSize),
		},
		{
			name: "panics with illegal chars in ApiID",
			args: args{
				config: speakeasy.Config{
					APIKey: "12345",
					ApiID:  "test api 1",
				},
			},
			wantErr: `ApiID contains invalid characters [^a-zA-Z0-9.\-_~]: ApiID is malformed`,
		},
		{
			name: "panics with missing VersionID",
			args: args{
				config: speakeasy.Config{
					APIKey: "12345",
					ApiID:  "testapi1",
				},
			},
			wantErr: speakeasy.ErrVersionIDMissing.Error(),
		},
		{
			name: "panics with too long VersionID",
			args: args{
				config: speakeasy.Config{
					APIKey:    "12345",
					ApiID:     "testapi1",
					VersionID: randStringRunes(speakeasy.ExportMaxIDSize + 1),
				},
			},
			wantErr: fmt.Sprintf("VersionID is too long. Max length is %d: VersionID is malformed", speakeasy.ExportMaxIDSize),
		},
		{
			name: "panics with illegal chars in VersionID",
			args: args{
				config: speakeasy.Config{
					APIKey:    "12345",
					ApiID:     "testapi1",
					VersionID: "v1,0,0",
				},
			},
			wantErr: `VersionID contains invalid characters [^a-zA-Z0-9.\-_~]: VersionID is malformed`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.PanicsWithError(t, tt.wantErr, func() {
				_ = speakeasy.New(tt.args.config)
			})
		})
	}
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
