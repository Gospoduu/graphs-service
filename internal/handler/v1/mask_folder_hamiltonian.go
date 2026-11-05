package v1

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Gospoduu/graphs-service/internal/domain"
	"github.com/Gospoduu/graphs-service/internal/handler/v1/dto"
	"github.com/Gospoduu/graphs-service/internal/service"
)

// CreateHamiltonianMasks обслуживает POST /api/v1/mask-folders/hamiltonian.
// Ответ отправляется после завершения перебора и сохранения масок.
func (mh *MaskFolderHandler) CreateHamiltonianMasks(c *gin.Context) {
	var req dto.CreateHamiltonianMasksRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.GraphID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "graph_id must not be zero"})
		return
	}

	result, err := mh.service.CreateHamiltonianMasks(c.Request.Context(), req.GraphID, 4)
	response := hamiltonianMasksResponse(result)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, context.DeadlineExceeded):
			status = http.StatusGatewayTimeout
		case errors.Is(err, context.Canceled):
			status = http.StatusRequestTimeout
		case errors.Is(err, gorm.ErrRecordNotFound):
			status = http.StatusNotFound
		case errors.Is(err, service.NonConnectedGraphError),
			errors.Is(err, service.ErrInvalidHamiltonianStart):
			status = http.StatusBadRequest
		}
		// При частичном сохранении не теряем ID папки и число готовых масок.
		response.Error = err.Error()
		c.JSON(status, response)
		return
	}

	status := http.StatusOK
	if result.Folder != nil {
		status = http.StatusCreated
	}
	c.JSON(status, response)
}

func hamiltonianMasksResponse(result service.HamiltonianMasksResult) dto.CreatePathMasksResponse[dto.HamiltonianAnalysisResponse] {
	convertEdges := func(edges []domain.Edge) []dto.SuggestedEdgeResponse {
		response := make([]dto.SuggestedEdgeResponse, 0, len(edges))
		for _, edge := range edges {
			response = append(response, dto.SuggestedEdgeResponse{
				GraphID: edge.GraphID, SourceID: edge.SourceID,
				TargetID: edge.TargetID, Weight: edge.Weight,
			})
		}
		return response
	}
	response := dto.CreatePathMasksResponse[dto.HamiltonianAnalysisResponse]{
		MasksCreated: result.MasksCreated,
		Analysis: dto.HamiltonianAnalysisResponse{
			Kind:        result.Analysis.Kind,
			StartNodeID: result.Analysis.StartNodeID,
			PathEdges:   convertEdges(result.Analysis.PathEdges),
			CycleEdges:  convertEdges(result.Analysis.CycleEdges),
		},
	}
	if folder := result.Folder; folder != nil {
		response.Folder = &dto.PathFolderResponse{
			ID: folder.ID, GraphID: folder.GraphID, Name: folder.Name,
			Metadata: json.RawMessage(folder.Metadata),
		}
	}
	return response
}
