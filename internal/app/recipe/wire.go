package recipe

import (
	"github.com/google/wire"
	"github.com/kodaikumatani/grpc-cqrs-go/internal/app/recipe/command"
	"github.com/kodaikumatani/grpc-cqrs-go/internal/app/recipe/query"
)

var Set = wire.NewSet(
	NewHandler,
	command.NewCommand,
	query.NewQuery,
)
