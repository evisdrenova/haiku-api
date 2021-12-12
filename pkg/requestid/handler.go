package requestid

import (
	"context"

	"github.com/lithammer/shortuuid/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const defaultXRequestIDKey string = "x-request-id"

func setIncoming(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, reqIDKey, requestID)
}

// see https://github.com/grpc/grpc-go/blob/master/Documentation/grpc-metadata.md
func setOutgoing(ctx context.Context, requestID string) {
	trailer := metadata.Pairs(defaultXRequestIDKey, requestID)
	grpc.SetTrailer(ctx, trailer)
}

func handleRequestID(ctx context.Context, gen IDGenerator) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return gen()
	}

	header, ok := md[defaultXRequestIDKey]
	if !ok || len(header) == 0 {
		return gen()
	}

	requestID := header[0]
	if requestID == "" {
		return gen()
	}

	return requestID
}

func newDefaultRequestID() string {
	return shortuuid.New()
}
