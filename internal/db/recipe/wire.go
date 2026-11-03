package recipe

import (
	"github.com/google/wire"
)

var Set = wire.NewSet(
	NewCommand,
)
