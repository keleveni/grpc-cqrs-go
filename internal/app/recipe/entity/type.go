package entity

import (
	"context"
	"uuid"

	"github.com/cockroachdb/errors"
)

var (
	ErrRecipeNotFound    = errors.New("recipe not found")
	ErrInvalidVisibility = errors.New("invalid visibility")
)

type Recipe struct {
	id          uuid.UUID
	userID      string
	title       string
	description string
	visibility  Visibility
}

func (r *Recipe) ID() uuid.UUID          { return r.id }
func (r *Recipe) UserID() string         { return r.userID }
func (r *Recipe) Title() string          { return r.title }
func (r *Recipe) Description() string    { return r.description }
func (r *Recipe) Visibility() Visibility { return r.visibility }

// NewRecipe は業務フィールドを受け取ってレシピを構築する。
// timestamps は永続化層(DB の default / now())が管理するため domain は持たない。
func NewRecipe(
	id uuid.UUID,
	userID string,
	title, description string,
	visibility Visibility,
) *Recipe {
	return &Recipe{
		id:          id,
		userID:      userID,
		title:       title,
		description: description,
		visibility:  visibility,
	}
}

// Edit はレシピの内容（タイトルと説明）を編集する。
func (r *Recipe) Edit(title, description string) {
	r.title = title
	r.description = description
}

func (r *Recipe) ChangeVisibility(v Visibility) {
	r.visibility = v
}

// CanBeViewedBy は公開範囲に応じて、userID がこのレシピを閲覧できるかを判定する。
//   - public:     誰でも閲覧できる
//   - private:    owner だけが閲覧できる
//   - restricted: view 権限を持つユーザーだけが閲覧できる
//
// owner は tuple で管理していて複数いてよいので、作成者（userID）との比較ではなく
// RelationChecker に問い合わせる。権限がなければ RelationChecker のエラーをそのまま返し、
// 未知の公開範囲は ErrInvalidVisibility を返す。
func (r *Recipe) CanBeViewedBy(ctx context.Context, rc RelationChecker, userID string) error {
	switch r.visibility {
	case VisibilityPublic:
		return nil
	case VisibilityPrivate:
		return rc.IsRecipeOwner(ctx, userID, r.id.String())
	case VisibilityRestricted:
		return rc.CanViewRecipe(ctx, userID, r.id.String())
	default:
		return ErrInvalidVisibility
	}
}

// CanBeEditedBy は公開範囲に応じて、userID がこのレシピを編集できるかを判定する。
//   - private:            owner だけが編集できる（閲覧と同じく owner 以外には閉じる）
//   - public / restricted: 編集権限（editor / owner）を持つユーザーが編集できる
//
// 権限がなければ RelationChecker のエラーをそのまま返し、未知の公開範囲は ErrInvalidVisibility を返す。
func (r *Recipe) CanBeEditedBy(ctx context.Context, rc RelationChecker, userID string) error {
	switch r.visibility {
	case VisibilityPrivate:
		return rc.IsRecipeOwner(ctx, userID, r.id.String())
	case VisibilityPublic, VisibilityRestricted:
		return rc.CanEditRecipe(ctx, userID, r.id.String())
	default:
		return ErrInvalidVisibility
	}
}

// RelationChecker は閲覧・編集権限の判定に必要な、ユーザーとレシピの関係の問い合わせ。
// authz.Checker がこれを満たす（entity は authz に依存しない）。
type RelationChecker interface {
	// IsRecipeOwner は userID がレシピの owner かを判定する。
	IsRecipeOwner(ctx context.Context, userID, recipeID string) error
	// CanViewRecipe は userID がレシピの view 権限（viewer / editor / owner）を持つかを判定する。
	CanViewRecipe(ctx context.Context, userID, recipeID string) error
	// CanEditRecipe は userID がレシピの edit 権限（editor / owner）を持つかを判定する。
	CanEditRecipe(ctx context.Context, userID, recipeID string) error
}

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
