package authz

import (
	"context"
	"errors"
	"slices"
	"testing"
	"uuid"
)

// fakeStorage は ListRelations が返す relation を差し替えられる Storage のスタブ。
// ListUserIDs は受け取った relations を記録し、userIDs を返す。
type fakeStorage struct {
	relations []Relation
	listErr   error

	userIDs         []string
	queriedRelation *[]Relation
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

func (f fakeStorage) ListUserIDs(
	_ context.Context,
	_ ObjectType,
	_ string,
	relations []Relation,
) ([]string, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	if f.queriedRelation != nil {
		*f.queriedRelation = relations
	}
	return f.userIDs, nil
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

func TestUsersWithRecipe(t *testing.T) {
	tests := []struct {
		name  string
		call  func(Checker) ([]string, error)
		wantQ []Relation
	}{
		{
			name:  "manage 権限は owner で検索する",
			call:  func(c Checker) ([]string, error) { return c.UsersWithManageRecipe(context.Background(), "recipe") },
			wantQ: []Relation{RelOwner},
		},
		{
			name:  "view 権限は viewer / editor / owner で検索する",
			call:  func(c Checker) ([]string, error) { return c.UsersWithViewRecipe(context.Background(), "recipe") },
			wantQ: []Relation{RelViewer, RelEditor, RelOwner},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var queried []Relation
			storage := fakeStorage{userIDs: []string{"u1", "u2"}, queriedRelation: &queried}

			got, err := tt.call(NewChecker(storage))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !slices.Equal(got, []string{"u1", "u2"}) {
				t.Fatalf("users = %v", got)
			}
			if !slices.Equal(queried, tt.wantQ) {
				t.Fatalf("queried relations = %v, want %v", queried, tt.wantQ)
			}
		})
	}
}
