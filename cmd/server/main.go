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
	"github.com/wangfeng/mcp-gateway2/internal/db"
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

	// Initialize database connection
	// Set default config from environment variables or use defaults
	dbConfig := db.GetConfig()

	// Use PostgreSQL if environment variable is set
	usePostgresEnv := os.Getenv("USE_POSTGRES")
	usePostgres := usePostgresEnv == "" || usePostgresEnv == "true" || usePostgresEnv == "1"

	var httpRepo repository.HTTPInterfaceRepository
	var mcpRepo repository.MCPServerRepository

	if usePostgres {
		// Connect to PostgreSQL database
		database, err := db.ConnectDB()
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}
		defer database.Close()

		// PostgreSQL repositories
		pgHttpRepo := repository.NewPgHTTPInterfaceRepository(database)
		pgMcpRepo := repository.NewPgMCPServerRepository(database)

		// Initialize tables
		if err := pgHttpRepo.Initialize(ctx); err != nil {
			log.Fatalf("Failed to initialize HTTP interface repository: %v", err)
		}
		if err := pgMcpRepo.Initialize(ctx); err != nil {
			log.Fatalf("Failed to initialize MCP server repository: %v", err)
		}

		httpRepo = pgHttpRepo
		mcpRepo = pgMcpRepo

		log.Printf("Using PostgreSQL repositories: %s@%s:%s/%s",
			dbConfig.User, dbConfig.Host, dbConfig.Port, dbConfig.Database)
	} else {
		// In-memory repositories (for development)
		httpRepo = repository.NewInMemoryHTTPInterfaceRepository()
		mcpRepo = repository.NewInMemoryMCPServerRepository()
		log.Println("Using in-memory repositories")
	}

	// Initialize MCP service
	mcpService, err := mcp.NewMCPService(wasmDir)
	if err != nil {
		log.Fatalf("Failed to initialize MCP service: %v", err)
	}
	defer mcpService.Close(ctx)

	// Initialize API handlers
	httpHandler := api.NewHTTPInterfaceHandler(httpRepo)
	mcpHandler := api.NewMCPServerHandler(mcpRepo, httpRepo, mcpService)
	// wasmHandler := api.NewWasmFileHandler(mcpRepo, mcpService)

	// Set up Gin router
	router := gin.Default()

	// Add CORS middleware
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Register API routes
	httpHandler.RegisterRoutes(router)
	mcpHandler.RegisterRoutes(router)
	// wasmHandler.RegisterRoutes(router)

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
	// Only in development mode or if no interfaces exist
	if !usePostgres {
		addExampleHTTPInterfaces(ctx, httpRepo)
	} else {
		// Check if we have any interfaces
		interfaces, err := httpRepo.GetAll(ctx)
		if err != nil {
			log.Printf("Failed to check for existing interfaces: %v", err)
		} else if len(interfaces) == 0 {
			log.Println("No HTTP interfaces found, adding examples")
			addExampleHTTPInterfaces(ctx, httpRepo)
		}
	}

	// Add debug routes
	router.GET("/debug/routes", func(c *gin.Context) {
		routes := router.Routes()
		var routesList []string
		for _, route := range routes {
			routesList = append(routesList, fmt.Sprintf("%s %s", route.Method, route.Path))
		}
		c.JSON(http.StatusOK, routesList)
	})

	// Add database configuration info endpoint (for debugging)
	router.GET("/debug/db-config", func(c *gin.Context) {
		config := db.GetConfig()
		// Don't expose the password
		config.Password = "********"
		c.JSON(http.StatusOK, config)
	})

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
