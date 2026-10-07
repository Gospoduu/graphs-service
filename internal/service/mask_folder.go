package service

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/Gospoduu/graphs-service/internal/domain"
)

var ErrMaskFolderGraphMismatch = errors.New("mask and folder must belong to the same graph")

type MaskFolderService struct {
	*BaseService[domain.MaskFolder, uuid.UUID]
	repo        domain.MaskFolderRepository
	membersRepo domain.MaskFolderMemberRepository
	maskService *MaskService
}

func NewMaskFolderService(
	repo domain.MaskFolderRepository,
	membersRepo domain.MaskFolderMemberRepository,
	maskServise *MaskService,
) *MaskFolderService {
	return &MaskFolderService{
		BaseService: NewBaseService[domain.MaskFolder, uuid.UUID](repo),
		repo:        repo,
		membersRepo: membersRepo,
		maskService: maskServise,
	}
}

func (mfs *MaskFolderService) GetAllMaskFoldersByGraph(ctx context.Context, graph uuid.UUID) ([]domain.MaskFolder, error) {
	return mfs.repo.GetAllMaskFoldersByGraph(ctx, graph)
}

func (mfs *MaskFolderService) GetMasksByFolder(ctx context.Context, folder uuid.UUID) ([]domain.Mask, error) {
	members, err := mfs.membersRepo.GetAllMaskFolderMembersByFolder(ctx, folder)
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, len(members))
	for idx, i := range members {
		ids[idx] = i.MaskID
	}
	return mfs.maskService.GetByIDs(ctx, ids...)
}

func (mfs *MaskFolderService) AddMember(ctx context.Context, folder uuid.UUID, mask uuid.UUID) (domain.MaskFolderMember, error) {
	folderData, err := mfs.repo.GetByID(ctx, folder)
	if err != nil {
		return domain.MaskFolderMember{}, err
	}
	maskData, err := mfs.maskService.GetByID(ctx, mask)
	if err != nil {
		return domain.MaskFolderMember{}, err
	}
	if folderData.GraphID != maskData.GraphID {
		return domain.MaskFolderMember{}, ErrMaskFolderGraphMismatch
	}

	member, err := mfs.membersRepo.Create(ctx, domain.MaskFolderMember{
		FolderID: folder,
		MaskID:   mask,
	})
	if err != nil {
		zero := domain.MaskFolderMember{}
		return zero, err
	}
	return member, nil
}
