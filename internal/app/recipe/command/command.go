package command

import (
	"context"

	"github.com/google/uuid"
	"github.com/kodaikumatani/grpc-cqrs-go/internal/app/recipe/entity"
	"github.com/kodaikumatani/grpc-cqrs-go/internal/authz"
	"github.com/samber/lo"
)

type Command struct {
	storage Storage
	tuple   authz.Storage
	checker authz.Checker
}

func NewCommand(
	storage Storage,
	tuple authz.Storage,
	checker authz.Checker,
) *Command {
	return &Command{
		storage: storage,
		tuple:   tuple,
		checker: checker,
	}
}

func (u *Command) CreateRecipe(
	ctx context.Context,
	userID string,
	title,
	description string,
) (*entity.Recipe, error) {
	recipe := entity.NewRecipe(
		lo.Must(uuid.NewV7()),
		userID,
		title,
		description,
		entity.VisibilityPrivate,
	)

	tp := authz.NewTuple(
		uuid.New(),
		authz.ObjectRecipe,
		recipe.ID().String(),
		authz.RelOwner,
		userID,
	)

	if err := u.storage.Create(ctx, recipe); err != nil {
		return nil, err
	}

	if err := u.tuple.CreateTuple(ctx, tp); err != nil {
		return nil, err
	}

	return recipe, nil
}

func (u *Command) UpdateRecipe(
	ctx context.Context,
	userID string,
	id uuid.UUID,
	title, description string,
) error {
	if err := u.checker.
		CanEditRecipe(ctx, userID, id.String()); err != nil {
		return err
	}

	recipe, err := u.storage.Get(ctx, id)
	if err != nil {
		return err
	}

	recipe.Edit(title, description)

	if err := u.storage.Update(ctx, recipe); err != nil {
		return err
	}

	return nil
}

func (u *Command) ChangeVisibility(
	ctx context.Context,
	userID string,
	id uuid.UUID,
	visibility entity.Visibility,
) error {
	if err := u.checker.IsRecipeOwner(ctx, userID, id.String()); err != nil {
		return err
	}

	recipe, err := u.storage.Get(ctx, id)
	if err != nil {
		return err
	}

	recipe.ChangeVisibility(visibility)

	return u.storage.Update(ctx, recipe)
}
