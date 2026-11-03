package entity

import (
	"context"
	"errors"
	"testing"
)

var errDenied = errors.New("denied")

// fakeRelations は owner / view 権限の有無を差し替えられる Relations のスタブ。
// 呼ばれたメソッドを記録し、公開範囲ごとに正しい問い合わせをしているかを確かめる。
type fakeRelations struct {
	owner, viewer, editor bool
	called                []string
}

func (f *fakeRelations) IsRecipeOwner(context.Context, string, string) error {
	f.called = append(f.called, "IsRecipeOwner")
	if f.owner {
		return nil
	}
	return errDenied
}

func (f *fakeRelations) CanViewRecipe(context.Context, string, string) error {
	f.called = append(f.called, "CanViewRecipe")
	if f.viewer {
		return nil
	}
	return errDenied
}

func (f *fakeRelations) CanEditRecipe(context.Context, string, string) error {
	f.called = append(f.called, "CanEditRecipe")
	if f.editor {
		return nil
	}
	return errDenied
}

func TestVisibility_CanView(t *testing.T) {
	tests := []struct {
		name       string
		visibility Visibility
		rels       fakeRelations
		want       error
		wantCalled []string
	}{
		{
			name:       "public は relation を問い合わせずに閲覧できる",
			visibility: VisibilityPublic,
			want:       nil,
			wantCalled: nil,
		},
		{
			name:       "private は owner なら閲覧できる",
			visibility: VisibilityPrivate,
			rels:       fakeRelations{owner: true},
			want:       nil,
			wantCalled: []string{"IsRecipeOwner"},
		},
		{
			name:       "private は view 権限があっても owner でなければ閲覧できない",
			visibility: VisibilityPrivate,
			rels:       fakeRelations{viewer: true},
			want:       errDenied,
			wantCalled: []string{"IsRecipeOwner"},
		},
		{
			name:       "restricted は view 権限があれば閲覧できる",
			visibility: VisibilityRestricted,
			rels:       fakeRelations{viewer: true},
			want:       nil,
			wantCalled: []string{"CanViewRecipe"},
		},
		{
			name:       "restricted は view 権限がなければ閲覧できない",
			visibility: VisibilityRestricted,
			want:       errDenied,
			wantCalled: []string{"CanViewRecipe"},
		},
		{
			name:       "未知の公開範囲は ErrInvalidVisibility",
			visibility: Visibility{},
			rels:       fakeRelations{owner: true, viewer: true},
			want:       ErrInvalidVisibility,
			wantCalled: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.visibility.CanView(context.Background(), &tt.rels, "user", "recipe")
			if !errors.Is(got, tt.want) {
				t.Fatalf("want %v, got %v", tt.want, got)
			}
			if len(tt.rels.called) != len(tt.wantCalled) {
				t.Fatalf("called %v, want %v", tt.rels.called, tt.wantCalled)
			}
			for i := range tt.wantCalled {
				if tt.rels.called[i] != tt.wantCalled[i] {
					t.Fatalf("called %v, want %v", tt.rels.called, tt.wantCalled)
				}
			}
		})
	}
}

func TestVisibility_CanEdit(t *testing.T) {
	tests := []struct {
		name       string
		visibility Visibility
		rels       fakeRelations
		want       error
		wantCalled []string
	}{
		{
			name:       "private は owner なら編集できる",
			visibility: VisibilityPrivate,
			rels:       fakeRelations{owner: true},
			want:       nil,
			wantCalled: []string{"IsRecipeOwner"},
		},
		{
			name:       "private は editor でも owner でなければ編集できない",
			visibility: VisibilityPrivate,
			rels:       fakeRelations{editor: true},
			want:       errDenied,
			wantCalled: []string{"IsRecipeOwner"},
		},
		{
			name:       "restricted は edit 権限があれば編集できる",
			visibility: VisibilityRestricted,
			rels:       fakeRelations{editor: true},
			want:       nil,
			wantCalled: []string{"CanEditRecipe"},
		},
		{
			name:       "public でも edit 権限がなければ編集できない",
			visibility: VisibilityPublic,
			rels:       fakeRelations{viewer: true},
			want:       errDenied,
			wantCalled: []string{"CanEditRecipe"},
		},
		{
			name:       "未知の公開範囲は ErrInvalidVisibility",
			visibility: Visibility{},
			rels:       fakeRelations{owner: true, editor: true},
			want:       ErrInvalidVisibility,
			wantCalled: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.visibility.CanEdit(context.Background(), &tt.rels, "user", "recipe")
			if !errors.Is(got, tt.want) {
				t.Fatalf("want %v, got %v", tt.want, got)
			}
			if len(tt.rels.called) != len(tt.wantCalled) {
				t.Fatalf("called %v, want %v", tt.rels.called, tt.wantCalled)
			}
			for i := range tt.wantCalled {
				if tt.rels.called[i] != tt.wantCalled[i] {
					t.Fatalf("called %v, want %v", tt.rels.called, tt.wantCalled)
				}
			}
		})
	}
}
