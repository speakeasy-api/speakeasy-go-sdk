package pathhints_test

import (
	"testing"

	"github.com/speakeasy-api/speakeasy-go-sdk/internal/pathhints"
	"github.com/stretchr/testify/assert"
)

func TestNormalizePathHint_Success(t *testing.T) {
	type args struct {
		pathHint string
	}
	tests := []struct {
		name           string
		args           args
		wantNormalized string
	}{
		{
			name: "normalizes gorilla mux path hint",
			args: args{
				pathHint: "/user/{id}/account/{accountID:[0-9]+}",
			},
			wantNormalized: "/user/{id}/account/{accountID}",
		},
		{
			name: "normalizes chi path hint",
			args: args{
				pathHint: "/user/{id}/account/*",
			},
			wantNormalized: "/user/{id}/account/{wildcard}",
		},
		{
			name: "normalizes gin path hint",
			args: args{
				pathHint: "/user/{id}/account/*action",
			},
			wantNormalized: "/user/{id}/account/{action}",
		},
		{
			name: "normalizes simple echo path hint",
			args: args{
				pathHint: "/user/:id",
			},
			wantNormalized: "/user/{id}",
		},
		{
			name: "normalizes complex echo path hint",
			args: args{
				pathHint: "/user/:id/account/*action",
			},
			wantNormalized: "/user/{id}/account/{action}",
		},
		{
			name: "doesn't normalize an unknown format",
			args: args{
				pathHint: "/user/<id>/account/<accountID>",
			},
			wantNormalized: "/user/<id>/account/<accountID>",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalized := pathhints.NormalizePathHint(tt.args.pathHint)
			assert.Equal(t, tt.wantNormalized, normalized)
		})
	}
}
