package main

import (
	"github.com/gin-gonic/gin"
)

type PingResponse struct {
	Message string
}

func main() {
	router := gin.Default()

	router.GET(
		"/ping",
		func(c *gin.Context) {
			c.JSON(200, PingResponse{Message: "pong"})
		},
	)
	router.Run()
}
