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

// CanEdit は公開範囲に応じて、userID が recipeID のレシピを編集できるかを判定する。
//   - private:            manage 権限を持つユーザーだけが編集できる（閲覧と同じく閉じる）
//   - public / restricted: edit 権限を持つユーザーが編集できる
//
// どの relation（owner 等）がどの権限を与えるかは authz が決める。entity は権限だけを問い合わせる。
//
// 権限がなければ Relations のエラーをそのまま返し、未知の公開範囲は ErrInvalidVisibility を返す。
func (v Visibility) CanEdit(ctx context.Context, rels Relations, userID, recipeID string) error {
	switch v {
	case VisibilityPrivate:
		return rels.CanManageRecipe(ctx, userID, recipeID)
	case VisibilityPublic, VisibilityRestricted:
		return rels.CanEditRecipe(ctx, userID, recipeID)
	default:
		return ErrInvalidVisibility
	}
}

// Everyone は「全ユーザー」を表す閲覧者。public のレシピの閲覧者として読み取りモデルに書く。
// ユーザー ID（IdP の sub）が "*" になることはないので、実在のユーザーと衝突しない。
const Everyone = "*"

// Viewers は公開範囲に応じて、recipeID のレシピを閲覧できるユーザーを返す。
//   - public:     全員（Everyone）
//   - private:    manage 権限を持つユーザー（編集と同じく閉じる）
//   - restricted: view 権限を持つユーザー
//
// 結果は読み取りモデル（recipe_viewers）に書き、読み取り側はそれを引くだけにする。
// 閲覧のルールはここだけが持つ。未知の公開範囲は ErrInvalidVisibility を返す。
func (v Visibility) Viewers(ctx context.Context, rels Relations, recipeID string) ([]string, error) {
	switch v {
	case VisibilityPublic:
		return []string{Everyone}, nil
	case VisibilityPrivate:
		return rels.UsersWithManageRecipe(ctx, recipeID)
	case VisibilityRestricted:
		return rels.UsersWithViewRecipe(ctx, recipeID)
	default:
		return nil, ErrInvalidVisibility
	}
}

// Relations は閲覧・編集の判定に必要な、ユーザーがレシピに対して持つ権限の問い合わせ。
// authz.Checker がこれを満たす（entity は authz に依存しない）。
type Relations interface {
	// CanEditRecipe は userID がレシピの edit 権限（editor / owner）を持つかを判定する。
	CanEditRecipe(ctx context.Context, userID, recipeID string) error
	// CanManageRecipe は userID がレシピの manage 権限（owner）を持つかを判定する。
	CanManageRecipe(ctx context.Context, userID, recipeID string) error
	// UsersWithManageRecipe はレシピの manage 権限（owner）を持つユーザーを返す。
	UsersWithManageRecipe(ctx context.Context, recipeID string) ([]string, error)
	// UsersWithViewRecipe はレシピの view 権限（viewer / editor / owner）を持つユーザーを返す。
	UsersWithViewRecipe(ctx context.Context, recipeID string) ([]string, error)
}
