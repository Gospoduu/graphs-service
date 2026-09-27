package main

import (
	"log"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/Gospoduu/graphs-service/internal/config"
	"github.com/Gospoduu/graphs-service/internal/database"
	"github.com/Gospoduu/graphs-service/internal/domain"
	v1 "github.com/Gospoduu/graphs-service/internal/handler/v1"
	"github.com/Gospoduu/graphs-service/internal/repository"
	"github.com/Gospoduu/graphs-service/internal/service"
)

func main() {

	dbCfg := config.NewPostgresConfigFromEnv()

	db, err := database.NewPostgresDB(dbCfg)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(
		&domain.User{},
		&domain.Graph{},
		&domain.Node{},
		&domain.Edge{},
	); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	userRepo := repository.NewUserRepository(db)
	graphRepo := repository.NewGraphRepository(db)
	nodeRepo := repository.NewNodeRepository(db)
	edgeRepo := repository.NewEdgeRepository(db)

	userService := service.NewUserService(userRepo)
	graphService := service.NewGraphService(graphRepo)
	nodeService := service.NewNodeService(nodeRepo)
	edgeService := service.NewEdgeService(edgeRepo)

	handlers := &v1.Handlers{
		User:  v1.NewUserHandler(userService),
		Graph: v1.NewGraphHandler(graphService),
		Node:  v1.NewNodeHandler(nodeService),
		Edge:  v1.NewEdgeHandler(edgeService),
	}

	router := gin.Default()
	router.Use(cors.Default())

	apiV1 := router.Group("/api/v1")
	v1.RegisterRoutes(apiV1, handlers)

	log.Println("starting server on :8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
