package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Gospoduu/graphs-service/internal/domain"
)

type UserRepository struct {
	*BaseRepository[domain.User, uuid.UUID]
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{
		BaseRepository: NewBaseRepository[domain.User, uuid.UUID](db),
	}
}
