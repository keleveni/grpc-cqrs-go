package entity

import (
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
