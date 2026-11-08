package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Gospoduu/graphs-service/internal/handler/v1/dto"
)

func (mh *MaskHandler) CreateDijkstraMask(c *gin.Context) {
	var req dto.CreateDijkstraMasksRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	newMask, members, err := mh.service.CreateDijkstraPathMask(
		c.Request.Context(),
		req.GraphID,
		req.From,
		req.To,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(
		http.StatusCreated,
		dto.MaskResponse{
			ID:       newMask.ID,
			GraphID:  newMask.GraphID,
			Name:     newMask.Name,
			Members:  members,
			Metadata: string(newMask.Metadata),
		},
	)
}

func (mh *MaskHandler) CreateFordBellmanMask(c *gin.Context) {
	var req dto.CreateFordBellmanMasksRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	newMask, members, err := mh.service.CreateFordBellmanPathMask(
		c.Request.Context(),
		req.GraphID,
		req.From,
		req.To,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(
		http.StatusCreated,
		dto.MaskResponse{
			ID:       newMask.ID,
			GraphID:  newMask.GraphID,
			Name:     newMask.Name,
			Members:  members,
			Metadata: string(newMask.Metadata),
		},
	)
}
