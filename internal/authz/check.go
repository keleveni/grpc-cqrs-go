package authz

import (
	"context"
)

type Checker struct {
	storage Storage
}

func NewChecker(storage Storage) Checker {
	return Checker{storage: storage}
}

// CanViewRecipe は userID がレシピの view 権限（viewer / editor / owner）を持つかを tuple で判定する。
// 公開範囲（Visibility）に応じた判定は recipe の domain service（entity.Recipe.CanBeViewedBy）が行う。
func (c Checker) CanViewRecipe(ctx context.Context, userID, recipeID string) error {
	return c.hasPermission(ctx, userID, ObjectRecipe, recipeID, PermView)
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
