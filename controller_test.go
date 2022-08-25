//nolint:testpackage
package speakeasy

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithQueryStringMask(t *testing.T) {
	type args struct {
		keys  []string
		masks []string
	}
	tests := []struct {
		name                 string
		args                 args
		wantQueryStringMasks map[string]string
	}{
		{
			name: "successfully adds single query string with default mask",
			args: args{
				keys:  []string{"test"},
				masks: []string{},
			},
			wantQueryStringMasks: map[string]string{
				"test": DefaultStringMask,
			},
		},
		{
			name: "successfully adds single query string with custom mask",
			args: args{
				keys:  []string{"test"},
				masks: []string{"testmask"},
			},
			wantQueryStringMasks: map[string]string{
				"test": "testmask",
			},
		},
		{
			name: "successfully adds multiple query string with default mask",
			args: args{
				keys:  []string{"test", "test2", "test3"},
				masks: []string{},
			},
			wantQueryStringMasks: map[string]string{
				"test":  DefaultStringMask,
				"test2": DefaultStringMask,
				"test3": DefaultStringMask,
			},
		},
		{
			name: "successfully adds multiple query string with single custom mask",
			args: args{
				keys:  []string{"test", "test2", "test3"},
				masks: []string{"testmask"},
			},
			wantQueryStringMasks: map[string]string{
				"test":  "testmask",
				"test2": "testmask",
				"test3": "testmask",
			},
		},
		{
			name: "successfully adds multiple query string with multiple matched custom masks",
			args: args{
				keys:  []string{"test", "test2", "test3"},
				masks: []string{"testmask", "test2mask", "test3mask"},
			},
			wantQueryStringMasks: map[string]string{
				"test":  "testmask",
				"test2": "test2mask",
				"test3": "test3mask",
			},
		},
		{
			name: "successfully adds multiple query string with multiple unmatched custom masks",
			args: args{
				keys:  []string{"test", "test2", "test3"},
				masks: []string{"testmask", "test2mask"},
			},
			wantQueryStringMasks: map[string]string{
				"test":  "testmask",
				"test2": "test2mask",
				"test3": DefaultStringMask,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, c := contextWithController(context.Background(), nil)
			c.Masking(WithQueryStringMask(tt.args.keys, tt.args.masks...))
			assert.Equal(t, tt.wantQueryStringMasks, c.queryStringMasks)
		})
	}
}
