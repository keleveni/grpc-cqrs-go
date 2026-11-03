package query

import (
	"context"
	"errors"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kodaikumatani/grpc-cqrs-go/internal/app/recipe/entity"
	"github.com/kodaikumatani/grpc-cqrs-go/internal/authz"
	"github.com/kodaikumatani/grpc-cqrs-go/internal/db/gen"
)

// Query は読み取り側のユースケース。
//
// 読み取りは守るべき不変条件がないので、Storage インタフェースを挟まずに
// sqlc で生成したクエリを直接使う（薄い読み取り層）。
// ErrInvalidCursor は一覧の起点（after）が読めないとき。
var ErrInvalidCursor = errors.New("invalid cursor")

type Query struct {
	queries *gen.Queries
	checker authz.Checker
}

func NewQuery(
	pool *pgxpool.Pool,
	checker authz.Checker,
) *Query {
	return &Query{
		queries: gen.New(pool),
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
	row, err := q.queries.GetRecipeWithUser(ctx, id)
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

	if err := visibility.CanView(ctx, q.checker, userID, id.String()); err != nil {
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
// 閲覧できるかの条件は SQL（ListViewableRecipes）で絞り込む。条件は entity.Visibility.CanView と同じで、
// どの relation が manage / view 権限を与えるかは authz から渡す。
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
		Before:          before,
		UserID:          userID,
		ManageRelations: relationNames(authz.PermManage),
		ViewRelations:   relationNames(authz.PermView),
		PageLimit:       limit + 1,
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

// relationNames は perm を与える relation を SQL に渡せる文字列にする。
func relationNames(perm authz.Permission) []string {
	relations := authz.RelationsGranting(authz.ObjectRecipe, perm)

	names := make([]string, 0, len(relations))
	for _, r := range relations {
		names = append(names, r.String())
	}

	return names
}
