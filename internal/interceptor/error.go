package interceptor

import (
	"context"
	"errors"

	"github.com/kodaikumatani/grpc-cqrs-go/internal/authz"
	"github.com/kodaikumatani/grpc-cqrs-go/internal/identity"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	msgInternalServerError = "internal server error"
	msgPermissionDenied    = "permission denied"
)

// ErrorHandlingUnaryInterceptor は handler が返したエラーを一元的に処理する。
//   - handler が付けた status（既知エラー）: cause をログに残し、安全な status を返す
//   - 横断的な authz エラー: PermissionDenied に変換
//   - 横断的な identity エラー（ctx に認証済みユーザーがいない）: Unauthenticated に変換
//   - 未変換の生エラー（想定外の内部エラー）: cause を error ログに残し、client には汎用文言
//
// ログレベルは gRPC コードで決める（想定内のクライアントエラーは info、サーバ起因のみ
// warn/error）。これにより内部エラーを client に漏らさず、cause は常にログへ残しつつ、
// PermissionDenied 等の日常的な拒否で warn を溢れさせない。
func ErrorHandlingUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		resp, err := handler(ctx, req)
		if err == nil {
			return resp, nil
		}

		if st, ok := status.FromError(err); ok {
			logStatus(ctx, st.Code(), err, info.FullMethod, "request failed")
			return resp, st.Err()
		}

		// 横断的な authz エラーはここで一元変換する（各 handler で繰り返さない）。
		if errors.Is(err, authz.ErrPermissionDenied) {
			logStatus(ctx, codes.PermissionDenied, err, info.FullMethod, "permission denied")
			return resp, status.Error(codes.PermissionDenied, msgPermissionDenied)
		}

		// 通常は GatewayAuthUnaryInterceptor が先に弾くので、ここに来るのは認証の
		// 除外設定と handler の想定がずれたとき。各 handler で変換を繰り返さないよう一元化する。
		if errors.Is(err, identity.ErrUnauthenticated) {
			logStatus(ctx, codes.Unauthenticated, err, info.FullMethod, "unauthenticated")
			return resp, status.Error(codes.Unauthenticated, msgUnauthenticated)
		}

		log.Ctx(ctx).Error().Err(err).Str("method", info.FullMethod).Msg("unhandled error")
		return resp, status.Error(codes.Internal, msgInternalServerError)
	}
}

// logStatus は gRPC コードに応じたレベルで cause をログする。
func logStatus(ctx context.Context, code codes.Code, cause error, method, msg string) {
	log.Ctx(ctx).WithLevel(levelForCode(code)).Err(cause).Str("method", method).Msg(msg)
}

// levelForCode は gRPC ステータスコードをログレベルに対応づける。
// 想定内のクライアントエラー（不正入力・未検出・権限なし等）は info、サーバ起因のみ
// warn / error にして、日常的な 4xx 相当でログを汚さない。
func levelForCode(c codes.Code) zerolog.Level {
	switch c {
	case codes.Internal, codes.Unknown, codes.DataLoss:
		return zerolog.ErrorLevel
	case codes.Unavailable, codes.DeadlineExceeded, codes.ResourceExhausted, codes.NotFound:
		// NotFound は「存在しない/権限外を存在しないように見せた」対象へのアクセスなので、
		// 気づけるよう warn にしておく。
		return zerolog.WarnLevel
	default:
		// InvalidArgument / AlreadyExists / PermissionDenied / Unauthenticated /
		// FailedPrecondition など想定内のクライアントエラー
		return zerolog.InfoLevel
	}
}
