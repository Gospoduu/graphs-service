package v1

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/Gospoduu/graphs-service/internal/domain"
	"github.com/Gospoduu/graphs-service/internal/handler/v1/dto"
	"github.com/Gospoduu/graphs-service/internal/service"
)

type EdgeHandler struct {
	service *service.EdgeService
}

func NewEdgeHandler(edgeService *service.EdgeService) *EdgeHandler {
	return &EdgeHandler{service: edgeService}
}

func (eh *EdgeHandler) Create(c *gin.Context) {
	var req dto.CreateEdgeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	newEdge, err := eh.service.Create(
		c.Request.Context(),
		domain.Edge{
			GraphID:  req.GraphID,
			SourceID: req.SourceID,
			TargetID: req.TargetID,
			Metadata: datatypes.JSON(req.Metadata),
			Weight:   req.Weight,
		},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(
		http.StatusCreated,
		dto.EdgeResponse{
			ID:       newEdge.ID,
			GraphID:  newEdge.GraphID,
			SourceID: newEdge.SourceID,
			TargetID: newEdge.TargetID,
			Metadata: json.RawMessage(newEdge.Metadata),
			Weight:   newEdge.Weight,
		},
	)
}

func (eh *EdgeHandler) GetEdgeByID(c *gin.Context) {
	rawID := c.Param("id")
	id, err := uuid.Parse(rawID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	edge, err := eh.service.GetByID(c.Request.Context(), id)
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
		dto.EdgeResponse{
			ID:       edge.ID,
			GraphID:  edge.GraphID,
			SourceID: edge.SourceID,
			TargetID: edge.TargetID,
			Metadata: json.RawMessage(edge.Metadata),
			Weight:   edge.Weight,
		},
	)
}

func (eh *EdgeHandler) DeleteEdgeByID(c *gin.Context) {
	rawID := c.Param("id")
	id, err := uuid.Parse(rawID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err = eh.service.DeleteByID(c.Request.Context(), id)
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
		dto.DeleteEdgeResponse{
			ID: id,
		},
	)
}

func (eh *EdgeHandler) ToggleDirect(c *gin.Context) {
	rawID := c.Param("id")
	id, err := uuid.Parse(rawID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	newSource, newTarget, err := eh.service.ToggleDirect(c.Request.Context(), id)
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
		dto.ToggleDirectionResponse{
			ID:       id,
			SourceID: newSource,
			TargetID: newTarget,
		},
	)
}

func (eh *EdgeHandler) ChangeWeight(c *gin.Context) {
	rawID := c.Param("id")
	id, err := uuid.Parse(rawID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var req dto.ChangeWeightRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err = eh.service.PatchByID(c.Request.Context(), id, map[string]any{"weight": req.Weight})
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
		dto.ChangeWeightResponse{
			ID: id,
		},
	)
}
