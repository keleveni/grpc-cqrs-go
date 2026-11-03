package entity

import (
	"context"

	"github.com/cockroachdb/errors"
)

var ErrInvalidVisibility = errors.New("invalid visibility")

type Visibility struct {
	value string
}

var (
	VisibilityPublic     = Visibility{"public"}
	VisibilityPrivate    = Visibility{"private"}
	VisibilityRestricted = Visibility{"restricted"}
)

func NewVisibility(s string) (Visibility, error) {
	switch s {
	case VisibilityPublic.value, VisibilityPrivate.value, VisibilityRestricted.value:
		return Visibility{s}, nil
	default:
		return Visibility{}, ErrInvalidVisibility
	}
}

func (v Visibility) String() string { return v.value }

// CanView は公開範囲に応じて、userID が recipeID のレシピを閲覧できるかを判定する。
//   - public:     誰でも閲覧できる
//   - private:    owner だけが閲覧できる
//   - restricted: view 権限を持つユーザーだけが閲覧できる
//
// owner は tuple で管理していて複数いてよいので、作成者との比較ではなく Relations に問い合わせる。
// 権限がなければ Relations のエラーをそのまま返し、未知の公開範囲は ErrInvalidVisibility を返す。
func (v Visibility) CanView(ctx context.Context, rels Relations, userID, recipeID string) error {
	switch v {
	case VisibilityPublic:
		return nil
	case VisibilityPrivate:
		return rels.IsRecipeOwner(ctx, userID, recipeID)
	case VisibilityRestricted:
		return rels.CanViewRecipe(ctx, userID, recipeID)
	default:
		return ErrInvalidVisibility
	}
}

// CanEdit は公開範囲に応じて、userID が recipeID のレシピを編集できるかを判定する。
//   - private:            owner だけが編集できる（閲覧と同じく owner 以外には閉じる）
//   - public / restricted: 編集権限（editor / owner）を持つユーザーが編集できる
//
// 権限がなければ Relations のエラーをそのまま返し、未知の公開範囲は ErrInvalidVisibility を返す。
func (v Visibility) CanEdit(ctx context.Context, rels Relations, userID, recipeID string) error {
	switch v {
	case VisibilityPrivate:
		return rels.IsRecipeOwner(ctx, userID, recipeID)
	case VisibilityPublic, VisibilityRestricted:
		return rels.CanEditRecipe(ctx, userID, recipeID)
	default:
		return ErrInvalidVisibility
	}
}

// Relations は閲覧・編集権限の判定に必要な、ユーザーとレシピの関係の問い合わせ。
// authz.Checker がこれを満たす（entity は authz に依存しない）。
type Relations interface {
	// IsRecipeOwner は userID がレシピの owner かを判定する。
	IsRecipeOwner(ctx context.Context, userID, recipeID string) error
	// CanViewRecipe は userID がレシピの view 権限（viewer / editor / owner）を持つかを判定する。
	CanViewRecipe(ctx context.Context, userID, recipeID string) error
	// CanEditRecipe は userID がレシピの edit 権限（editor / owner）を持つかを判定する。
	CanEditRecipe(ctx context.Context, userID, recipeID string) error
}
