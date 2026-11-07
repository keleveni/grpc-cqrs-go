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

	if err := u.refreshViewers(ctx, recipe); err != nil {
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
//   - entity.ErrInvalidVisibility: visibility が公開範囲として不正
//   - authz.ErrPermissionDenied: manage 権限がない（レシピが存在しない場合もこちらになる）
//   - entity.ErrRecipeNotFound: manage 権限はあるがレシピが存在しない
func (u *Command) ChangeVisibility(
	ctx context.Context,
	userID string,
	id uuid.UUID,
	visibility string,
) error {
	v, err := entity.NewVisibility(visibility)
	if err != nil {
		return err
	}

	if err := u.checker.CanManageRecipe(ctx, userID, id.String()); err != nil {
		return err
	}

	recipe, err := u.storage.Get(ctx, id)
	if err != nil {
		return err
	}

	recipe.ChangeVisibility(v)

	if err := u.storage.Update(ctx, recipe); err != nil {
		return err
	}

	// 公開範囲が変わると閲覧できるユーザーも変わる
	return u.refreshViewers(ctx, recipe)
}

// ShareRecipe はレシピを targetUserID に relation で共有する。共有できるのは owner だけ。
//
// 返しうるエラー:
//   - authz.ErrUnknownRelation: relation が relation として不正
//   - authz.ErrPermissionDenied: 共有権限がない（レシピが存在しない場合もこちらになる）
//   - entity.ErrRecipeNotFound: 共有権限はあるがレシピが存在しない
//   - authz.ErrAlreadyExists: 同じ relation で既に共有済み
func (u *Command) ShareRecipe(
	ctx context.Context,
	userID string,
	id uuid.UUID,
	targetUserID string,
	relation string,
) error {
	rel, err := authz.NewRelation(relation)
	if err != nil {
		return err
	}

	if err := u.checker.CanShareRecipe(ctx, userID, id.String()); err != nil {
		return err
	}

	recipe, err := u.storage.Get(ctx, id)
	if err != nil {
		return err
	}

	tp := authz.NewTuple(
		uuid.New(),
		authz.ObjectRecipe,
		recipe.ID().String(),
		rel,
		targetUserID,
	)

	if err := u.tuple.CreateTuple(ctx, tp); err != nil {
		return err
	}

	// restricted なら共有相手が閲覧できるようになる
	return u.refreshViewers(ctx, recipe)
}

// refreshViewers は読み取りモデル（recipe_viewers: 誰がレシピを閲覧できるか）を、
// 今の公開範囲と tuple から計算し直して保存する（projection）。
//
// 閲覧者は公開範囲と tuple から決まるので、どちらかを変える書き込み
// （作成・公開範囲の変更・共有）は必ずこれを呼ぶ。読み取り側はこの結果を引くだけで、閲覧のルールを持たない。
func (u *Command) refreshViewers(ctx context.Context, recipe *entity.Recipe) error {
	viewers, err := recipe.Viewers(ctx, u.checker)
	if err != nil {
		return err
	}

	return u.storage.ReplaceViewers(ctx, recipe.ID(), viewers)
}
