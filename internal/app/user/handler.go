package user

import (
	"context"
	"errors"

	"github.com/kodaikumatani/grpc-cqrs-go/internal/app/user/command"
	"github.com/kodaikumatani/grpc-cqrs-go/internal/app/user/entity"
	"github.com/kodaikumatani/grpc-cqrs-go/internal/grpcerr"
	"github.com/kodaikumatani/grpc-cqrs-go/internal/identity"
	pb "github.com/kodaikumatani/grpc-cqrs-go/pkg/pb/user/v1"
	"google.golang.org/grpc/codes"
)

const (
	msgUnauthenticated   = "unauthenticated"
	msgUserAlreadyExists = "user already exists"
)

type handler struct {
	pb.UnimplementedUserServiceServer
	command *command.Command
}

func NewHandler(command *command.Command) pb.UserServiceServer {
	return &handler{command: command}
}

func (h *handler) CreateUser(
	ctx context.Context,
	in *pb.CreateUserRequest,
) (*pb.CreateUserResponse, error) {
	id, err := identity.UserID(ctx)
	if err != nil {
		return nil, grpcerr.WithStatus(err, codes.Unauthenticated, msgUnauthenticated)
	}

	result, err := h.command.Create(ctx, id, in.GetName(), in.GetEmail())
	if err != nil {
		if errors.Is(err, entity.ErrAlreadyExists) {
			return nil, grpcerr.WithStatus(err, codes.AlreadyExists, msgUserAlreadyExists)
		}
		return nil, err
	}

	return &pb.CreateUserResponse{
		UserId: result.ID(),
	}, nil
}
