package query

import (
	"context"

	"github.com/google/uuid"
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

func (q *Query) Get(
	ctx context.Context,
	userID string,
	id uuid.UUID,
) (*RecipeWithUser, error) {
	result, err := q.storage.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := q.checker.CanViewRecipe(ctx, userID, id.String(), result.Visibility); err != nil {
		return nil, err
	}

	return result, nil
}
