package share

import (
	"context"
	"errors"

	"github.com/kodaikumatani/grpc-cqrs-go/internal/authz"
	"github.com/kodaikumatani/grpc-cqrs-go/internal/grpcerr"
	"github.com/kodaikumatani/grpc-cqrs-go/internal/identity"
	pb "github.com/kodaikumatani/grpc-cqrs-go/pkg/pb/share/v1"
	"google.golang.org/grpc/codes"
)

const (
	msgAlreadyShared   = "recipe already shared with this user"
	msgInvalidRelation = "invalid relation"
)

type handler struct {
	pb.UnimplementedShareServiceServer
	command *Command
}

func NewHandler(command *Command) pb.ShareServiceServer {
	return &handler{command: command}
}

func (h *handler) ShareRecipe(
	ctx context.Context,
	in *pb.ShareRecipeRequest,
) (*pb.ShareRecipeResponse, error) {
	relation, err := authz.NewRelation(in.GetRelation())
	if err != nil {
		return nil, grpcerr.WithStatus(err, codes.InvalidArgument, msgInvalidRelation)
	}

	userID, err := identity.UserID(ctx)
	if err != nil {
		return nil, err
	}

	if err := h.command.ShareRecipe(ctx,
		userID,
		in.GetRecipeId(),
		in.GetTargetUserId(),
		relation); err != nil {
		if errors.Is(err, authz.ErrAlreadyExists) {
			return nil, grpcerr.WithStatus(err, codes.AlreadyExists, msgAlreadyShared)
		}
		return nil, err
	}

	return &pb.ShareRecipeResponse{}, nil
}
