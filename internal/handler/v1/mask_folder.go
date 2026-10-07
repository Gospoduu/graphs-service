package v1

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Gospoduu/graphs-service/internal/domain"
	"github.com/Gospoduu/graphs-service/internal/handler/v1/dto"
	"github.com/Gospoduu/graphs-service/internal/service"
)

type MaskFolderHandler struct {
	service *service.MaskFolderService
}

func NewMaskFolderHandler(maskFolderService *service.MaskFolderService) *MaskFolderHandler {
	return &MaskFolderHandler{service: maskFolderService}
}

func (mh *MaskFolderHandler) GetMaskFolderByID(c *gin.Context) {
	rawID := c.Param("id")
	id, err := uuid.Parse(rawID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	maskFolder, err := mh.service.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	members, err := mh.service.GetMasksByFolder(c.Request.Context(), maskFolder.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	membersForResponse := make([]dto.MaskResponse, len(members))
	for i, member := range members {
		membersForResponse[i] = dto.MaskResponse{
			ID:      member.ID,
			GraphID: member.GraphID,
			Name:    member.Name,
			Members: nil,
		}
	}
	c.JSON(
		http.StatusOK,
		dto.MaskFolderResponse{
			ID:            maskFolder.ID,
			GraphID:       maskFolder.GraphID,
			Name:          maskFolder.Name,
			FolderMembers: membersForResponse,
		},
	)
}

func (mh *MaskFolderHandler) DeleteMaskFolderByID(c *gin.Context) {
	rawID := c.Param("id")
	id, err := uuid.Parse(rawID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err = mh.service.DeleteByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(
		http.StatusOK,
		dto.DeleteMaskFolderResponse{
			ID: id,
		},
	)
}

func (mh *MaskFolderHandler) GetAllMaskFoldersByGraph(c *gin.Context) {
	rawID := c.Param("graph_id")
	id, err := uuid.Parse(rawID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	maskFolders, err := mh.service.GetAllMaskFoldersByGraph(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	res := make([]dto.MaskFolderResponse, len(maskFolders))
	for i, maskFolder := range maskFolders {
		res[i] = dto.MaskFolderResponse{
			ID:      maskFolder.ID,
			GraphID: maskFolder.GraphID,
			Name:    maskFolder.Name,
		}
	}
	c.JSON(http.StatusOK, res)
}

func (mh *MaskFolderHandler) AddMaskToFolder(c *gin.Context) {
	var req dto.AddMaskToFolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	newMember, err := mh.service.AddMember(c.Request.Context(), req.FolderID, req.MaskID)
	if err != nil {
		if errors.Is(err, service.ErrMaskFolderGraphMismatch) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(
		http.StatusOK,
		dto.AddMaskToFolderResponse{
			FolderID: newMember.FolderID,
			MaskID:   newMember.MaskID,
		},
	)
}

func (mh *MaskFolderHandler) Create(c *gin.Context) {
	var req dto.CreateMaskFolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	newFolder, err := mh.service.Create(
		c.Request.Context(),
		domain.MaskFolder{
			GraphID: req.GraphID,
			Name:    req.Name,
		},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(
		http.StatusCreated,
		dto.MaskFolderResponse{
			ID:            newFolder.ID,
			GraphID:       newFolder.GraphID,
			Name:          newFolder.Name,
			FolderMembers: []dto.MaskResponse{},
		},
	)
}
