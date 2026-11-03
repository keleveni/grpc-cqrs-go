package query

import (
	"context"
	"time"
	"uuid"

	"github.com/kodaikumatani/grpc-cqrs-go/internal/app/recipe/entity"
)

// Storage は読み取り側の永続化。メソッドとその戻り値の読み取りモデルは 1:1 で、
// このファイルに並べて定義する。
type Storage interface {
	GetRecipe(ctx context.Context, id uuid.UUID) (*GetRecipeResult, error)
}

// GetRecipeResult は GetRecipe の読み取りモデル。
type GetRecipeResult struct {
	ID          string
	Title       string
	Description string
	Visibility  entity.Visibility
	CreatedAt   time.Time
	UpdatedAt   time.Time
	User        User
}

// User はレシピ作成者の読み取りモデル。
type User struct {
	ID    string
	Name  string
	Email string
}
