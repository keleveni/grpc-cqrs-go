package authz

import (
	"context"

	"github.com/kodaikumatani/grpc-cqrs-go/internal/app/recipe/entity"
)

type Checker struct {
	storage Storage
}

func NewChecker(storage Storage) Checker {
	return Checker{storage: storage}
}

// CanViewRecipe はレシピの公開範囲に応じて閲覧可否を判定する。
//   - public:     誰でも可
//   - private:    owner のみ
//   - restricted: view 権限（viewer / editor / owner）を持つユーザー
func (c Checker) CanViewRecipe(
	ctx context.Context,
	userID, recipeID string,
	visibility entity.Visibility,
) error {
	switch visibility {
	case entity.VisibilityPublic:
		return nil
	case entity.VisibilityPrivate:
		return c.IsRecipeOwner(ctx, userID, recipeID)
	case entity.VisibilityRestricted:
		return c.hasPermission(ctx, userID, ObjectRecipe, recipeID, PermView)
	default:
		return ErrPermissionDenied
	}
}

func (c Checker) CanEditRecipe(ctx context.Context, userID, recipeID string) error {
	return c.hasPermission(ctx, userID, ObjectRecipe, recipeID, PermEdit)
}

func (c Checker) CanShareRecipe(ctx context.Context, userID, recipeID string) error {
	return c.hasPermission(ctx, userID, ObjectRecipe, recipeID, PermShare)
}

func (c Checker) IsRecipeOwner(ctx context.Context, userID, recipeID string) error {
	return c.hasRelation(ctx, userID, ObjectRecipe, recipeID, RelOwner)
}

func (c Checker) hasPermission(
	ctx context.Context,
	userID string,
	objectType ObjectType,
	objectID string,
	perm Permission,
) error {
	tuples, err := c.storage.ListRelations(
		ctx,
		objectType,
		objectID,
		userID,
	)
	if err != nil {
		return err
	}

	for _, relation := range permissionRelations[objectType][perm] {
		for _, tuple := range tuples {
			if relation == tuple.Relation() {
				return nil
			}
		}
	}

	return ErrPermissionDenied
}

func (c Checker) hasRelation(
	ctx context.Context,
	userID string,
	objectType ObjectType,
	objectID string,
	relation Relation,
) error {
	tuples, err := c.storage.ListRelations(
		ctx,
		objectType,
		objectID,
		userID,
	)
	if err != nil {
		return err
	}

	for _, tuple := range tuples {
		if relation == tuple.Relation() {
			return nil
		}
	}

	return ErrPermissionDenied
}
