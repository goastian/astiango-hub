package middlewares

import (
	"context"

	"github.com/goastian/astiango-hub/core/errors"
	"github.com/goastian/astiango-hub/core/utils"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const GrpcHeaderAuthorization = "authorization"

func GetGrpcServerAuthTokenFunc() grpc_auth.AuthFunc {
	return func(ctx context.Context) (ctx2 context.Context, err error) {
		// authentication (token verification)
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, errors.ErrorGrpcUnauthorized
		}

		// auth key from incoming context
		res, ok := md[GrpcHeaderAuthorization]
		if !ok {
			return ctx, errors.ErrorGrpcUnauthorized
		}
		if len(res) != 1 {
			return ctx, errors.ErrorGrpcUnauthorized
		}
		authKey := res[0]

		// validate
		svrAuthKey := utils.GetAuthKey()
		if authKey != svrAuthKey {
			return ctx, errors.ErrorGrpcUnauthorized
		}

		return ctx, nil
	}
}

func GetGrpcClientAuthTokenUnaryChainInterceptor() grpc.UnaryClientInterceptor {
	// set auth key
	md := metadata.Pairs(GrpcHeaderAuthorization, utils.GetAuthKey())
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		ctx = metadata.NewOutgoingContext(ctx, md)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func GetGrpcClientAuthTokenStreamChainInterceptor() grpc.StreamClientInterceptor {
	// set auth key
	md := metadata.Pairs(GrpcHeaderAuthorization, utils.GetAuthKey())
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		ctx = metadata.NewOutgoingContext(ctx, md)
		s, err := streamer(ctx, desc, cc, method, opts...)
		if err != nil {
			return nil, err
		}
		return s, nil
	}
}
