package command

import (
	"context"
	"uuid"

	"github.com/kodaikumatani/grpc-cqrs-go/internal/app/recipe/entity"
	"github.com/kodaikumatani/grpc-cqrs-go/internal/authz"
)

// ViewerProjector は読み取りモデル（recipe_viewers: 誰がレシピを閲覧できるか）を書き込み側から更新する。
//
// 閲覧者は公開範囲と tuple から決まるので、どちらかが変わったら RefreshViewers で計算し直す
// （レシピの作成・公開範囲の変更・共有）。読み取り側はこの結果を引くだけで、閲覧のルールを持たない。
type ViewerProjector struct {
	storage Storage
	checker authz.Checker
}

func NewViewerProjector(
	storage Storage,
	checker authz.Checker,
) *ViewerProjector {
	return &ViewerProjector{
		storage: storage,
		checker: checker,
	}
}

// RefreshViewers は recipeID のレシピの閲覧者を、今の公開範囲と tuple から計算し直して保存する。
//
// 返しうるエラー:
//   - entity.ErrRecipeNotFound: レシピが存在しない
func (p *ViewerProjector) RefreshViewers(ctx context.Context, recipeID string) error {
	id, err := uuid.Parse(recipeID)
	if err != nil {
		return entity.ErrRecipeNotFound // uuid として読めない id のレシピは存在しない
	}

	recipe, err := p.storage.Get(ctx, id)
	if err != nil {
		return err
	}

	return p.project(ctx, recipe)
}

func (p *ViewerProjector) project(ctx context.Context, recipe *entity.Recipe) error {
	viewers, err := recipe.Viewers(ctx, p.checker)
	if err != nil {
		return err
	}

	return p.storage.ReplaceViewers(ctx, recipe.ID(), viewers)
}
