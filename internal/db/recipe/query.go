package recipe

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kodaikumatani/grpc-cqrs-go/internal/app/recipe/entity"
	"github.com/kodaikumatani/grpc-cqrs-go/internal/app/recipe/query"
	"github.com/kodaikumatani/grpc-cqrs-go/internal/db/gen"
)

// queryStore は recipe の読み込み側 (query.Storage) 実装。
type queryStore struct {
	queries *gen.Queries
}

func NewQuery(pool *pgxpool.Pool) query.Storage {
	return &queryStore{queries: gen.New(pool)}
}

func (s *queryStore) GetRecipe(ctx context.Context, id uuid.UUID) (*query.GetRecipeResult, error) {
	row, err := s.queries.GetRecipeWithUser(ctx, id)
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

	return &query.GetRecipeResult{
		ID:          row.ID.String(),
		Title:       row.Title,
		Description: row.Description,
		Visibility:  visibility,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
		User: query.User{
			ID:    row.UserID,
			Name:  row.UserName,
			Email: row.UserEmail,
		},
	}, nil
}
