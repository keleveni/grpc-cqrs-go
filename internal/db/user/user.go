package user

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kodaikumatani/grpc-cqrs-go/internal/app/user/command"
	"github.com/kodaikumatani/grpc-cqrs-go/internal/app/user/entity"
	"github.com/kodaikumatani/grpc-cqrs-go/internal/db/gen"
)

type store struct {
	queries *gen.Queries
}

func NewStore(pool *pgxpool.Pool) command.Storage {
	return &store{queries: gen.New(pool)}
}

func (s *store) Create(ctx context.Context, usr *entity.User) error {
	err := s.queries.CreateUser(ctx, gen.CreateUserParams{
		ID:    usr.ID(),
		Name:  usr.Name(),
		Email: usr.Email(),
	})

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return entity.ErrAlreadyExists
	}

	return err
}
