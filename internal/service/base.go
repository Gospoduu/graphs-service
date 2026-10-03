package service

import (
	"context"

	"github.com/Gospoduu/graphs-service/internal/domain"
)

type BaseService[T any, ID domain.IDConstraint] struct {
	repo domain.Repository[T, ID]
}

func NewBaseService[T any, ID domain.IDConstraint](
	repo domain.Repository[T, ID],
) *BaseService[T, ID] {
	return &BaseService[T, ID]{repo: repo}
}

func (s *BaseService[T, ID]) GetByID(ctx context.Context, id ID) (T, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *BaseService[T, ID]) Create(ctx context.Context, entity T) (T, error) {
	return s.repo.Create(ctx, entity)
}

func (s *BaseService[T, ID]) UpdateByID(ctx context.Context, id ID, entity T) error {
	return s.repo.UpdateByID(ctx, id, entity)
}

func (s *BaseService[T, ID]) PatchByID(ctx context.Context, id ID, fields map[string]any) error {
	return s.repo.PatchByID(ctx, id, fields)
}

func (s *BaseService[T, ID]) DeleteByID(ctx context.Context, id ID) error {
	return s.repo.DeleteByID(ctx, id)
}
