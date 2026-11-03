package entity

import (
	"context"
	"uuid"

	"github.com/cockroachdb/errors"
)

var ErrRecipeNotFound = errors.New("recipe not found")

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

// CanBeEditedBy は userID がこのレシピを編集できるかを判定する（Visibility.CanEdit に委ねる）。
func (r *Recipe) CanBeEditedBy(ctx context.Context, rc RelationChecker, userID string) error {
	return r.visibility.CanEdit(ctx, rc, userID, r.id.String())
}
