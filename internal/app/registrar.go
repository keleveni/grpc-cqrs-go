package app

import (
	recipepb "github.com/kodaikumatani/grpc-cqrs-go/pkg/pb/recipe/v1"
	userpb "github.com/kodaikumatani/grpc-cqrs-go/pkg/pb/user/v1"
	"google.golang.org/grpc"
)

type Registrar struct {
	recipeHandler recipepb.RecipeServiceServer
	userHandler   userpb.UserServiceServer
}

func NewRegistrar(
	recipeHandler recipepb.RecipeServiceServer,
	userHandler userpb.UserServiceServer,
) *Registrar {
	return &Registrar{
		recipeHandler: recipeHandler,
		userHandler:   userHandler,
	}
}

func (r *Registrar) Register(app *grpc.Server) *grpc.Server {
	recipepb.RegisterRecipeServiceServer(app, r.recipeHandler)
	userpb.RegisterUserServiceServer(app, r.userHandler)

	return app
}
