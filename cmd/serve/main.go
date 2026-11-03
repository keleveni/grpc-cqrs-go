package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"

	"buf.build/go/protovalidate"
	"github.com/cockroachdb/errors"
	protovalidatemw "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/protovalidate"
	"github.com/kodaikumatani/grpc-cqrs-go/internal/interceptor"
	"github.com/kodaikumatani/grpc-cqrs-go/internal/logger"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var (
	port = flag.Int("port", 50051, "The server port")
	// host はデフォルトで loopback。app はヘッダ(x-endpoint-api-userinfo)を信頼するため、
	// 前段(同一 Pod の Envoy 等)からのみ到達させる。別コンテナから叩く構成では 0.0.0.0 に上書き。
	host = flag.String("host", "127.0.0.1", "The server listen address")
)

func run(ctx context.Context) error {
	flag.Parse()

	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", *host, *port))
	if err != nil {
		return errors.Wrap(err, "failed to listen")
	}

	services, cleanup, err := initializeServices(ctx, os.Getenv("DATABASE_URL"))
	defer cleanup()
	if err != nil {
		return errors.Wrap(err, "failed to initialize services")
	}

	validator, err := protovalidate.New()
	if err != nil {
		return errors.Wrap(err, "failed to create validator")
	}

	s := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptor.LoggingUnaryInterceptor(),
			interceptor.RecoveryUnaryInterceptor(),
			interceptor.ErrorHandlingUnaryInterceptor(),
			interceptor.GatewayAuthUnaryInterceptor(),
			// 入力の形式検証は proto の buf.validate ルールで一元的に行う（認証の後）
			protovalidatemw.UnaryServerInterceptor(validator),
		),
		grpc.ChainStreamInterceptor(
			interceptor.LoggingStreamInterceptor(),
			interceptor.RecoveryStreamInterceptor(),
		),
	)
	s = services.Register(s)
	reflection.Register(s)

	// grpc.health.v1.Health を登録し、DB 疎通をステータスに反映させる。
	// これが無いと「プロセスは生きているが応答できない」状態を外から判定できず、
	// k8s の Pod が Ready のままトラフィックを受け続けてしまう。
	services.Health.Register(s)
	healthCtx, stopHealth := context.WithCancel(ctx)
	defer stopHealth()
	go services.Health.Start(healthCtx)

	log.Ctx(ctx).Info().Msgf("server listening at %v", lis.Addr())

	if err := s.Serve(lis); err != nil {
		return errors.Wrap(err, "failed to serve")
	}

	return nil
}

func main() {
	ctx := context.Background()
	ctx = logger.WithLogger(ctx)

	if err := run(ctx); err != nil {
		log.Ctx(ctx).Fatal().Err(err).Send()
	}
}
