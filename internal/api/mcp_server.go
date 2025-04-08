package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/wangfeng/mcp-gateway2/internal/repository"
	"github.com/wangfeng/mcp-gateway2/pkg/mcp"
	"github.com/wangfeng/mcp-gateway2/pkg/models"
)

// MCPServerHandler handles API requests for MCP Servers
type MCPServerHandler struct {
	mcpRepo    repository.MCPServerRepository
	httpRepo   repository.HTTPInterfaceRepository
	mcpService *mcp.MCPService
}

// NewMCPServerHandler creates a new MCP Server handler
func NewMCPServerHandler(mcpRepo repository.MCPServerRepository, httpRepo repository.HTTPInterfaceRepository, mcpService *mcp.MCPService) *MCPServerHandler {
	return &MCPServerHandler{
		mcpRepo:    mcpRepo,
		httpRepo:   httpRepo,
		mcpService: mcpService,
	}
}

// RegisterRoutes registers the MCP Server API routes
func (h *MCPServerHandler) RegisterRoutes(router *gin.Engine) {
	mcpGroup := router.Group("/api/mcp-servers")
	{
		mcpGroup.GET("", h.GetAllMCPServers)
		mcpGroup.GET("/:id", h.GetMCPServer)
		mcpGroup.POST("", h.CreateMCPServer)
		mcpGroup.PUT("/:id", h.UpdateMCPServer)
		mcpGroup.DELETE("/:id", h.DeleteMCPServer)
		mcpGroup.GET("/:id/versions", h.GetMCPServerVersions)
		mcpGroup.GET("/:id/versions/:version", h.GetMCPServerByVersion)
		mcpGroup.POST("/:id/compile", h.CompileMCPServer)
		mcpGroup.POST("/:id/activate", h.ActivateMCPServer)
		mcpGroup.POST("/:id/tools/:tool", h.InvokeTool)
		mcpGroup.GET("/:id/http-interfaces", h.GetMCPServerHTTPInterfaces)
	}
}

// GetAllMCPServers returns all MCP Servers
func (h *MCPServerHandler) GetAllMCPServers(c *gin.Context) {
	servers, err := h.mcpRepo.GetAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, servers)
}

// GetMCPServer returns a specific MCP Server
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

// CreateMCPServer creates a new MCP Server from HTTP interfaces
type CreateMCPServerRequest struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description"`
	HTTPIDs     []string `json:"httpIds" binding:"required"`
}

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
	version := c.Param("version")
	versionInt := 0
	if _, err := fmt.Sscanf(version, "%d", &versionInt); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid version number"})
		return
	}

	server, err := h.mcpRepo.GetByVersion(c.Request.Context(), id, versionInt)
	if err != nil {
		if err == repository.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "MCP Server version not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, server)
}

// CompileMCPServer compiles an MCP Server to WebAssembly
func (h *MCPServerHandler) CompileMCPServer(c *gin.Context) {
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

	// Compile to WebAssembly
	wasmPath, err := h.mcpService.CompileToWasm(server)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to compile MCP Server: " + err.Error()})
		return
	}

	// Update WASM path
	if err := h.mcpRepo.UpdateWasmPath(c.Request.Context(), id, wasmPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "MCP Server compiled successfully", "wasmPath": wasmPath})
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

	// Ensure the server has been compiled
	if server.WasmPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "MCP Server not compiled yet"})
		return
	}

	// Load the WASM module
	if err := h.mcpService.LoadWasmModule(c.Request.Context(), server); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load WASM module: " + err.Error()})
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

	// Check if the server is active
	if server.Status != "active" {
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
		c.JSON(http.StatusNotFound, gin.H{"error": "Tool not found or not allowed"})
		return
	}

	// Get tool parameters
	var params map[string]interface{}
	if err := c.ShouldBindJSON(&params); err != nil {
		params = make(map[string]interface{})
	}

	// Execute the tool
	result, err := h.mcpService.HandleToolRequest(c.Request.Context(), id, toolName, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to execute tool: " + err.Error()})
		return
	}

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
