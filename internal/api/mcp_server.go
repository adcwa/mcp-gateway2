package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/wangfeng/mcp-gateway2/internal/repository"
	"github.com/wangfeng/mcp-gateway2/pkg/mcp"
	"github.com/wangfeng/mcp-gateway2/pkg/models"
)

// MCPServerHandler handles HTTP requests for MCP servers
type MCPServerHandler struct {
	mcpRepo    repository.MCPServerRepository
	httpRepo   repository.HTTPInterfaceRepository
	mcpService *mcp.MCPService
}

// NewMCPServerHandler creates a new MCP server handler
func NewMCPServerHandler(mcpRepo repository.MCPServerRepository, httpRepo repository.HTTPInterfaceRepository, mcpService *mcp.MCPService) *MCPServerHandler {
	return &MCPServerHandler{
		mcpRepo:    mcpRepo,
		httpRepo:   httpRepo,
		mcpService: mcpService,
	}
}

// RegisterRoutes registers the routes for MCP servers
func (h *MCPServerHandler) RegisterRoutes(router *gin.Engine) {
	mcpGroup := router.Group("/api/mcp-servers")
	mcpGroup.GET("", h.GetAllMCPServers)
	mcpGroup.GET("/:id", h.GetMCPServer)
	mcpGroup.POST("", h.CreateMCPServer)
	mcpGroup.PUT("/:id", h.UpdateMCPServer)
	mcpGroup.DELETE("/:id", h.DeleteMCPServer)
	mcpGroup.GET("/:id/versions", h.GetMCPServerVersions)
	mcpGroup.GET("/:id/versions/:version", h.GetMCPServerByVersion)
	mcpGroup.POST("/:id/register", h.RegisterMCPServer)
	mcpGroup.POST("/:id/activate", h.ActivateMCPServer)
	mcpGroup.POST("/:id/tools/:tool", h.InvokeTool)
	mcpGroup.GET("/:id/http-interfaces", h.GetMCPServerHTTPInterfaces)
}

// GetAllMCPServers returns all MCP servers
func (h *MCPServerHandler) GetAllMCPServers(c *gin.Context) {
	servers, err := h.mcpRepo.GetAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, servers)
}

// GetMCPServer returns a single MCP server
func (h *MCPServerHandler) GetMCPServer(c *gin.Context) {
	id := c.Param("id")
	server, err := h.mcpRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == repository.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "MCP Server not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, server)
}

// CreateMCPServerRequest is the request for creating a new MCP Server
type CreateMCPServerRequest struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description"`
	HTTPIDs     []string `json:"httpIds" binding:"required"`
}

// CreateMCPServer creates a new MCP Server from HTTP interfaces
func (h *MCPServerHandler) CreateMCPServer(c *gin.Context) {
	var req CreateMCPServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get HTTP interfaces
	httpInterfaces := make([]models.HTTPInterface, 0, len(req.HTTPIDs))
	for _, id := range req.HTTPIDs {
		httpInterface, err := h.httpRepo.GetByID(c.Request.Context(), id)
		if err != nil {
			if err == repository.ErrNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "HTTP interface not found: " + id})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		httpInterfaces = append(httpInterfaces, *httpInterface)
	}

	// Create MCP Server
	mcpServer := models.NewMCPServerFromHTTPInterfaces(req.Name, req.Description, httpInterfaces)

	// Persist in repository
	if err := h.mcpRepo.Create(c.Request.Context(), mcpServer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, mcpServer)
}

