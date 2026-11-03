package entity

import (
	"context"
)

// RelationChecker は閲覧権限の判定に必要な、ユーザーとレシピの関係の問い合わせ。
// authz.Checker がこれを満たす（entity は authz に依存しない）。
type RelationChecker interface {
	// IsRecipeOwner は userID がレシピの owner かを判定する。
	IsRecipeOwner(ctx context.Context, userID, recipeID string) error
	// CanViewRecipe は userID がレシピの view 権限（viewer / editor / owner）を持つかを判定する。
	CanViewRecipe(ctx context.Context, userID, recipeID string) error
}

// CanView は公開範囲に応じて、userID がレシピを閲覧できるかを判定する domain service。
//   - public:     誰でも閲覧できる
//   - private:    owner だけが閲覧できる
//   - restricted: view 権限を持つユーザーだけが閲覧できる
//
// 権限がなければ RelationChecker が返すエラーをそのまま返す。未知の公開範囲は ErrInvalidVisibility。
func CanView(ctx context.Context, rc RelationChecker, userID, recipeID string, v Visibility) error {
	switch v {
	case VisibilityPublic:
		return nil
	case VisibilityPrivate:
		return rc.IsRecipeOwner(ctx, userID, recipeID)
	case VisibilityRestricted:
		return rc.CanViewRecipe(ctx, userID, recipeID)
	default:
		return ErrInvalidVisibility
	}
}
