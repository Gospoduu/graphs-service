package dto

import (
	"github.com/google/uuid"
)

type MaskFolderResponse struct {
	ID            uuid.UUID      `json:"id"`
	GraphID       uuid.UUID      `json:"graph_id"`
	Name          string         `json:"name"`
	FolderMembers []MaskResponse `json:"folder_members"`
}

type CreateMaskFolderRequest struct {
	GraphID uuid.UUID `json:"graph_id"`
	Name    string    `json:"name"`
}

type AddMaskToFolderRequest struct {
	FolderID uuid.UUID `json:"folder_id"`
	MaskID   uuid.UUID `json:"mask_id"`
}

type DeleteMaskFolderResponse struct {
	ID uuid.UUID `json:"id"`
}

type AddMaskToFolderResponse struct {
	FolderID uuid.UUID `json:"folder_id"`
	MaskID   uuid.UUID `json:"mask_id"`
}
