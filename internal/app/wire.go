package app

import (
	"github.com/google/wire"
	"github.com/kodaikumatani/grpc-cqrs-go/internal/app/recipe"
	"github.com/kodaikumatani/grpc-cqrs-go/internal/app/recipe/command"
	"github.com/kodaikumatani/grpc-cqrs-go/internal/app/share"
	"github.com/kodaikumatani/grpc-cqrs-go/internal/app/user"
)

var Set = wire.NewSet(
	NewRegistrar,
	recipe.Set,
	share.Set,
	user.Set,
	// 共有したら recipe の閲覧者を計算し直す
	wire.Bind(new(share.RecipeViewers), new(*command.ViewerProjector)),
)
