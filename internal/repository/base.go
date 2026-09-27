package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/Gospoduu/graphs-service/internal/domain"
)

type BaseRepository[T any, ID domain.IDConstraint] struct {
	db *gorm.DB
}

func NewBaseRepository[T any, ID domain.IDConstraint](db *gorm.DB) *BaseRepository[T, ID] {
	return &BaseRepository[T, ID]{db: db}
}

func (r *BaseRepository[T, ID]) Create(ctx context.Context, entity T) (*T, error) {
	err := r.db.WithContext(ctx).Create(&entity).Error
	if err != nil {
		return nil, err
	}
	return &entity, nil
}

func (r *BaseRepository[T, ID]) GetByID(ctx context.Context, id ID) (*T, error) {
	var entity T
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&entity).Error
	if err != nil {
		return nil, err
	}
	return &entity, nil
}

func (r *BaseRepository[T, ID]) DeleteByID(ctx context.Context, id ID) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(new(T)).Error
}

func (r *BaseRepository[T, ID]) UpdateByID(ctx context.Context, id ID, entity T) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Select("*").Updates(entity).Error
}

func (r *BaseRepository[T, ID]) PatchByID(ctx context.Context, id ID, fields map[string]any) error {
	return r.db.WithContext(ctx).Model(new(T)).Where("id = ?", id).Updates(fields).Error
}
