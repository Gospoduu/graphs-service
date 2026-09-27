package v1

import "github.com/gin-gonic/gin"

type Handlers struct {
	Graph *GraphHandler
	Node  *NodeHandler
	Edge  *EdgeHandler
	User  *UserHandler
}

func RegisterRoutes(rg *gin.RouterGroup, h *Handlers) {
	graphs := rg.Group("/graphs")
	{
		graphs.POST("", h.Graph.Create)
		graphs.GET("/:id", h.Graph.GetGraphByID)
		graphs.DELETE("/:id", h.Graph.DeleteGraphByID)
		graphs.PATCH("/:id/rename", h.Graph.RenameGraph)
		graphs.PATCH("/:id/toggle-directed", h.Graph.ToggleIsDirected)
	}

	nodes := rg.Group("/nodes")
	{
		nodes.POST("", h.Node.Create)
		nodes.GET("/:id", h.Node.GetNodeByID)
		nodes.DELETE("/:id", h.Node.DeleteNodeByID)
		nodes.PATCH("/:id/position", h.Node.ChangePosition)
	}

	edges := rg.Group("/edges")
	{
		edges.POST("", h.Edge.Create)
		edges.GET("/:id", h.Edge.GetEdgeByID)
		edges.DELETE("/:id", h.Edge.DeleteEdgeByID)
		edges.PATCH("/:id/toggle-direction", h.Edge.ToggleDirect)
		edges.PATCH("/:id/weight", h.Edge.ChangeWeight)
	}

	users := rg.Group("/users")
	{
		users.POST("", h.User.Create)
		users.GET("/:id", h.User.GetUserByID)
	}
}
