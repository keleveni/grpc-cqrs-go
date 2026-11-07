package recipe

import (
	"context"
	"errors"
	"uuid"

	"github.com/kodaikumatani/grpc-cqrs-go/internal/app/recipe/command"
	"github.com/kodaikumatani/grpc-cqrs-go/internal/app/recipe/entity"
	"github.com/kodaikumatani/grpc-cqrs-go/internal/app/recipe/query"
	"github.com/kodaikumatani/grpc-cqrs-go/internal/authz"
	"github.com/kodaikumatani/grpc-cqrs-go/internal/cursor"
	"github.com/kodaikumatani/grpc-cqrs-go/internal/grpcerr"
	"github.com/kodaikumatani/grpc-cqrs-go/internal/identity"
	pb "github.com/kodaikumatani/grpc-cqrs-go/pkg/pb/recipe/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	msgInvalidVisibility = "invalid visibility"
	msgRecipeNotFound    = "recipe not found"
	msgInvalidPageToken  = "invalid page token"
	msgInvalidRelation   = "invalid relation"
	msgAlreadyShared     = "recipe already shared with this user"
)

// defaultPageSize は ListRecipes で page_size が 0 のときの件数（上限は proto のルールで 100）。
const defaultPageSize = 20

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
		if errors.Is(err, entity.ErrRecipeNotFound) {
			return nil, grpcerr.WithStatus(err, codes.NotFound, msgRecipeNotFound)
		}
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
			UserId:      result.UserID,
			Title:       result.Title,
			Description: result.Description,
			Visibility:  visibilityToPb(result.Visibility),
			CreatedAt:   timestamppb.New(result.CreatedAt),
			UpdatedAt:   timestamppb.New(result.UpdatedAt),
		},
		User: &pb.User{
			Id:    result.UserID,
			Name:  result.UserName,
			Email: result.UserEmail,
		},
	}, nil
}

func (h *handler) ListRecipes(
	ctx context.Context,
	in *pb.ListRecipesRequest,
) (*pb.ListRecipesResponse, error) {
	after, err := cursor.Decode(in.GetPageToken())
	if err != nil {
		return nil, grpcerr.WithStatus(err, codes.InvalidArgument, msgInvalidPageToken)
	}

	pageSize := in.GetPageSize()
	if pageSize == 0 {
		pageSize = defaultPageSize
	}

	userID, err := identity.UserID(ctx)
	if err != nil {
		return nil, err
	}

	result, err := h.query.ListRecipes(ctx, userID, after, pageSize)
	if err != nil {
		if errors.Is(err, query.ErrInvalidCursor) {
			return nil, grpcerr.WithStatus(err, codes.InvalidArgument, msgInvalidPageToken)
		}
		return nil, err
	}

	recipes := make([]*pb.Recipe, 0, len(result.Recipes))
	for _, r := range result.Recipes {
		recipes = append(recipes, &pb.Recipe{
			Id:          r.ID,
			UserId:      r.UserID,
			Title:       r.Title,
			Description: r.Description,
			Visibility:  visibilityToPb(r.Visibility),
			CreatedAt:   timestamppb.New(r.CreatedAt),
			UpdatedAt:   timestamppb.New(r.UpdatedAt),
		})
	}

	return &pb.ListRecipesResponse{
		Recipes:       recipes,
		NextPageToken: cursor.Encode(result.Next),
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

	userID, err := identity.UserID(ctx)
	if err != nil {
		return nil, err
	}

	if err := h.command.ChangeVisibility(ctx, userID, id, visibilityFromPb(in.GetVisibility())); err != nil {
		switch {
		case errors.Is(err, entity.ErrInvalidVisibility):
			return nil, grpcerr.WithStatus(err, codes.InvalidArgument, msgInvalidVisibility)
		case errors.Is(err, entity.ErrRecipeNotFound):
			return nil, grpcerr.WithStatus(err, codes.NotFound, msgRecipeNotFound)
		default:
			return nil, err
		}
	}

	return &pb.ChangeVisibilityResponse{Success: true}, nil
}

func (h *handler) ShareRecipe(
	ctx context.Context,
	in *pb.ShareRecipeRequest,
) (*pb.ShareRecipeResponse, error) {
	id, err := uuid.Parse(in.GetId())
	if err != nil {
		return nil, grpcerr.WithStatus(err, codes.InvalidArgument, err.Error())
	}

	userID, err := identity.UserID(ctx)
	if err != nil {
		return nil, err
	}

	if err := h.command.ShareRecipe(ctx,
		userID,
		id,
		in.GetTargetUserId(),
		relationFromPb(in.GetRelation())); err != nil {
		switch {
		case errors.Is(err, authz.ErrUnknownRelation):
			return nil, grpcerr.WithStatus(err, codes.InvalidArgument, msgInvalidRelation)
		case errors.Is(err, entity.ErrRecipeNotFound):
			return nil, grpcerr.WithStatus(err, codes.NotFound, msgRecipeNotFound)
		case errors.Is(err, authz.ErrAlreadyExists):
			return nil, grpcerr.WithStatus(err, codes.AlreadyExists, msgAlreadyShared)
		default:
			return nil, err
		}
	}

	return &pb.ShareRecipeResponse{}, nil
}

// visibilityFromPb は pb の Visibility を、ユースケースに渡す公開範囲の文字列にする。
// UNSPECIFIED や未定義の値は空文字にする（ドメインの entity.NewVisibility が ErrInvalidVisibility にする）。
func visibilityFromPb(v pb.Visibility) string {
	switch v {
	case pb.Visibility_VISIBILITY_PUBLIC:
		return entity.VisibilityPublic.String()
	case pb.Visibility_VISIBILITY_PRIVATE:
		return entity.VisibilityPrivate.String()
	case pb.Visibility_VISIBILITY_RESTRICTED:
		return entity.VisibilityRestricted.String()
	default:
		return ""
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

// relationFromPb は pb の Relation を、ユースケースに渡す relation の文字列にする。
// UNSPECIFIED や未定義の値は空文字にする（authz.NewRelation が ErrUnknownRelation にする）。
func relationFromPb(r pb.Relation) string {
	switch r {
	case pb.Relation_RELATION_VIEWER:
		return authz.RelViewer.String()
	case pb.Relation_RELATION_EDITOR:
		return authz.RelEditor.String()
	case pb.Relation_RELATION_OWNER:
		return authz.RelOwner.String()
	default:
		return ""
	}
}
