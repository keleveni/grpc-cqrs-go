package recipe

import (
	"context"
	"errors"
	"uuid"

	"github.com/kodaikumatani/grpc-cqrs-go/internal/app/recipe/command"
	"github.com/kodaikumatani/grpc-cqrs-go/internal/app/recipe/entity"
	"github.com/kodaikumatani/grpc-cqrs-go/internal/app/recipe/query"
	"github.com/kodaikumatani/grpc-cqrs-go/internal/grpcerr"
	"github.com/kodaikumatani/grpc-cqrs-go/internal/identity"
	pb "github.com/kodaikumatani/grpc-cqrs-go/pkg/pb/recipe/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	msgInvalidVisibility = "invalid visibility"
	msgRecipeNotFound    = "recipe not found"
)

type handler struct {
	pb.UnimplementedRecipeServiceServer
	command *command.Command
	query   *query.Query
}

func NewHandler(
	command *command.Command,
	query *query.Query,
) pb.RecipeServiceServer {
	return &handler{
		command: command,
		query:   query,
	}
}

func (h *handler) CreateRecipe(
	ctx context.Context,
	in *pb.CreateRecipeRequest,
) (*pb.CreateRecipeResponse, error) {
	userID, err := identity.UserID(ctx)
	if err != nil {
		return nil, err
	}

	result, err := h.command.CreateRecipe(ctx,
		userID,
		in.GetTitle(),
		in.GetDescription())
	if err != nil {
		return nil, err
	}

	return &pb.CreateRecipeResponse{
		RecipeId: result.ID().String(),
	}, nil
}

func (h *handler) UpdateRecipe(
	ctx context.Context,
	in *pb.UpdateRecipeRequest,
) (*pb.UpdateRecipeResponse, error) {
	recipeID, err := uuid.Parse(in.GetId())
	if err != nil {
		return nil, grpcerr.WithStatus(err, codes.InvalidArgument, err.Error())
	}

	userID, err := identity.UserID(ctx)
	if err != nil {
		return nil, err
	}

	if err := h.command.UpdateRecipe(ctx, userID, recipeID, in.GetTitle(), in.GetDescription()); err != nil {
		return nil, err
	}

	return &pb.UpdateRecipeResponse{Success: true}, nil
}

func (h *handler) GetRecipe(
	ctx context.Context,
	in *pb.GetRecipeRequest,
) (*pb.GetRecipeResponse, error) {
	id, err := uuid.Parse(in.GetId())
	if err != nil {
		return nil, grpcerr.WithStatus(err, codes.InvalidArgument, err.Error())
	}

	userID, err := identity.UserID(ctx)
	if err != nil {
		return nil, err
	}

	result, err := h.query.GetRecipe(ctx, userID, id)
	if err != nil {
		if errors.Is(err, entity.ErrRecipeNotFound) {
			return nil, grpcerr.WithStatus(err, codes.NotFound, msgRecipeNotFound)
		}
		return nil, err
	}

	return &pb.GetRecipeResponse{
		Recipe: &pb.Recipe{
			Id:          result.ID,
			UserId:      result.User.ID,
			Title:       result.Title,
			Description: result.Description,
			Visibility:  visibilityToPb(result.Visibility),
			CreatedAt:   timestamppb.New(result.CreatedAt),
			UpdatedAt:   timestamppb.New(result.UpdatedAt),
		},
		User: &pb.User{
			Id:    result.User.ID,
			Name:  result.User.Name,
			Email: result.User.Email,
		},
	}, nil
}

func (h *handler) ChangeVisibility(
	ctx context.Context,
	in *pb.ChangeVisibilityRequest,
) (*pb.ChangeVisibilityResponse, error) {
	id, err := uuid.Parse(in.GetId())
	if err != nil {
		return nil, grpcerr.WithStatus(err, codes.InvalidArgument, err.Error())
	}

	visibility, err := visibilityFromPb(in.GetVisibility())
	if err != nil {
		return nil, grpcerr.WithStatus(err, codes.InvalidArgument, msgInvalidVisibility)
	}

	userID, err := identity.UserID(ctx)
	if err != nil {
		return nil, err
	}

	if err := h.command.ChangeVisibility(ctx, userID, id, visibility); err != nil {
		return nil, err
	}

	return &pb.ChangeVisibilityResponse{Success: true}, nil
}

// visibilityFromPb は pb の Visibility をドメインの Visibility に変換する。
// UNSPECIFIED や未定義の値は ErrInvalidVisibility を返す。
func visibilityFromPb(v pb.Visibility) (entity.Visibility, error) {
	switch v {
	case pb.Visibility_VISIBILITY_PUBLIC:
		return entity.VisibilityPublic, nil
	case pb.Visibility_VISIBILITY_PRIVATE:
		return entity.VisibilityPrivate, nil
	case pb.Visibility_VISIBILITY_RESTRICTED:
		return entity.VisibilityRestricted, nil
	default:
		return entity.Visibility{}, entity.ErrInvalidVisibility
	}
}

// visibilityToPb はドメインの Visibility を pb の Visibility に変換する。
// ドメインの値は閉じた集合なので失敗しない（未知の値は UNSPECIFIED）。
func visibilityToPb(v entity.Visibility) pb.Visibility {
	switch v {
	case entity.VisibilityPublic:
		return pb.Visibility_VISIBILITY_PUBLIC
	case entity.VisibilityPrivate:
		return pb.Visibility_VISIBILITY_PRIVATE
	case entity.VisibilityRestricted:
		return pb.Visibility_VISIBILITY_RESTRICTED
	default:
		return pb.Visibility_VISIBILITY_UNSPECIFIED
	}
}
