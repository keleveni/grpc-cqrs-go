package tuple

import (
	"context"
	"errors"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/kodaikumatani/grpc-cqrs-go/internal/authz"
	"github.com/kodaikumatani/grpc-cqrs-go/internal/db/gen"
)

// fakeDBTX は Exec が返すエラーを差し替えられる gen.DBTX のスタブ。
type fakeDBTX struct {
	execErr error
}

func (f fakeDBTX) Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, f.execErr
}

func (f fakeDBTX) Query(context.Context, string, ...interface{}) (pgx.Rows, error) {
	return nil, nil
}

func (f fakeDBTX) QueryRow(context.Context, string, ...interface{}) pgx.Row {
	return nil
}

var errBoom = errors.New("boom")

func TestCreateTuple_ErrorTranslation(t *testing.T) {
	tests := []struct {
		name    string
		execErr error
		check   func(t *testing.T, got error)
	}{
		{
			name:    "unique violation is translated to ErrAlreadyExists",
			execErr: &pgconn.PgError{Code: "23505"},
			check: func(t *testing.T, got error) {
				if !errors.Is(got, authz.ErrAlreadyExists) {
					t.Fatalf("want ErrAlreadyExists, got %v", got)
				}
			},
		},
		{
			name:    "other pg error is passed through",
			execErr: &pgconn.PgError{Code: "23503"}, // foreign_key_violation
			check: func(t *testing.T, got error) {
				if errors.Is(got, authz.ErrAlreadyExists) {
					t.Fatalf("must not translate non-23505 to ErrAlreadyExists: %v", got)
				}
				var pgErr *pgconn.PgError
				if !errors.As(got, &pgErr) || pgErr.Code != "23503" {
					t.Fatalf("want passthrough pg error 23503, got %v", got)
				}
			},
		},
		{
			name:    "generic error is passed through",
			execErr: errBoom,
			check: func(t *testing.T, got error) {
				if !errors.Is(got, errBoom) {
					t.Fatalf("want errBoom, got %v", got)
				}
			},
		},
		{
			name:    "success returns nil",
			execErr: nil,
			check: func(t *testing.T, got error) {
				if got != nil {
					t.Fatalf("want nil, got %v", got)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &store{queries: gen.New(fakeDBTX{execErr: tt.execErr})}

			got := repo.CreateTuple(
				context.Background(),
				authz.NewTuple(uuid.New(), authz.ObjectRecipe, "recipe-1", authz.RelViewer, "user-1"),
			)

			tt.check(t, got)
		})
	}
}
