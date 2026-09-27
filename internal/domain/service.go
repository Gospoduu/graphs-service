package domain

import (
	"context"
)

type Service[T any, ID IDConstraint] interface {
	GetByID(ctx context.Context, id ID) (*T, error)
	Create(ctx context.Context, entity T) (*T, error)
	DeleteByID(ctx context.Context, id ID) error
	UpdateByID(ctx context.Context, id ID, entity T) error
	PatchByID(ctx context.Context, id ID, fields map[string]any) error
}
