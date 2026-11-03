package share

import (
	"context"
	"uuid"

	"github.com/kodaikumatani/grpc-cqrs-go/internal/authz"
)

type Command struct {
	storage authz.Storage
	checker authz.Checker
}

func NewCommand(
	storage authz.Storage,
	checker authz.Checker,
) *Command {
	return &Command{
		storage: storage,
		checker: checker,
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

	return nil
}
