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
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

var GRPCIngestTimeout = 1 * time.Second

type DialerFunc func() func(context.Context, string) (net.Conn, error)

type GRPCClient struct {
	apiKey     string
	serverURL  string
	secure     bool
	grpcDialer DialerFunc
}

func newGRPCClient(apiKey, serverURL string, secure bool, grpcDialer DialerFunc) *GRPCClient {
	return &GRPCClient{
		apiKey:     apiKey,
		serverURL:  serverURL,
		secure:     secure,
		grpcDialer: grpcDialer,
	}
}

func (c *GRPCClient) SendToIngest(ctx context.Context, req *ingest.IngestRequest) {
	ctx, cancel := context.WithTimeout(ctx, GRPCIngestTimeout)
	defer cancel()

	conn, err := c.getConn(ctx)
	if err != nil {
		log.From(ctx).Error("speakeasy-sdk: failed to create grpc connection", zap.Error(err))
		return
	}
	defer conn.Close()

	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("x-api-key", c.apiKey))

	_, err = ingest.NewIngestServiceClient(conn).Ingest(ctx, req)
	if err != nil {
		log.From(ctx).Error("speakeasy-sdk: failed to send ingest request", zap.Error(err))
		return
	}
}

func (c *GRPCClient) GetEmbedAccessToken(ctx context.Context, req *embedaccesstoken.EmbedAccessTokenRequest) (string, error) {
	conn, err := c.getConn(ctx)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("x-api-key", c.apiKey))

	res, err := embedaccesstoken.NewEmbedAccessTokenServiceClient(conn).Get(ctx, req)
	if err != nil {
		return "", err
	}

	return res.AccessToken, nil
}

func (c *GRPCClient) getConn(ctx context.Context) (*grpc.ClientConn, error) {
	opts := []grpc.DialOption{}

	if c.secure {
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
			MinVersion: tls.VersionTLS12,
		})))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	if c.grpcDialer != nil {
		opts = append(opts, grpc.WithContextDialer(c.grpcDialer()))
	}

	conn, err := grpc.DialContext(ctx, c.serverURL, opts...)
	if err != nil {
		return nil, err
	}

	return conn, nil
}
