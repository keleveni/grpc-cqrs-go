package entity

import (
	"context"
	"errors"
	"testing"
)

var errDenied = errors.New("denied")

// fakeRelationChecker は owner / view 権限の有無を差し替えられる RelationChecker のスタブ。
// 呼ばれたメソッドを記録し、公開範囲ごとに正しい問い合わせをしているかを確かめる。
type fakeRelationChecker struct {
	owner, viewer, editor bool
	called                []string
}

func (f *fakeRelationChecker) IsRecipeOwner(context.Context, string, string) error {
	f.called = append(f.called, "IsRecipeOwner")
	if f.owner {
		return nil
	}
	return errDenied
}

func (f *fakeRelationChecker) CanViewRecipe(context.Context, string, string) error {
	f.called = append(f.called, "CanViewRecipe")
	if f.viewer {
		return nil
	}
	return errDenied
}

func (f *fakeRelationChecker) CanEditRecipe(context.Context, string, string) error {
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
		checker    fakeRelationChecker
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
			checker:    fakeRelationChecker{owner: true},
			want:       nil,
			wantCalled: []string{"IsRecipeOwner"},
		},
		{
			name:       "private は view 権限があっても owner でなければ閲覧できない",
			visibility: VisibilityPrivate,
			checker:    fakeRelationChecker{viewer: true},
			want:       errDenied,
			wantCalled: []string{"IsRecipeOwner"},
		},
		{
			name:       "restricted は view 権限があれば閲覧できる",
			visibility: VisibilityRestricted,
			checker:    fakeRelationChecker{viewer: true},
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
			checker:    fakeRelationChecker{owner: true, viewer: true},
			want:       ErrInvalidVisibility,
			wantCalled: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.visibility.CanView(context.Background(), &tt.checker, "user", "recipe")
			if !errors.Is(got, tt.want) {
				t.Fatalf("want %v, got %v", tt.want, got)
			}
			if len(tt.checker.called) != len(tt.wantCalled) {
				t.Fatalf("called %v, want %v", tt.checker.called, tt.wantCalled)
			}
			for i := range tt.wantCalled {
				if tt.checker.called[i] != tt.wantCalled[i] {
					t.Fatalf("called %v, want %v", tt.checker.called, tt.wantCalled)
				}
			}
		})
	}
}

func TestVisibility_CanEdit(t *testing.T) {
	tests := []struct {
		name       string
		visibility Visibility
		checker    fakeRelationChecker
		want       error
		wantCalled []string
	}{
		{
			name:       "private は owner なら編集できる",
			visibility: VisibilityPrivate,
			checker:    fakeRelationChecker{owner: true},
			want:       nil,
			wantCalled: []string{"IsRecipeOwner"},
		},
		{
			name:       "private は editor でも owner でなければ編集できない",
			visibility: VisibilityPrivate,
			checker:    fakeRelationChecker{editor: true},
			want:       errDenied,
			wantCalled: []string{"IsRecipeOwner"},
		},
		{
			name:       "restricted は edit 権限があれば編集できる",
			visibility: VisibilityRestricted,
			checker:    fakeRelationChecker{editor: true},
			want:       nil,
			wantCalled: []string{"CanEditRecipe"},
		},
		{
			name:       "public でも edit 権限がなければ編集できない",
			visibility: VisibilityPublic,
			checker:    fakeRelationChecker{viewer: true},
			want:       errDenied,
			wantCalled: []string{"CanEditRecipe"},
		},
		{
			name:       "未知の公開範囲は ErrInvalidVisibility",
			visibility: Visibility{},
			checker:    fakeRelationChecker{owner: true, editor: true},
			want:       ErrInvalidVisibility,
			wantCalled: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.visibility.CanEdit(context.Background(), &tt.checker, "user", "recipe")
			if !errors.Is(got, tt.want) {
				t.Fatalf("want %v, got %v", tt.want, got)
			}
			if len(tt.checker.called) != len(tt.wantCalled) {
				t.Fatalf("called %v, want %v", tt.checker.called, tt.wantCalled)
			}
			for i := range tt.wantCalled {
				if tt.checker.called[i] != tt.wantCalled[i] {
					t.Fatalf("called %v, want %v", tt.checker.called, tt.wantCalled)
				}
			}
		})
	}
}
