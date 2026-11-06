package authz

import (
	"context"
	"errors"
	"slices"
	"testing"
	"uuid"
)

// fakeStorage は ListRelations が返す relation を差し替えられる Storage のスタブ。
// ListObjectIDs は受け取った relations を記録し、objectIDs を返す。
type fakeStorage struct {
	relations []Relation
	listErr   error

	objectIDs       []string
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

func (f fakeStorage) ListObjectIDs(
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
	return f.objectIDs, nil
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

func TestRecipePermissions(t *testing.T) {
	tests := []struct {
		name      string
		relations []Relation
		want      map[Permission]bool
	}{
		{
			name:      "owner は view / edit / manage をすべて持つ",
			relations: []Relation{RelOwner},
			want:      map[Permission]bool{PermView: true, PermEdit: true, PermManage: true},
		},
		{
			name:      "editor は view / edit を持ち、manage は持たない",
			relations: []Relation{RelEditor},
			want:      map[Permission]bool{PermView: true, PermEdit: true, PermManage: false},
		},
		{
			name:      "viewer は view だけを持つ",
			relations: []Relation{RelViewer},
			want:      map[Permission]bool{PermView: true, PermEdit: false, PermManage: false},
		},
		{
			name:      "relation がなければ何も持たない",
			relations: nil,
			want:      map[Permission]bool{PermView: false, PermEdit: false, PermManage: false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewChecker(fakeStorage{relations: tt.relations}).
				RecipePermissions(context.Background(), "user", "recipe")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			for perm, want := range tt.want {
				if got.Has(perm) != want {
					t.Errorf("Has(%v) = %v, want %v", perm, got.Has(perm), want)
				}
			}
		})
	}

	t.Run("storage のエラーはそのまま返す", func(t *testing.T) {
		_, err := NewChecker(fakeStorage{listErr: errBoom}).RecipePermissions(context.Background(), "user", "recipe")
		if !errors.Is(err, errBoom) {
			t.Fatalf("want errBoom, got %v", err)
		}
	})
}

func TestRecipeIDsWith(t *testing.T) {
	t.Run("権限を与える relation で検索し、見つかった ID を返す", func(t *testing.T) {
		var queried []Relation
		storage := fakeStorage{objectIDs: []string{"r1", "r2"}, queriedRelation: &queried}

		got, err := NewChecker(storage).RecipeIDsWith(context.Background(), "user", PermManage)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !slices.Equal(got, []string{"r1", "r2"}) {
			t.Fatalf("ids = %v", got)
		}
		if !slices.Equal(queried, []Relation{RelOwner}) {
			t.Fatalf("queried relations = %v, want [owner]", queried)
		}
	})
}
