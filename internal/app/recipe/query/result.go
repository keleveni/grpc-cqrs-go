package query

import (
	"time"

	"github.com/kodaikumatani/grpc-cqrs-go/internal/app/recipe/entity"
)

// GetRecipeResult は GetRecipe の読み取りモデル。
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

// ListRecipesResult は ListRecipes の読み取りモデル。
type ListRecipesResult struct {
	Recipes []ListRecipesItem
	// Next は次のページの起点。ListRecipes の after に渡す。次のページがなければ空。
	Next string
}

// ListRecipesItem は一覧の 1 件。
type ListRecipesItem struct {
	ID          string
	UserID      string
	Title       string
	Description string
	Visibility  entity.Visibility
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
