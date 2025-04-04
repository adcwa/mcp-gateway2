package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wangfeng/mcp-gateway2/internal/api"
	"github.com/wangfeng/mcp-gateway2/internal/repository"
	"github.com/wangfeng/mcp-gateway2/pkg/mcp"
	"github.com/wangfeng/mcp-gateway2/pkg/models"
)

const (
	defaultPort = "8080"
	wasmDir     = "./wasm"
)

func main() {
	// Set up context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create the wasm directory if it doesn't exist
	if err := os.MkdirAll(wasmDir, 0755); err != nil {
		log.Fatalf("Failed to create wasm directory: %v", err)
	}

	// Initialize repositories
	httpRepo := repository.NewInMemoryHTTPInterfaceRepository()
	mcpRepo := repository.NewInMemoryMCPServerRepository()

	// Initialize MCP service
	mcpService, err := mcp.NewMCPService(wasmDir)
	if err != nil {
		log.Fatalf("Failed to initialize MCP service: %v", err)
	}
	defer mcpService.Close(ctx)

	// Initialize API handlers
	httpHandler := api.NewHTTPInterfaceHandler(httpRepo)
	mcpHandler := api.NewMCPServerHandler(mcpRepo, httpRepo, mcpService)

	// Set up Gin router
	router := gin.Default()

	// Register API routes
	httpHandler.RegisterRoutes(router)
	mcpHandler.RegisterRoutes(router)

	// Create a basic index page
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Welcome to MCP Gateway",
			"version": "1.0.0",
		})
	})

	// Add API version endpoint
	router.GET("/api/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"version": "1.0.0",
			"name":    "MCP Gateway",
		})
	})

	// Add a simple health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "UP",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	// Pre-add some example HTTP interfaces for testing
	addExampleHTTPInterfaces(ctx, httpRepo)

	// Determine port to listen on
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	// Start the server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: router,
	}

	// Run the server in a separate goroutine
	go func() {
		log.Printf("Server starting on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Set up graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited properly")
}

// addExampleHTTPInterfaces adds some example HTTP interfaces for testing
func addExampleHTTPInterfaces(ctx context.Context, repo repository.HTTPInterfaceRepository) {
	// Example 1: Random User API
	randomUserAPI := &models.HTTPInterface{
		Name:        "get-user",
		Description: "Get random user information",
		Method:      "GET",
		Path:        "https://randomuser.me/api/",
		Headers:     []models.Header{},
		Parameters:  []models.Param{},
		Responses: []models.Response{
			{
				StatusCode:  200,
				Description: "Random user information",
				Body: &models.Body{
					ContentType: "application/json",
					Schema:      `{"type": "object"}`,
					Example:     `{"results": [{"name": {"first": "John", "last": "Doe"}, "email": "john.doe@example.com", "location": {"city": "New York", "country": "USA"}, "phone": "123-456-7890"}]}`,
				},
			},
		},
	}

	// Example 2: Weather API
	weatherAPI := &models.HTTPInterface{
		Name:        "get-weather",
		Description: "Get weather information for a location",
		Method:      "GET",
		Path:        "https://api.openweathermap.org/data/2.5/weather",
		Headers:     []models.Header{},
		Parameters: []models.Param{
			{
				Name:        "q",
				Description: "City name",
				In:          "query",
				Required:    true,
				Type:        "string",
			},
			{
				Name:        "appid",
				Description: "API key",
				In:          "query",
				Required:    true,
				Type:        "string",
			},
		},
		Responses: []models.Response{
			{
				StatusCode:  200,
				Description: "Weather information",
				Body: &models.Body{
					ContentType: "application/json",
					Schema:      `{"type": "object"}`,
					Example:     `{"weather": [{"main": "Clear", "description": "clear sky"}], "main": {"temp": 293.15, "humidity": 75}}`,
				},
			},
		},
	}

	// Add the examples
	if err := repo.Create(ctx, randomUserAPI); err != nil {
		log.Printf("Failed to add example HTTP interface: %v", err)
	}

	if err := repo.Create(ctx, weatherAPI); err != nil {
		log.Printf("Failed to add example HTTP interface: %v", err)
	}
}
