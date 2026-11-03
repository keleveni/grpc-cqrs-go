package query

import (
	"context"
	"uuid"

	"github.com/kodaikumatani/grpc-cqrs-go/internal/authz"
)

type Query struct {
	storage Storage
	checker authz.Checker
}

func NewQuery(
	storage Storage,
	checker authz.Checker,
) *Query {
	return &Query{
		storage: storage,
		checker: checker,
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

	return &GetRecipeResult{
		ID:          record.ID,
		Title:       record.Title,
		Description: record.Description,
		Visibility:  record.Visibility,
		CreatedAt:   record.CreatedAt,
		UpdatedAt:   record.UpdatedAt,
		UserID:      record.UserID,
		UserName:    record.UserName,
		UserEmail:   record.UserEmail,
	}, nil
}
