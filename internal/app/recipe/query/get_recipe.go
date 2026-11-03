package query

import (
	"context"
	"time"
	"uuid"

	"github.com/kodaikumatani/grpc-cqrs-go/internal/app/recipe/entity"
)

// GetRecipeResult は GetRecipe の読み取りモデル（呼び出し側に返す形）。
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
	record, err := q.storage.GetRecipe(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := q.checker.CanViewRecipe(ctx, userID, id.String(), record.Visibility); err != nil {
		return nil, err
	}

	result := &GetRecipeResult{
		ID:          record.ID,
		Title:       record.Title,
		Description: record.Description,
		Visibility:  record.Visibility,
		CreatedAt:   record.CreatedAt,
		UpdatedAt:   record.UpdatedAt,
	}
	result.User.ID = record.User.ID
	result.User.Name = record.User.Name
	result.User.Email = record.User.Email

	return result, nil
}
