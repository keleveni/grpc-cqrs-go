package share

import (
	"context"
	"uuid"

	"github.com/kodaikumatani/grpc-cqrs-go/internal/authz"
)

// RecipeViewers は共有の変更を、レシピの閲覧者（読み取りモデル）に反映する。
// recipe の command.ViewerProjector がこれを満たす（share は recipe に依存しない）。
type RecipeViewers interface {
	RefreshViewers(ctx context.Context, recipeID string) error
}

type Command struct {
	storage authz.Storage
	checker authz.Checker
	viewers RecipeViewers
}

func NewCommand(
	storage authz.Storage,
	checker authz.Checker,
	viewers RecipeViewers,
) *Command {
	return &Command{
		storage: storage,
		checker: checker,
		viewers: viewers,
	}
}

// ShareRecipe はレシピを targetUserID に relation で共有する。共有できるのは owner だけ。
//
// 返しうるエラー:
//   - authz.ErrPermissionDenied: 共有権限がない
//   - authz.ErrAlreadyExists: 同じ relation で既に共有済み
func (u *Command) ShareRecipe(
	ctx context.Context,
	userID,
	recipeID,
	targetUserID string,
	relation authz.Relation,
) error {
	if err := u.checker.CanShareRecipe(ctx, userID, recipeID); err != nil {
		return err
	}

	tuple := authz.NewTuple(
		uuid.New(),
		authz.ObjectRecipe,
		recipeID,
		relation,
		targetUserID,
	)

	if err := u.storage.CreateTuple(ctx, tuple); err != nil {
		return err
	}

	// restricted なら共有相手が閲覧できるようになる（public / private では閲覧者は変わらない）
	return u.viewers.RefreshViewers(ctx, recipeID)
}
