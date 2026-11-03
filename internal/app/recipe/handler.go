package recipe

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/kodaikumatani/grpc-cqrs-go/internal/app/recipe/command"
	"github.com/kodaikumatani/grpc-cqrs-go/internal/app/recipe/entity"
	"github.com/kodaikumatani/grpc-cqrs-go/internal/app/recipe/query"
	"github.com/kodaikumatani/grpc-cqrs-go/internal/grpcerr"
	"github.com/kodaikumatani/grpc-cqrs-go/internal/identity"
	pb "github.com/kodaikumatani/grpc-cqrs-go/pkg/pb/recipe"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	msgUnauthenticated   = "unauthenticated"
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
	request := struct {
		Title       string `validate:"required"`
		Description string `validate:"required"`
	}{
		Title:       in.GetTitle(),
		Description: in.GetDescription(),
	}

	if err := validator.New().Struct(request); err != nil {
		return nil, grpcerr.WithStatus(err, codes.InvalidArgument, err.Error())
	}

	userID, err := identity.UserID(ctx)
	if err != nil {
		return nil, grpcerr.WithStatus(err, codes.Unauthenticated, msgUnauthenticated)
	}

	result, err := h.command.Create(ctx,
		userID,
		request.Title,
		request.Description)
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
	request := struct {
		ID          string `validate:"required"`
		Title       string `validate:"required"`
		Description string `validate:"required"`
	}{
		ID:          in.GetId(),
		Title:       in.GetTitle(),
		Description: in.GetDescription(),
	}

	if err := validator.New().Struct(request); err != nil {
		return nil, grpcerr.WithStatus(err, codes.InvalidArgument, err.Error())
	}

	recipeID, err := uuid.Parse(request.ID)
	if err != nil {
		return nil, grpcerr.WithStatus(err, codes.InvalidArgument, err.Error())
	}

	userID, err := identity.UserID(ctx)
	if err != nil {
		return nil, grpcerr.WithStatus(err, codes.Unauthenticated, msgUnauthenticated)
	}

	if err := h.command.Update(ctx, userID, recipeID, request.Title, request.Description); err != nil {
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
		return nil, grpcerr.WithStatus(err, codes.Unauthenticated, msgUnauthenticated)
	}

	result, err := h.query.GetRecipe(ctx, userID, id)
	if err != nil {
		if errors.Is(err, entity.ErrRecipeNotFound) {
			return nil, grpcerr.WithStatus(err, codes.NotFound, msgRecipeNotFound)
		}
		return nil, err
	}

	visibility, ok := domainVisibilityToPb[result.Visibility]
	if !ok {
		return nil, errors.New("visibility has no protobuf representation")
	}

	return &pb.GetRecipeResponse{
		Recipe: &pb.Recipe{
			Id:          result.ID,
			UserId:      result.User.ID,
			Title:       result.Title,
			Description: result.Description,
			Visibility:  visibility,
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

	visibility, ok := pbToDomainVisibility[in.GetVisibility()]
	if !ok {
		return nil, grpcerr.WithStatus(entity.ErrInvalidVisibility, codes.InvalidArgument, msgInvalidVisibility)
	}

	userID, err := identity.UserID(ctx)
	if err != nil {
		return nil, grpcerr.WithStatus(err, codes.Unauthenticated, msgUnauthenticated)
	}

	if err := h.command.UpdateVisibility(ctx, userID, id, visibility); err != nil {
		return nil, err
	}

	return &pb.ChangeVisibilityResponse{Success: true}, nil
}

var pbToDomainVisibility = map[pb.Visibility]entity.Visibility{
	pb.Visibility_VISIBILITY_PUBLIC:     entity.VisibilityPublic,
	pb.Visibility_VISIBILITY_PRIVATE:    entity.VisibilityPrivate,
	pb.Visibility_VISIBILITY_RESTRICTED: entity.VisibilityRestricted,
}

var domainVisibilityToPb = func() map[entity.Visibility]pb.Visibility {
	out := make(map[entity.Visibility]pb.Visibility, len(pbToDomainVisibility))

	for k, v := range pbToDomainVisibility {
		if dup, ok := out[v]; ok {
			panic(fmt.Sprintf("invert: duplicate value %v (%v, %v)", v, dup, k))
		}

		out[v] = k
	}

	return out
}()
