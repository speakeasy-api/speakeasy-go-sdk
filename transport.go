package speakeasy

import (
	"context"
	"crypto/tls"

	"github.com/speakeasy-api/speakeasy-go-sdk/internal/log"
	"github.com/speakeasy-api/speakeasy-schemas/grpc/go/registry/ingest"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func (s *Speakeasy) sendToIngest(ctx context.Context, req *ingest.IngestRequest) {
	opts := []grpc.DialOption{}

	if s.secure {
		//nolint: gosec
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	if s.config.GRPCDialer != nil {
		opts = append(opts, grpc.WithContextDialer(s.config.GRPCDialer()))
	}

	conn, err := grpc.DialContext(ctx, s.serverURL, opts...)
	if err != nil {
		log.From(ctx).Error("speakeasy-sdk: failed to create grpc connection", zap.Error(err))
		return
	}
	defer conn.Close()

	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("x-api-key", s.config.APIKey))

	_, err = ingest.NewIngestServiceClient(conn).Ingest(ctx, req)
	if err != nil {
		log.From(ctx).Error("speakeasy-sdk: failed to send ingest request", zap.Error(err))
		return
	}
}
