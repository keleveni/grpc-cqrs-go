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

// ErrInvalidCursor は一覧の起点（after）が読めないとき。
var ErrInvalidCursor = errors.New("invalid cursor")

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

// viewerIDs は recipe_viewers で照合する閲覧者: ユーザー本人と「全員」（public のレシピ）。
func viewerIDs(userID string) []string {
	return []string{userID, entity.Everyone}
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
		UserIds: viewerIDs(userID),
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

// ListRecipes は userID が閲覧できるレシピを新しい順に最大 limit 件返す。
// after は前のページの ListRecipesResult.Next（このレシピより古いものを返す）。空なら最初のページ。
//
// 返しうるエラー:
//   - ErrInvalidCursor: after がこのクエリの起点として読めない
func (q *Query) ListRecipes(
	ctx context.Context,
	userID string,
	after string,
	limit int32,
) (*ListRecipesResult, error) {
	before := uuid.Max() // 最初のページ: すべてのレシピより「新しい」起点
	if after != "" {
		id, err := uuid.Parse(after)
		if err != nil {
			return nil, ErrInvalidCursor
		}
		before = id
	}

	// 次のページがあるかを知るために 1 件多く取る
	rows, err := q.queries.ListViewableRecipes(ctx, gen.ListViewableRecipesParams{
		Before:    before,
		UserIds:   viewerIDs(userID),
		PageLimit: limit + 1,
	})
	if err != nil {
		return nil, err
	}

	result := &ListRecipesResult{}
	if len(rows) > int(limit) {
		rows = rows[:limit]
		result.Next = rows[len(rows)-1].ID.String()
	}

	result.Recipes = make([]ListRecipesItem, 0, len(rows))
	for _, row := range rows {
		visibility, err := entity.NewVisibility(string(row.Visibility))
		if err != nil {
			return nil, err
		}

		result.Recipes = append(result.Recipes, ListRecipesItem{
			ID:          row.ID.String(),
			UserID:      row.UserID,
			Title:       row.Title,
			Description: row.Description,
			Visibility:  visibility,
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
		})
	}

	return result, nil
}
