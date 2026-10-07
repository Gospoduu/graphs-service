package domain

import (
	"context"

	"github.com/google/uuid"
)

type IDConstraint interface {
	int | uuid.UUID
}

type Repository[T any, ID IDConstraint] interface {
	Create(ctx context.Context, entity T) (T, error)
	GetByID(ctx context.Context, id ID) (T, error)
	GetByIDs(ctx context.Context, ids ...ID) ([]T, error)
	DeleteByID(ctx context.Context, id ID) error
	UpdateByID(ctx context.Context, id ID, entity T) error
	PatchByID(ctx context.Context, id ID, fields map[string]any) error
	GetByFilter(ctx context.Context, filters map[string]any) ([]T, error)
}

type GraphRepository interface {
	Repository[Graph, uuid.UUID]
}
type NodeRepository interface {
	Repository[Node, uuid.UUID]

	GetAllNodesByGraph(
		ctx context.Context,
		graph uuid.UUID,
	) ([]Node, error)
}
type EdgeRepository interface {
	Repository[Edge, uuid.UUID]

	GetAllEdgesByGraph(
		ctx context.Context,
		graph uuid.UUID,
		sort bool,
	) ([]Edge, error)
}

type MaskRepository interface {
	Repository[Mask, uuid.UUID]
	GetAllMasksByGraph(
		ctx context.Context,
		graph uuid.UUID,
	) ([]Mask, error)
}
type MaskMembersRepository interface {
	Repository[MaskMember, uuid.UUID]
	GetAllMembersByMask(
		ctx context.Context,
		mask uuid.UUID,
	) ([]MaskMember, error)
}

type MaskFolderRepository interface {
	Repository[MaskFolder, uuid.UUID]
	GetAllMaskFoldersByGraph(
		ctx context.Context,
		graph uuid.UUID,
	) ([]MaskFolder, error)
}

type MaskFolderMemberRepository interface {
	Repository[MaskFolderMember, uuid.UUID]
	GetAllMaskFolderMembersByFolder(
		ctx context.Context,
		folder uuid.UUID,
	) ([]MaskFolderMember, error)
}
