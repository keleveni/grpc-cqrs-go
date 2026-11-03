package interceptor

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/kodaikumatani/grpc-cqrs-go/internal/authz"
	"github.com/kodaikumatani/grpc-cqrs-go/internal/grpcerr"
	"github.com/kodaikumatani/grpc-cqrs-go/internal/identity"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestLevelForCode(t *testing.T) {
	tests := []struct {
		code codes.Code
		want zerolog.Level
	}{
		// 想定内のクライアントエラーは info(warn にしない)
		{codes.PermissionDenied, zerolog.InfoLevel},
		{codes.InvalidArgument, zerolog.InfoLevel},
		{codes.AlreadyExists, zerolog.InfoLevel},
		{codes.Unauthenticated, zerolog.InfoLevel},
		// サーバ起因は error
		{codes.Internal, zerolog.ErrorLevel},
		{codes.Unknown, zerolog.ErrorLevel},
		// NotFound と一時的な障害は warn
		{codes.NotFound, zerolog.WarnLevel},
		{codes.Unavailable, zerolog.WarnLevel},
		{codes.DeadlineExceeded, zerolog.WarnLevel},
	}

	for _, tt := range tests {
		if got := levelForCode(tt.code); got != tt.want {
			t.Errorf("levelForCode(%v) = %v, want %v", tt.code, got, tt.want)
		}
	}
}

func TestErrorHandlingUnaryInterceptor(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode codes.Code
		wantMsg  string
	}{
		{
			name:     "handler が付けた status はそのまま返す",
			err:      grpcerr.WithStatus(errors.New("cause"), codes.NotFound, "recipe not found"),
			wantCode: codes.NotFound,
			wantMsg:  "recipe not found",
		},
		{
			name:     "authz の拒否は PermissionDenied",
			err:      fmt.Errorf("wrap: %w", authz.ErrPermissionDenied),
			wantCode: codes.PermissionDenied,
			wantMsg:  msgPermissionDenied,
		},
		{
			name:     "認証済みユーザーがいなければ Unauthenticated",
			err:      fmt.Errorf("wrap: %w", identity.ErrUnauthenticated),
			wantCode: codes.Unauthenticated,
			wantMsg:  msgUnauthenticated,
		},
		{
			name:     "未変換の生エラーは Internal で中身を隠す",
			err:      errors.New("db connection refused"),
			wantCode: codes.Internal,
			wantMsg:  msgInternalServerError,
		},
	}

	icp := ErrorHandlingUnaryInterceptor()
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := icp(context.Background(), nil, info, func(context.Context, any) (any, error) {
				return nil, tt.err
			})

			st, ok := status.FromError(err)
			if !ok {
				t.Fatalf("want status error, got %v", err)
			}
			if st.Code() != tt.wantCode || st.Message() != tt.wantMsg {
				t.Fatalf("got (%v, %q), want (%v, %q)", st.Code(), st.Message(), tt.wantCode, tt.wantMsg)
			}
		})
	}
}
