package bodymasking_test

import (
	"testing"

	"github.com/speakeasy-api/speakeasy-go-sdk/internal/bodymasking"
	"github.com/stretchr/testify/assert"
)

func TestMaskBodyRegex_Success(t *testing.T) {
	type args struct {
		body        string
		mimeType    string
		stringMasks map[string]string
		numberMasks map[string]string
	}
	tests := []struct {
		name     string
		args     args
		wantBody string
	}{
		{
			name: "successfully masks body with single string field",
			args: args{
				body:     `{"test": "test"}`,
				mimeType: "application/json",
				stringMasks: map[string]string{
					"test": "testmask",
				},
				numberMasks: map[string]string{},
			},
			wantBody: `{"test": "testmask"}`,
		},
		{
			name: "successfully masks body with single int field",
			args: args{
				body:        `{"test": 123}`,
				mimeType:    "application/json",
				stringMasks: map[string]string{},
				numberMasks: map[string]string{
					"test": "-123456789",
				},
			},
			wantBody: `{"test": -123456789}`,
		},
		{
			name: "successfully masks body with single negative field",
			args: args{
				body:        `{"test": -123}`,
				mimeType:    "application/json",
				stringMasks: map[string]string{},
				numberMasks: map[string]string{
					"test": "-123456789",
				},
			},
			wantBody: `{"test": -123456789}`,
		},
		{
			name: "successfully masks body with single float field",
			args: args{
				body:        `{"test": 123.123}`,
				mimeType:    "application/json",
				stringMasks: map[string]string{},
				numberMasks: map[string]string{
					"test": "-123456789",
				},
			},
			wantBody: `{"test": -123456789}`,
		},
		{
			name: "successfully masks body with nested fields",
			args: args{
				body:     `{"test": {"test": "test", "test1": 123}}`,
				mimeType: "application/json",
				stringMasks: map[string]string{
					"test": "testmask",
				},
				numberMasks: map[string]string{
					"test1": "-123456789",
				},
			},
			wantBody: `{"test": {"test": "testmask", "test1": -123456789}}`,
		},
		{
			name: "successfully masks formatted body",
			args: args{
				body: `{
			"test": {
				"test": "test",
				"test1": 123
			}
		}`,
				mimeType: "application/json",
				stringMasks: map[string]string{
					"test": "testmask",
				},
				numberMasks: map[string]string{
					"test1": "-123456789",
				},
			},
			wantBody: `{
			"test": {
				"test": "testmask",
				"test1": -123456789
			}
		}`,
		},
		{
			name: "successfully masks body with complex string field",
			args: args{
				body:     `{"test": "\",{abc}: .\""}`,
				mimeType: "application/json",
				stringMasks: map[string]string{
					"test": "testmask",
				},
				numberMasks: map[string]string{},
			},
			wantBody: `{"test": "testmask"}`,
		},
		{
			name: "successfully masks body with complex field key",
			args: args{
				body:     `{"test\"hello\": ": "\",{abc}: .\""}`,
				mimeType: "application/json",
				stringMasks: map[string]string{
					`test\"hello\": `: "testmask",
				},
				numberMasks: map[string]string{},
			},
			wantBody: `{"test\"hello\": ": "testmask"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			maskedBody, err := bodymasking.MaskBodyRegex(tt.args.body, tt.args.mimeType, tt.args.stringMasks, tt.args.numberMasks)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantBody, maskedBody)
		})
	}
}
