package query

import (
	"context"
	"time"
	"uuid"

	"github.com/kodaikumatani/grpc-cqrs-go/internal/app/recipe/entity"
)

// Storage は読み取り側の永続化。各メソッドと、その戻り値の XxxRecord をこのファイルに 1:1 で並べる。
type Storage interface {
	GetRecipe(ctx context.Context, id uuid.UUID) (*GetRecipeRecord, error)
}

// GetRecipeRecord は Storage.GetRecipe が返すデータ。
// ユースケースが GetRecipeResult に変換するので、呼び出し側に返す形とは独立に変えられる。
type GetRecipeRecord struct {
	ID          string
	Title       string
	Description string
	Visibility  entity.Visibility
	CreatedAt   time.Time
	UpdatedAt   time.Time
	UserID      string
	UserName    string
	UserEmail   string
}
