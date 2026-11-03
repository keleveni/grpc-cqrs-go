package query

import (
	"time"

	"github.com/kodaikumatani/grpc-cqrs-go/internal/app/recipe/entity"
)

// GetRecipeResult は GetRecipe ユースケースの読み取りモデル。
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
