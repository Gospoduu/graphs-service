package service

import (
	"github.com/Gospoduu/graphs-service/internal/domain"
	"github.com/google/uuid"
)

type UserService struct {
	*BaseService[domain.User, uuid.UUID]
}

func NewUserService(repo domain.Repository[domain.User, uuid.UUID]) *UserService {
	return &UserService{
		BaseService: NewBaseService[domain.User, uuid.UUID](repo),
	}
}
