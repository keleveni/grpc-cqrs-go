package tuple

import (
	"context"
	"errors"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kodaikumatani/grpc-cqrs-go/internal/authz"
	"github.com/kodaikumatani/grpc-cqrs-go/internal/db/gen"
)

type store struct {
	queries *gen.Queries
}

func NewStore(pool *pgxpool.Pool) authz.Storage {
	return &store{queries: gen.New(pool)}
}

func (s *store) CreateTuple(ctx context.Context, tuple *authz.Tuple) error {
	err := s.queries.CreateTuple(ctx, gen.CreateTupleParams{
		ID:         tuple.ID(),
		ObjectType: tuple.ObjectType().String(),
		ObjectID:   tuple.ObjectID(),
		Relation:   tuple.Relation().String(),
		UserID:     tuple.UserID(),
	})

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return authz.ErrAlreadyExists
	}

	return err
}

func (s *store) DeleteTuple(ctx context.Context, id uuid.UUID) error {
	return s.queries.DeleteTuple(ctx, id)
}

func (s *store) ListRelations(
	ctx context.Context,
	objectType authz.ObjectType,
	objectID string,
	userID string,
) ([]*authz.Tuple, error) {
	rows, err := s.queries.ListRelations(ctx, gen.ListRelationsParams{
		ObjectType: objectType.String(),
		ObjectID:   objectID,
		UserID:     userID,
	})
	if err != nil {
		return nil, err
	}

	tuples := make([]*authz.Tuple, len(rows))
	for i, row := range rows {
		objectType, err := authz.NewObjectType(row.ObjectType)
		if err != nil {
			return nil, err
		}

		relation, err := authz.NewRelation(row.Relation)
		if err != nil {
			return nil, err
		}

		tuples[i] = authz.NewTuple(
			row.ID,
			objectType,
			row.ObjectID,
			relation,
			row.UserID,
		)
	}

	return tuples, nil
}

func (s *store) ListUserIDs(
	ctx context.Context,
	objectType authz.ObjectType,
	objectID string,
	relations []authz.Relation,
) ([]string, error) {
	names := make([]string, 0, len(relations))
	for _, r := range relations {
		names = append(names, r.String())
	}

	return s.queries.ListUserIDsByRelations(ctx, gen.ListUserIDsByRelationsParams{
		ObjectType: objectType.String(),
		ObjectID:   objectID,
		Relations:  names,
	})
}
