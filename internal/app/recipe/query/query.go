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

// ErrInvalidCursor は一覧の起点（after）が読めないとき。
var ErrInvalidCursor = errors.New("invalid cursor")

// Query は読み取り側のユースケース。
//
// 読み取りは守るべき不変条件がないので、Storage インタフェースを挟まずに
// sqlc で生成したクエリを直接使う（薄い読み取り層）。
// 閲覧の判定は 2 段に分ける:
//   - ユーザーがどのレシピにどの権限を持つか: authz（tuple を見る）
//   - 公開範囲ごとにどの権限が必要か: SQL（GetViewableRecipe / ListViewableRecipes の WHERE）
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
	perms, err := q.checker.RecipePermissions(ctx, userID, id.String())
	if err != nil {
		return nil, err
	}

	row, err := q.queries.GetViewableRecipe(ctx, gen.GetViewableRecipeParams{
		ID:        id,
		CanManage: perms.Has(authz.PermManage),
		CanView:   perms.Has(authz.PermView),
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
// manage / view 権限を持つレシピの id を authz から受け取り、公開範囲の条件と合わせて SQL で絞り込む。
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

	manageIDs, err := q.recipeIDsWith(ctx, userID, authz.PermManage)
	if err != nil {
		return nil, err
	}
	viewIDs, err := q.recipeIDsWith(ctx, userID, authz.PermView)
	if err != nil {
		return nil, err
	}

	// 次のページがあるかを知るために 1 件多く取る
	rows, err := q.queries.ListViewableRecipes(ctx, gen.ListViewableRecipesParams{
		Before:    before,
		ManageIds: manageIDs,
		ViewIds:   viewIDs,
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

// recipeIDsWith は userID が perm を持つレシピの id を、SQL に渡せる UUID にして返す。
func (q *Query) recipeIDsWith(ctx context.Context, userID string, perm authz.Permission) ([]uuid.UUID, error) {
	objectIDs, err := q.checker.RecipeIDsWith(ctx, userID, perm)
	if err != nil {
		return nil, err
	}

	ids := make([]uuid.UUID, 0, len(objectIDs))
	for _, objectID := range objectIDs {
		id, err := uuid.Parse(objectID)
		if err != nil {
			return nil, err // tuple の object_id はレシピの id なので、読めなければデータの不整合
		}
		ids = append(ids, id)
	}

	return ids, nil
}
