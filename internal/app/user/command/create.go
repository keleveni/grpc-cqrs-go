package command

import (
	"context"

	"github.com/kodaikumatani/grpc-cqrs-go/internal/app/user/entity"
)

type Command struct {
	storage Storage
}

func NewCommand(storage Storage) *Command {
	return &Command{storage: storage}
}

func (c *Command) CreateUser(
	ctx context.Context,
	id string,
	name,
	email string,
) (*entity.User, error) {
	user := entity.NewUser(id, name, email)

	if err := c.storage.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}
