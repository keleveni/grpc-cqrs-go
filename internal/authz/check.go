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

// UsersWithManageRecipe は recipeID の manage 権限（owner）を持つユーザーを返す（閲覧者の計算用）。
func (c Checker) UsersWithManageRecipe(ctx context.Context, recipeID string) ([]string, error) {
	return c.usersWith(ctx, ObjectRecipe, recipeID, PermManage)
}

// UsersWithViewRecipe は recipeID の view 権限（viewer / editor / owner）を持つユーザーを返す（閲覧者の計算用）。
func (c Checker) UsersWithViewRecipe(ctx context.Context, recipeID string) ([]string, error) {
	return c.usersWith(ctx, ObjectRecipe, recipeID, PermView)
}

func (c Checker) usersWith(
	ctx context.Context,
	objectType ObjectType,
	objectID string,
	perm Permission,
) ([]string, error) {
	return c.storage.ListUserIDs(ctx, objectType, objectID, permissionRelations[objectType][perm])
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
