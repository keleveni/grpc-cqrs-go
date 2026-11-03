package query

import (
	"context"
	"time"
	"uuid"

	"github.com/kodaikumatani/grpc-cqrs-go/internal/app/recipe/entity"
)

// getRecipeStorage は GetRecipe ユースケースが必要とする永続化。
type getRecipeStorage interface {
	GetRecipe(ctx context.Context, id uuid.UUID) (*GetRecipeResult, error)
}

// GetRecipeResult は GetRecipe の読み取りモデル。
type GetRecipeResult struct {
	ID          string
	Title       string
	Description string
	Visibility  entity.Visibility
	CreatedAt   time.Time
	UpdatedAt   time.Time
	// User はレシピの作成者
	User struct {
		ID    string
		Name  string
		Email string
	}
}

// GetRecipe はレシピと作成者の情報を、公開範囲に応じた閲覧権限を確認したうえで返す。
//
// 返しうるエラー:
//   - entity.ErrRecipeNotFound: レシピが存在しない
//   - authz.ErrPermissionDenied: 公開範囲に対して閲覧権限がない
func (q *Query) GetRecipe(
	ctx context.Context,
	userID string,
	id uuid.UUID,
) (*GetRecipeResult, error) {
	result, err := q.storage.GetRecipe(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := q.checker.CanViewRecipe(ctx, userID, id.String(), result.Visibility); err != nil {
		return nil, err
	}

	return result, nil
}
