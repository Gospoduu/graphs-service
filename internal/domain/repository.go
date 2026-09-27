package domain

import (
	"context"

	"github.com/google/uuid"
)

type IDConstraint interface {
	int | uuid.UUID
}

type Repository[T any, ID IDConstraint] interface {
	Create(ctx context.Context, entity T) (*T, error)
	GetByID(ctx context.Context, id ID) (*T, error)
	DeleteByID(ctx context.Context, id ID) error
	UpdateByID(ctx context.Context, id ID, entity T) error
	PatchByID(ctx context.Context, id ID, fields map[string]any) error
}
