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

type NodeHandler struct {
	service *service.NodeService
}

func NewNodeHandler(nodeService *service.NodeService) *NodeHandler {
	return &NodeHandler{service: nodeService}
}

func (nh *NodeHandler) Create(c *gin.Context) {
	var req dto.CreateNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	newNode, err := nh.service.Create(
		c.Request.Context(),
		domain.Node{
			GraphID:  req.GraphID,
			Name:     req.Name,
			Metadata: datatypes.JSON(req.Metadata),
			X:        req.X,
			Y:        req.Y,
		},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(
		http.StatusCreated,
		dto.NodeResponse{
			ID:       newNode.ID,
			GraphID:  newNode.GraphID,
			Metadata: json.RawMessage(newNode.Metadata),
			Name:     newNode.Name,
			X:        newNode.X,
			Y:        newNode.Y,
		},
	)
}

func (nh *NodeHandler) GetNodeByID(c *gin.Context) {
	rawID := c.Param("id")
	id, err := uuid.Parse(rawID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	node, err := nh.service.GetByID(c.Request.Context(), id)
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
		dto.NodeResponse{
			ID:       node.ID,
			GraphID:  node.GraphID,
			Name:     node.Name,
			Metadata: json.RawMessage(node.Metadata),
			X:        node.X,
			Y:        node.Y,
		},
	)
}

func (nh *NodeHandler) DeleteNodeByID(c *gin.Context) {
	rawID := c.Param("id")
	id, err := uuid.Parse(rawID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err = nh.service.DeleteByID(c.Request.Context(), id)
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
		dto.DeleteNodeResponse{
			ID: id,
		},
	)
}

func (nh *NodeHandler) ChangePosition(c *gin.Context) {
	rawID := c.Param("id")
	id, err := uuid.Parse(rawID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var req dto.ChangePositionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err = nh.service.PatchByID(c.Request.Context(), id, map[string]any{"x": req.X, "y": req.Y})
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
		dto.ChangePositionResponse{
			ID: id,
		},
	)
}

func (nh *NodeHandler) GetAllNodesByGraph(c *gin.Context) {
	rawGraphID := c.Param("graph_id")
	graphID, err := uuid.Parse(rawGraphID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	nodes, err := nh.service.GetAllNodesByGraph(c.Request.Context(), graphID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	res := make([]dto.NodeResponse, 0, len(nodes))
	for _, node := range nodes {
		res = append(res, dto.NodeResponse{
			ID:       node.ID,
			GraphID:  node.GraphID,
			Metadata: json.RawMessage(node.Metadata),
			Name:     node.Name,
			X:        node.X,
			Y:        node.Y,
		})
	}
	c.JSON(http.StatusOK, res)
}
