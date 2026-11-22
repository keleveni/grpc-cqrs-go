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

func (c Checker) CanEditRecipe(ctx context.Context, userID, recipeID string) error {
	return c.hasPermission(ctx, userID, ObjectRecipe, recipeID, PermEdit)
}

func (c Checker) CanShareRecipe(ctx context.Context, userID, recipeID string) error {
	return c.hasPermission(ctx, userID, ObjectRecipe, recipeID, PermShare)
}

// CanManageRecipe は userID がレシピの manage 権限（owner）を持つかを tuple で判定する。
func (c Checker) CanManageRecipe(ctx context.Context, userID, recipeID string) error {
	return c.hasPermission(ctx, userID, ObjectRecipe, recipeID, PermManage)
}

// EditableRecipeIDs は userID が edit 権限（editor / owner）を持つレシピの ID を返す（一覧の絞り込み用）。
func (c Checker) EditableRecipeIDs(ctx context.Context, userID string) ([]string, error) {
	return c.storage.ListObjectIDs(ctx, ObjectRecipe, userID, permissionRelations[ObjectRecipe][PermEdit])
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
