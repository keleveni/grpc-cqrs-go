package query

import (
	"github.com/kodaikumatani/grpc-cqrs-go/internal/authz"
)

type Query struct {
	storage Storage
	checker authz.Checker
}

func NewQuery(
	storage Storage,
	checker authz.Checker,
) *Query {
	return &Query{
		storage: storage,
		checker: checker,
	}
}
