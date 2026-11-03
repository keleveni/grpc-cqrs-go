package query

import (
	"context"

	"github.com/google/uuid"
)

type Storage interface {
	GetRecipe(ctx context.Context, id uuid.UUID) (*GetRecipeResult, error)
}
