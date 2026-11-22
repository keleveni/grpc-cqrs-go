package authz

import (
	"context"
	"uuid"
)

type Storage interface {
	CreateTuple(ctx context.Context, tuple *Tuple) error
	DeleteTuple(ctx context.Context, id uuid.UUID) error
	ListRelations(
		ctx context.Context,
		objectType ObjectType,
		objectID string,
		userID string,
	) ([]*Tuple, error)
	// ListUserIDs は objectType の objectID に対して relations のいずれかを持つ userID を返す。
	ListUserIDs(
		ctx context.Context,
		objectType ObjectType,
		objectID string,
		relations []Relation,
	) ([]string, error)
}
