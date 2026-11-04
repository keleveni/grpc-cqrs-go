package authz

import (
	"context"
	"errors"
	"testing"
	"uuid"
)

// fakeStorage は ListRelations が返す relation を差し替えられる Storage のスタブ。
type fakeStorage struct {
	relations []Relation
	listErr   error
}

func (f fakeStorage) CreateTuple(context.Context, *Tuple) error { return nil }

func (f fakeStorage) DeleteTuple(context.Context, uuid.UUID) error { return nil }

func (f fakeStorage) ListRelations(
	_ context.Context,
	objectType ObjectType,
	objectID string,
	userID string,
) ([]*Tuple, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}

	tuples := make([]*Tuple, 0, len(f.relations))
	for _, r := range f.relations {
		tuples = append(tuples, NewTuple(uuid.New(), objectType, objectID, r, userID))
	}

	return tuples, nil
}

var errBoom = errors.New("boom")

func TestCanManageRecipe(t *testing.T) {
	tests := []struct {
		name    string
		storage fakeStorage
		want    error
	}{
		{name: "owner は manage 権限を持つ", storage: fakeStorage{relations: []Relation{RelOwner}}, want: nil},
		{name: "editor は manage 権限を持たない", storage: fakeStorage{relations: []Relation{RelEditor}}, want: ErrPermissionDenied},
		{name: "viewer は manage 権限を持たない", storage: fakeStorage{relations: []Relation{RelViewer}}, want: ErrPermissionDenied},
		{name: "storage のエラーはそのまま返す", storage: fakeStorage{listErr: errBoom}, want: errBoom},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewChecker(tt.storage).CanManageRecipe(context.Background(), "user", "recipe")
			if !errors.Is(got, tt.want) {
				t.Fatalf("want %v, got %v", tt.want, got)
			}
		})
	}
}