// UpdateMCPServer updates an MCP Server
func (h *MCPServerHandler) UpdateMCPServer(c *gin.Context) {
	id := c.Param("id")
	var server models.MCPServer
	if err := c.ShouldBindJSON(&server); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Ensure ID matches
	server.ID = id

	// Update in repository
	if err := h.mcpRepo.Update(c.Request.Context(), &server); err != nil {
		if err == repository.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "MCP Server not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, server)
}

// DeleteMCPServer deletes an MCP Server
func (h *MCPServerHandler) DeleteMCPServer(c *gin.Context) {
	id := c.Param("id")
	if err := h.mcpRepo.Delete(c.Request.Context(), id); err != nil {
		if err == repository.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "MCP Server not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetMCPServerVersions returns all versions of an MCP Server
func (h *MCPServerHandler) GetMCPServerVersions(c *gin.Context) {
	id := c.Param("id")
	versions, err := h.mcpRepo.GetVersions(c.Request.Context(), id)
	if err != nil {
		if err == repository.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "MCP Server not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, versions)
}

// GetMCPServerByVersion returns a specific version of an MCP Server
func (h *MCPServerHandler) GetMCPServerByVersion(c *gin.Context) {
	id := c.Param("id")
	versionStr := c.Param("version")
	version, err := strconv.Atoi(versionStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid version number"})
		return
	}

	server, err := h.mcpRepo.GetByVersion(c.Request.Context(), id, version)
	if err != nil {
		if err == repository.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "MCP Server or version not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, server)
}

// RegisterMCPServer registers an MCP Server with the service
func (h *MCPServerHandler) RegisterMCPServer(c *gin.Context) {
	id := c.Param("id")

	// Get MCP Server
	server, err := h.mcpRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == repository.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "MCP Server not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Register with the MCP service
	if err := h.mcpService.RegisterServer(server); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register MCP Server: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "MCP Server registered successfully"})
}

// ActivateMCPServer activates an MCP Server
func (h *MCPServerHandler) ActivateMCPServer(c *gin.Context) {
	id := c.Param("id")

	// Get MCP Server
	server, err := h.mcpRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == repository.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "MCP Server not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Register with the MCP service if not already registered
	if err := h.mcpService.RegisterServer(server); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register MCP Server: " + err.Error()})
		return
	}

	// Update status
	if err := h.mcpRepo.UpdateStatus(c.Request.Context(), id, "active"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "MCP Server activated successfully"})
}

// InvokeTool invokes a tool in an MCP Server
func (h *MCPServerHandler) InvokeTool(c *gin.Context) {
	id := c.Param("id")
	toolName := c.Param("tool")

	fmt.Printf("INFO: Processing tool invocation request: server=%s, tool=%s\n", id, toolName)

	// Get MCP Server
	server, err := h.mcpRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == repository.ErrNotFound {
			fmt.Printf("ERROR: MCP Server not found: id=%s\n", id)
			c.JSON(http.StatusNotFound, gin.H{"error": "MCP Server not found"})
			return
		}
		fmt.Printf("ERROR: Failed to get MCP server: id=%s, error=%v\n", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Check if the server is active
	if server.Status != "active" {
		fmt.Printf("ERROR: MCP Server is not active: id=%s, status=%s\n", id, server.Status)
		c.JSON(http.StatusBadRequest, gin.H{"error": "MCP Server is not active"})
		return
	}

	// Check if the tool exists
	toolExists := false
	for _, allowed := range server.AllowTools {
		if allowed == toolName {
			toolExists = true
			break
		}
	}
	if !toolExists {
		fmt.Printf("ERROR: Tool not found or not allowed: server=%s, tool=%s\n", id, toolName)
		c.JSON(http.StatusNotFound, gin.H{"error": "Tool not found or not allowed"})
		return
	}

	// IMPORTANT: Register the server with the MCP service if it's not already registered
	// This ensures the server is available in the MCP service's in-memory map
	fmt.Printf("INFO: Ensuring server is registered with MCP service: id=%s\n", id)
	err = h.mcpService.RegisterServer(server)
	if err != nil {
		fmt.Printf("ERROR: Failed to register server with MCP service: id=%s, error=%v\n", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register server: " + err.Error()})
		return
	}

	// Get tool parameters
	var params map[string]interface{}
	if err := c.ShouldBindJSON(&params); err != nil {
		fmt.Printf("WARNING: Could not parse request body, using empty params: error=%v\n", err)
		params = make(map[string]interface{})
	} else {
		fmt.Printf("INFO: Parsed parameters: %v\n", params)
	}

	// Execute the tool
	fmt.Printf("INFO: Executing tool request: server=%s, tool=%s\n", id, toolName)
	result, err := h.mcpService.HandleToolRequest(c.Request.Context(), id, toolName, params)
	if err != nil {
		fmt.Printf("ERROR: Failed to execute tool: server=%s, tool=%s, error=%v\n", id, toolName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to execute tool: " + err.Error()})
		return
	}

	fmt.Printf("INFO: Tool executed successfully: server=%s, tool=%s\n", id, toolName)

	// Try to parse result as JSON
	var jsonResult interface{}
	if json.Valid([]byte(result)) {
		if err := json.Unmarshal([]byte(result), &jsonResult); err == nil {
			fmt.Printf("INFO: Returning JSON result\n")
			c.JSON(http.StatusOK, jsonResult)
			return
		}
	}

	// If not valid JSON, return as text
	fmt.Printf("INFO: Returning text result\n")
	c.JSON(http.StatusOK, gin.H{"result": result})
}

// GetMCPServerHTTPInterfaces returns the HTTP interfaces used to create a specific MCP server
func (h *MCPServerHandler) GetMCPServerHTTPInterfaces(c *gin.Context) {
	id := c.Param("id")
	server, err := h.mcpRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == repository.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "MCP Server not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get all HTTP interfaces
	allInterfaces, err := h.httpRepo.GetAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Filter interfaces that match the tools in the MCP server
	var matchedInterfaces []models.HTTPInterface
	for _, httpInterface := range allInterfaces {
		for _, tool := range server.Tools {
			if tool.Name == httpInterface.Name &&
				tool.RequestTemplate.Method == httpInterface.Method &&
				tool.RequestTemplate.URL == httpInterface.Path {
				matchedInterfaces = append(matchedInterfaces, httpInterface)
				break
			}
		}
	}

	c.JSON(http.StatusOK, matchedInterfaces)
}
