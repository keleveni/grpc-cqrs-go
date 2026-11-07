package recipe

import (
	"context"
	"errors"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kodaikumatani/grpc-cqrs-go/internal/app/recipe/command"
	"github.com/kodaikumatani/grpc-cqrs-go/internal/app/recipe/entity"
	"github.com/kodaikumatani/grpc-cqrs-go/internal/db/gen"
)

// store は recipe の command.Storage 実装。
type store struct {
	pool    *pgxpool.Pool
	queries *gen.Queries
}

func NewStore(pool *pgxpool.Pool) command.Storage {
	return &store{pool: pool, queries: gen.New(pool)}
}

func (s *store) Create(ctx context.Context, rec *entity.Recipe) error {
	return s.queries.CreateRecipe(ctx, gen.CreateRecipeParams{
		ID:          rec.ID(),
		UserID:      rec.UserID(),
		Title:       rec.Title(),
		Description: rec.Description(),
		Visibility:  gen.Visibility(rec.Visibility().String()),
	})
}

func (s *store) Get(ctx context.Context, id uuid.UUID) (*entity.Recipe, error) {
	row, err := s.queries.GetRecipe(ctx, id)
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

	return entity.NewRecipe(
		row.ID,
		row.UserID,
		row.Title,
		row.Description,
		visibility,
	), nil
}

func (s *store) Update(ctx context.Context, rec *entity.Recipe) error {
	return s.queries.UpdateRecipe(ctx, gen.UpdateRecipeParams{
		ID:          rec.ID(),
		Title:       rec.Title(),
		Description: rec.Description(),
		Visibility:  gen.Visibility(rec.Visibility().String()),
	})
}

// ReplaceViewers は削除と追加を 1 つのトランザクションで行い、閲覧者が一時的に空に見えないようにする。
func (s *store) ReplaceViewers(ctx context.Context, id uuid.UUID, userIDs []string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // Commit 後の Rollback は何もしない

	queries := s.queries.WithTx(tx)
	if err := queries.DeleteRecipeViewers(ctx, id); err != nil {
		return err
	}
	if err := queries.CreateRecipeViewers(ctx, gen.CreateRecipeViewersParams{
		RecipeID: id,
		UserIds:  userIDs,
	}); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
