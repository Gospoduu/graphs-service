package v1

import "github.com/gin-gonic/gin"

type Handlers struct {
	Graph *GraphHandler
	Node  *NodeHandler
	Edge  *EdgeHandler
	User  *UserHandler
	Mask  *MaskHandler
}

func RegisterRoutes(rg *gin.RouterGroup, h *Handlers) {
	graphs := rg.Group("/graphs")
	{
		graphs.POST("", h.Graph.Create)
		graphs.GET("/:id", h.Graph.GetGraphByID)
		graphs.GET("/user/:user_id", h.Graph.GetAllGraphsByUser)
		graphs.DELETE("/:id", h.Graph.DeleteGraphByID)
		graphs.PATCH("/:id/rename", h.Graph.RenameGraph)
		graphs.PATCH("/:id/toggle-directed", h.Graph.ToggleIsDirected)
	}

	nodes := rg.Group("/nodes")
	{
		nodes.POST("", h.Node.Create)
		nodes.GET("/:id", h.Node.GetNodeByID)
		nodes.GET("/graph/:graph_id", h.Node.GetAllNodesByGraph)
		nodes.DELETE("/:id", h.Node.DeleteNodeByID)
		nodes.PATCH("/:id/position", h.Node.ChangePosition)
	}

	edges := rg.Group("/edges")
	{
		edges.POST("", h.Edge.Create)
		edges.GET("/:id", h.Edge.GetEdgeByID)
		edges.GET("/graph/:graph_id", h.Edge.GetAllEdgesByGraph)
		edges.DELETE("/:id", h.Edge.DeleteEdgeByID)
		edges.PATCH("/:id/toggle-direction", h.Edge.ToggleDirect)
		edges.PATCH("/:id/weight", h.Edge.ChangeWeight)
	}

	users := rg.Group("/users")
	{
		users.POST("", h.User.Create)
		users.GET("/:id", h.User.GetUserByID)
	}
	masks := rg.Group("/masks")
	{
		masks.POST("/dfs", h.Mask.CreateDFSMask)
		masks.POST("/bfs", h.Mask.CreateBFSMask)
		masks.POST("/mst", h.Mask.CreateMSTMask)
		masks.GET("/:id", h.Mask.GetMaskByID)
		masks.GET("/graph/:graph_id", h.Mask.GetAllMasksByGraph)
		masks.DELETE("/:id", h.Mask.DeleteMaskByID)
	}
}
