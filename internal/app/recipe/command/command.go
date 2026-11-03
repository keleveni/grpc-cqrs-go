package command

import (
	"context"
	"uuid"

	"github.com/kodaikumatani/grpc-cqrs-go/internal/app/recipe/entity"
	"github.com/kodaikumatani/grpc-cqrs-go/internal/authz"
)

type Command struct {
	storage Storage
	tuple   authz.Storage
	checker authz.Checker
}

func NewCommand(
	storage Storage,
	tuple authz.Storage,
	checker authz.Checker,
) *Command {
	return &Command{
		storage: storage,
		tuple:   tuple,
		checker: checker,
	}
}

// CreateRecipe は userID を owner とする private なレシピを作成する。
//
// ドメインのエラーは返さない（永続化の失敗は想定外の内部エラー）。
func (u *Command) CreateRecipe(
	ctx context.Context,
	userID string,
	title,
	description string,
) (*entity.Recipe, error) {
	recipe := entity.NewRecipe(
		uuid.NewV7(),
		userID,
		title,
		description,
		entity.VisibilityPrivate,
	)

	tp := authz.NewTuple(
		uuid.New(),
		authz.ObjectRecipe,
		recipe.ID().String(),
		authz.RelOwner,
		userID,
	)

	if err := u.storage.Create(ctx, recipe); err != nil {
		return nil, err
	}

	if err := u.tuple.CreateTuple(ctx, tp); err != nil {
		return nil, err
	}

	return recipe, nil
}

// UpdateRecipe はレシピの内容（タイトルと説明）を編集する。
// 編集できるのは、private なら owner だけ、それ以外は editor 以上（Recipe.CanBeEditedBy）。
//
// 返しうるエラー:
//   - entity.ErrRecipeNotFound: レシピが存在しない
//   - authz.ErrPermissionDenied: 編集権限がない
func (u *Command) UpdateRecipe(
	ctx context.Context,
	userID string,
	id uuid.UUID,
	title, description string,
) error {
	recipe, err := u.storage.Get(ctx, id)
	if err != nil {
		return err
	}

	if err := recipe.CanBeEditedBy(ctx, u.checker, userID); err != nil {
		return err
	}

	recipe.Edit(title, description)

	if err := u.storage.Update(ctx, recipe); err != nil {
		return err
	}

	return nil
}

// ChangeVisibility はレシピの公開範囲を変更する。manage 権限（owner）を持つユーザーだけが変更できる。
//
// 返しうるエラー:
//   - authz.ErrPermissionDenied: manage 権限がない（レシピが存在しない場合もこちらになる）
//   - entity.ErrRecipeNotFound: manage 権限はあるがレシピが存在しない
func (u *Command) ChangeVisibility(
	ctx context.Context,
	userID string,
	id uuid.UUID,
	visibility entity.Visibility,
) error {
	if err := u.checker.CanManageRecipe(ctx, userID, id.String()); err != nil {
		return err
	}

	recipe, err := u.storage.Get(ctx, id)
	if err != nil {
		return err
	}

	recipe.ChangeVisibility(visibility)

	return u.storage.Update(ctx, recipe)
}
