package requestid

import (
	"context"

	"github.com/lithammer/shortuuid/v3"
	"google.golang.org/grpc/metadata"
)

const defaultXRequestIDKey string = "x-request-id"

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
	return shortuuid.NewWithNamespace("haiku.io")
}
