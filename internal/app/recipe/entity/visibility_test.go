package entity

import (
	"context"
	"errors"
	"testing"
)

var errDenied = errors.New("denied")

// fakeRelations は manage / edit 権限の有無を差し替えられる Relations のスタブ。
// 呼ばれたメソッドを記録し、公開範囲ごとに正しい問い合わせをしているかを確かめる。
type fakeRelations struct {
	manager, editor bool
	called          []string
}

func (f *fakeRelations) CanManageRecipe(context.Context, string, string) error {
	f.called = append(f.called, "CanManageRecipe")
	if f.manager {
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

func TestVisibility_CanEdit(t *testing.T) {
	tests := []struct {
		name       string
		visibility Visibility
		rels       fakeRelations
		want       error
		wantCalled []string
	}{
		{
			name:       "private は manage 権限があれば編集できる",
			visibility: VisibilityPrivate,
			rels:       fakeRelations{manager: true},
			want:       nil,
			wantCalled: []string{"CanManageRecipe"},
		},
		{
			name:       "private は edit 権限があっても manage 権限がなければ編集できない",
			visibility: VisibilityPrivate,
			rels:       fakeRelations{editor: true},
			want:       errDenied,
			wantCalled: []string{"CanManageRecipe"},
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
			rels:       fakeRelations{},
			want:       errDenied,
			wantCalled: []string{"CanEditRecipe"},
		},
		{
			name:       "未知の公開範囲は ErrInvalidVisibility",
			visibility: Visibility{},
			rels:       fakeRelations{manager: true, editor: true},
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
