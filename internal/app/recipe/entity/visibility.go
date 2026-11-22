package entity

import (
	"github.com/cockroachdb/errors"
)

var ErrInvalidVisibility = errors.New("invalid visibility")

// Visibility はレシピの公開範囲。閲覧だけに効く（編集は edit 権限で決まり、公開範囲は関係ない）。
//   - public:  誰でも閲覧できる
//   - private: edit 権限（owner / editor）を持つユーザーだけが閲覧できる（非公開の共同作業）
//
// 閲覧の判定は SQL（GetViewableRecipe / ListViewableRecipes）で行う。
type Visibility struct {
	value string
}

var (
	VisibilityPublic  = Visibility{"public"}
	VisibilityPrivate = Visibility{"private"}
)

func NewVisibility(s string) (Visibility, error) {
	switch s {
	case VisibilityPublic.value, VisibilityPrivate.value:
		return Visibility{s}, nil
	default:
		return Visibility{}, ErrInvalidVisibility
	}
}

func (v Visibility) String() string { return v.value }
