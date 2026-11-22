package query

import (
	"context"
	"errors"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kodaikumatani/grpc-cqrs-go/internal/app/recipe/entity"
	"github.com/kodaikumatani/grpc-cqrs-go/internal/db/gen"
)

// Query は読み取り側のユースケース。
//
// 読み取りは守るべき不変条件がないので、Storage インタフェースを挟まずに
// sqlc で生成したクエリを直接使う（薄い読み取り層）。
// 閲覧できるかは、書き込み側が計算した読み取りモデル（recipe_viewers）を引くだけで判定する。
// 閲覧のルール（entity.Visibility.Viewers）も authz も使わない。
type Query struct {
	queries *gen.Queries
}

func NewQuery(pool *pgxpool.Pool) *Query {
	return &Query{queries: gen.New(pool)}
}

// GetRecipe は userID が閲覧できるレシピと作成者の情報を返す。
//
// 返しうるエラー:
//   - entity.ErrRecipeNotFound: レシピが存在しない、または閲覧できない
//     （閲覧できないレシピの存在を外から区別できないようにするため、同じエラーにする）
func (q *Query) GetRecipe(
	ctx context.Context,
	userID string,
	id uuid.UUID,
) (*GetRecipeResult, error) {
	row, err := q.queries.GetViewableRecipe(ctx, gen.GetViewableRecipeParams{
		ID:      id,
		UserIds: []string{userID, entity.Everyone},
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, entity.ErrRecipeNotFound
	}
	if err != nil {
		return nil, err
	}

	visibility, err := entity.NewVisibility(string(row.Visibility))
	if err != nil {
		return nil, err
	}

	return &GetRecipeResult{
		ID:          row.ID.String(),
		Title:       row.Title,
		Description: row.Description,
		Visibility:  visibility,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
		UserID:      row.UserID,
		UserName:    row.UserName,
		UserEmail:   row.UserEmail,
	}, nil
}
