package authz

import (
	"context"
	"slices"
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

// PermissionSet はユーザーが 1 つの対象に対して持つ権限の集合。
type PermissionSet map[Permission]bool

// Has は p を持つかを返す。
func (s PermissionSet) Has(p Permission) bool { return s[p] }

// RecipePermissions は userID が recipeID に対して持つ権限を、tuple を 1 回だけ検索して返す。
func (c Checker) RecipePermissions(ctx context.Context, userID, recipeID string) (PermissionSet, error) {
	tuples, err := c.storage.ListRelations(ctx, ObjectRecipe, recipeID, userID)
	if err != nil {
		return nil, err
	}

	set := PermissionSet{}
	for perm, relations := range permissionRelations[ObjectRecipe] {
		for _, tuple := range tuples {
			if slices.Contains(relations, tuple.Relation()) {
				set[perm] = true
				break
			}
		}
	}

	return set, nil
}

// RecipeIDsWith は userID が perm を持つレシピの ID を返す（一覧の絞り込み用）。
func (c Checker) RecipeIDsWith(ctx context.Context, userID string, perm Permission) ([]string, error) {
	return c.storage.ListObjectIDs(ctx, ObjectRecipe, userID, permissionRelations[ObjectRecipe][perm])
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
