package query

import (
	"time"

	"github.com/kodaikumatani/grpc-cqrs-go/internal/app/recipe/entity"
)

// GetRecipeResult は GetRecipe の読み取りモデル（呼び出し側に返す形）。
type GetRecipeResult struct {
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
