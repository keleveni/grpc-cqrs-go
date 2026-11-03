package authz

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/kodaikumatani/grpc-cqrs-go/internal/app/recipe/entity"
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

func TestCanViewRecipe(t *testing.T) {
	tests := []struct {
		name       string
		visibility entity.Visibility
		storage    fakeStorage
		want       error
	}{
		{
			name:       "public is viewable without relation",
			visibility: entity.VisibilityPublic,
			storage:    fakeStorage{listErr: errBoom}, // storage を引かないこと
			want:       nil,
		},
		{
			name:       "private is viewable by owner",
			visibility: entity.VisibilityPrivate,
			storage:    fakeStorage{relations: []Relation{RelOwner}},
			want:       nil,
		},
		{
			name:       "private is not viewable by viewer",
			visibility: entity.VisibilityPrivate,
			storage:    fakeStorage{relations: []Relation{RelViewer}},
			want:       ErrPermissionDenied,
		},
		{
			name:       "restricted is viewable by viewer",
			visibility: entity.VisibilityRestricted,
			storage:    fakeStorage{relations: []Relation{RelViewer}},
			want:       nil,
		},
		{
			name:       "restricted is viewable by editor",
			visibility: entity.VisibilityRestricted,
			storage:    fakeStorage{relations: []Relation{RelEditor}},
			want:       nil,
		},
		{
			name:       "restricted is not viewable without relation",
			visibility: entity.VisibilityRestricted,
			storage:    fakeStorage{},
			want:       ErrPermissionDenied,
		},
		{
			name:       "storage error is passed through",
			visibility: entity.VisibilityRestricted,
			storage:    fakeStorage{listErr: errBoom},
			want:       errBoom,
		},
		{
			name:       "unknown visibility is denied",
			visibility: entity.Visibility{},
			storage:    fakeStorage{relations: []Relation{RelOwner}},
			want:       ErrPermissionDenied,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewChecker(tt.storage).
				CanViewRecipe(context.Background(), "user", "recipe", tt.visibility)
			if !errors.Is(got, tt.want) {
				t.Fatalf("want %v, got %v", tt.want, got)
			}
		})
	}
}
