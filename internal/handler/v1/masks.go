package v1

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Gospoduu/graphs-service/internal/handler/v1/dto"
	"github.com/Gospoduu/graphs-service/internal/service"
)

type MaskHandler struct {
	service *service.MaskService
}

func NewMaskHandler(maskService *service.MaskService) *MaskHandler {
	return &MaskHandler{service: maskService}
}

func (mh *MaskHandler) CreateDFSMask(c *gin.Context) {
	var req dto.CreateMaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	newMask, members, err := mh.service.CreateDFSMask(
		c.Request.Context(),
		req.GraphID,
		req.StartID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(
		http.StatusCreated,
		dto.MaskResponse{
			ID:      newMask.ID,
			GraphID: newMask.GraphID,
			Name:    newMask.Name,
			Members: members,
		},
	)
}
func (mh *MaskHandler) CreateBFSMask(c *gin.Context) {
	var req dto.CreateMaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	newMask, members, err := mh.service.CreateBFSMask(
		c.Request.Context(),
		req.GraphID,
		req.StartID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(
		http.StatusCreated,
		dto.MaskResponse{
			ID:      newMask.ID,
			GraphID: newMask.GraphID,
			Name:    newMask.Name,
			Members: members,
		},
	)
}
func (mh *MaskHandler) CreateMSTMask(c *gin.Context) {
	var req dto.CreateMSTMaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	newMask, members, err := mh.service.CreateMSTMask(
		c.Request.Context(),
		req.GraphID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(
		http.StatusCreated,
		dto.MaskResponse{
			ID:      newMask.ID,
			GraphID: newMask.GraphID,
			Name:    newMask.Name,
			Members: members,
		},
	)
}

func (mh *MaskHandler) GetMaskByID(c *gin.Context) {
	rawID := c.Param("id")
	id, err := uuid.Parse(rawID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	mask, err := mh.service.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	members, err := mh.service.GetAllMembersByMask(c.Request.Context(), mask.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	memberIDs := make([]uuid.UUID, len(members))
	for i, member := range members {
		memberIDs[i] = member.EdgeID
	}
	c.JSON(
		http.StatusOK,
		dto.MaskResponse{
			ID:      mask.ID,
			GraphID: mask.GraphID,
			Name:    mask.Name,
			Members: memberIDs,
		},
	)
}

func (mh *MaskHandler) DeleteMaskByID(c *gin.Context) {
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
		dto.DeleteMaskResponse{
			ID: id,
		},
	)
}

func (mh *MaskHandler) GetAllMasksByGraph(c *gin.Context) {
	rawID := c.Param("graph_id")
	id, err := uuid.Parse(rawID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	masks, err := mh.service.GetAllMasksByGraph(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	res := make([]dto.MaskResponse, 0, len(masks))
	for _, mask := range masks {
		res = append(res, dto.MaskResponse{
			ID:      mask.ID,
			GraphID: mask.GraphID,
			Name:    mask.Name,
		})
	}
	c.JSON(http.StatusOK, []dto.MaskResponse(res))
}
