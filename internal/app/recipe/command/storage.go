package command

import (
	"context"
	"uuid"

	"github.com/kodaikumatani/grpc-cqrs-go/internal/app/recipe/entity"
)

type Storage interface {
	Create(ctx context.Context, recipe *entity.Recipe) error
	Get(ctx context.Context, id uuid.UUID) (*entity.Recipe, error)
	Update(ctx context.Context, rec *entity.Recipe) error
	// ReplaceViewers は読み取りモデル（recipe_viewers）のレシピの閲覧者を userIDs で置き換える。
	ReplaceViewers(ctx context.Context, id uuid.UUID, userIDs []string) error
}
