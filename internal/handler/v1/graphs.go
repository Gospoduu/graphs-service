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

type GraphHandler struct {
	service *service.GraphService
}

func NewGraphHandler(graphService *service.GraphService) *GraphHandler {
	return &GraphHandler{service: graphService}
}

func (gh *GraphHandler) Create(c *gin.Context) {
	var req dto.CreateGraphRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	newGraph, err := gh.service.Create(
		c.Request.Context(),
		domain.Graph{
			Name:       req.Name,
			UserID:     req.UserID,
			IsDirected: req.IsDirected,
		},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(
		http.StatusCreated,
		dto.GraphResponse{
			ID:         newGraph.ID,
			Name:       newGraph.Name,
			UserID:     newGraph.UserID,
			IsDirected: newGraph.IsDirected,
		},
	)
}

func (gh *GraphHandler) GetGraphByID(c *gin.Context) {
	rawID := c.Param("id")
	id, err := uuid.Parse(rawID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	graph, err := gh.service.GetByID(c.Request.Context(), id)
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
		dto.GraphResponse{
			ID:         graph.ID,
			Name:       graph.Name,
			UserID:     graph.UserID,
			IsDirected: graph.IsDirected,
		},
	)
}

func (gh *GraphHandler) DeleteGraphByID(c *gin.Context) {
	rawID := c.Param("id")
	id, err := uuid.Parse(rawID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err = gh.service.DeleteByID(c.Request.Context(), id)
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
		dto.DeleteGraphResponse{
			ID: id,
		},
	)
}

func (gh *GraphHandler) ToggleIsDirected(c *gin.Context) {
	rawID := c.Param("id")
	id, err := uuid.Parse(rawID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	isDirected, err := gh.service.ToggleIsDirected(c.Request.Context(), id)
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
		dto.ToggleIsDirectedResponse{
			ID:         id,
			IsDirected: isDirected,
		},
	)
}

func (gh *GraphHandler) RenameGraph(c *gin.Context) {
	rawID := c.Param("id")
	id, err := uuid.Parse(rawID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var req dto.RenameGraphRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err = gh.service.PatchByID(c.Request.Context(), id, map[string]any{"name": req.NewName})
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
		dto.RenameGraphResponse{
			ID:      id,
			NewName: req.NewName,
		},
	)
}

func (gh *GraphHandler) GetAllGraphsByUser(c *gin.Context) {
	rawID := c.Param("user_id")
	userID, err := uuid.Parse(rawID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	graphs, err := gh.service.GetAllGraphsByUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	res := make([]dto.GraphResponse, 0, len(graphs))
	for _, graph := range graphs {
		res = append(res, dto.GraphResponse{
			ID:         graph.ID,
			Name:       graph.Name,
			UserID:     graph.UserID,
			IsDirected: graph.IsDirected,
		})
	}
	c.JSON(
		http.StatusOK,
		res,
	)
}
