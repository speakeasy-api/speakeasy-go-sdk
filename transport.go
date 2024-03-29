package speakeasy

import (
	"context"
	"crypto/tls"
	"net"
	"time"

	"github.com/speakeasy-api/speakeasy-go-sdk/internal/log"
	"github.com/speakeasy-api/speakeasy-schemas/grpc/go/registry/embedaccesstoken"
	"github.com/speakeasy-api/speakeasy-schemas/grpc/go/registry/ingest"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var GRPCIngestTimeout = 1 * time.Second

type DialerFunc func() func(context.Context, string) (net.Conn, error)

type GRPCClient struct {
	apiKey    string
	serverURL string
	secure    bool
	conn      *grpc.ClientConn
}

func newGRPCClient(ctx context.Context, apiKey, serverURL string, secure bool, grpcDialer DialerFunc) (*GRPCClient, error) {
	conn, err := createConn(ctx, secure, serverURL, grpcDialer)
	if err != nil {
		return nil, err
	}
	return &GRPCClient{
		apiKey:    apiKey,
		serverURL: serverURL,
		secure:    secure,
		conn:      conn,
	}, nil
}

func (c *GRPCClient) SendToIngest(ctx context.Context, req *ingest.IngestRequest) {
	ctx, cancel := context.WithTimeout(ctx, GRPCIngestTimeout)
	defer cancel()

	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("x-api-key", c.apiKey))

	_, err := ingest.NewIngestServiceClient(c.conn).Ingest(ctx, req)
	if err != nil {
		if status.Code(err) != codes.DeadlineExceeded {
			log.From(ctx).Error("speakeasy-sdk: failed to send ingest request", zap.Error(err))
		}
		return
	}
}

func (c *GRPCClient) GetEmbedAccessToken(ctx context.Context, req *embedaccesstoken.EmbedAccessTokenRequest) (string, error) {
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("x-api-key", c.apiKey))

	res, err := embedaccesstoken.NewEmbedAccessTokenServiceClient(c.conn).Get(ctx, req)
	if err != nil {
		return "", err
	}

	return res.AccessToken, nil
}

func createConn(ctx context.Context, secure bool, serverURL string, grpcDialer DialerFunc) (*grpc.ClientConn, error) {
	opts := []grpc.DialOption{}

	if secure {
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
			MinVersion: tls.VersionTLS12,
		})))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	if grpcDialer != nil {
		opts = append(opts, grpc.WithContextDialer(grpcDialer()))
	}

	conn, err := grpc.DialContext(ctx, serverURL, opts...)
	if err != nil {
		return nil, err
	}

	return conn, nil
}
