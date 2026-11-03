package query

import (
	"context"
	"uuid"
)

type Storage interface {
	GetRecipe(ctx context.Context, id uuid.UUID) (*GetRecipeResult, error)
}
