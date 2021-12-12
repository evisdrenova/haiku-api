package requestid

import (
	"context"

	"google.golang.org/grpc"
)

type requestIDKey string

const (
	reqIDKey requestIDKey = "x-request-id"
)

func UnaryServerInterceptor(opt ...Option) grpc.UnaryServerInterceptor {
	opts := &options{
		gen: newDefaultRequestID,
	}
	for _, o := range opt {
		o(opts)
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		requestID := handleRequestID(ctx, opts.gen)
		ctx = setIncoming(ctx, requestID)
		res, err := handler(ctx, req)
		setOutgoing(ctx, requestID)
		return res, err
	}
}

func StreamServerInterceptor(opt ...Option) grpc.StreamServerInterceptor {
	opts := &options{
		gen: newDefaultRequestID,
	}
	for _, o := range opt {
		o(opts)
	}

	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		ctx := stream.Context()
		requestID := handleRequestID(ctx, opts.gen)
		ctx = context.WithValue(ctx, reqIDKey, requestID)
		stream = newServerStreamWithContext(stream, ctx)
		return handler(srv, stream)
	}
}

func FromContext(ctx context.Context) string {
	id, ok := ctx.Value(reqIDKey).(string)
	if !ok {
		return ""
	}
	return id
}

type serverStreamWithContext struct {
	grpc.ServerStream
	ctx context.Context
}

func (ss serverStreamWithContext) Context() context.Context {
	return ss.ctx
}

func newServerStreamWithContext(stream grpc.ServerStream, ctx context.Context) grpc.ServerStream {
	return serverStreamWithContext{
		ServerStream: stream,
		ctx:          ctx,
	}
}
