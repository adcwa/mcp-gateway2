package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/wangfeng/mcp-gateway2/internal/repository"
	"github.com/wangfeng/mcp-gateway2/pkg/mcp"
	"github.com/wangfeng/mcp-gateway2/pkg/models"
)

// Create a new MCPServerValidator interface for validation logic
type MCPServerValidator interface {
	ValidateName(ctx context.Context, name string, excludeID string) error
}

// Implement the validator with repository access
type MCPServerValidatorImpl struct {
	repo repository.MCPServerRepository
}

// NewMCPServerValidator creates a new validator
func NewMCPServerValidator(repo repository.MCPServerRepository) MCPServerValidator {
	return &MCPServerValidatorImpl{repo: repo}
}

// ValidateName checks if the name is already taken by another server
func (v *MCPServerValidatorImpl) ValidateName(ctx context.Context, name string, excludeID string) error {
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}

	servers, err := v.repo.GetAll(ctx)
	if err != nil {
		return err
	}

	for _, server := range servers {
		if server.Name == name && server.ID != excludeID {
			return fmt.Errorf("MCP server with name '%s' already exists", name)
		}
	}

	return nil
}

// MCPServerHandler handles HTTP requests for MCP servers
type MCPServerHandler struct {
	mcpRepo    repository.MCPServerRepository
	httpRepo   repository.HTTPInterfaceRepository
	mcpService *mcp.MCPService
	validator  MCPServerValidator
}

// NewMCPServerHandler creates a new MCP server handler
func NewMCPServerHandler(mcpRepo repository.MCPServerRepository, httpRepo repository.HTTPInterfaceRepository, mcpService *mcp.MCPService) *MCPServerHandler {
	return &MCPServerHandler{
		mcpRepo:    mcpRepo,
		httpRepo:   httpRepo,
		mcpService: mcpService,
		validator:  NewMCPServerValidator(mcpRepo),
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
	mcpGroup.POST("/:id/deactivate", h.DeactivateMCPServer)
	mcpGroup.POST("/:id/tools/:tool", h.InvokeTool)
	mcpGroup.GET("/:id/http-interfaces", h.GetMCPServerHTTPInterfaces)
	mcpGroup.POST("/validate-name", h.ValidateMCPServerName)

	// Add new information endpoints
	mcpGroup.GET("/:id/metadata", h.GetMCPServerMetadata)
	mcpGroup.GET("/:id/usage-guide", h.GetMCPServerUsageGuide)
	mcpGroup.GET("/:id/client-examples", h.GetMCPServerClientExamples)

	// Add MCP protocol compliant endpoints
	mcpProtoGroup := router.Group("/api/mcp-server/:name")
	mcpProtoGroup.GET("/tools", h.GetMCPServerTools)
	mcpProtoGroup.GET("/resources", h.GetMCPServerResources)
	mcpProtoGroup.GET("/prompts", h.GetMCPServerPrompts)

	// Add dynamic routing for tools invocation through MCP protocol
	mcpProtoGroup.POST("/tools/:tool", h.InvokeToolMCP)
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

// ValidateNameRequest is the request for validating a MCP server name
type ValidateNameRequest struct {
	Name      string `json:"name" binding:"required"`
	ExcludeID string `json:"excludeId"` // Optional, used when updating
}

// ValidateMCPServerName validates if a name is available for use
func (h *MCPServerHandler) ValidateMCPServerName(c *gin.Context) {
	var req ValidateNameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.validator.ValidateName(c.Request.Context(), req.Name, req.ExcludeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "valid": false})
		return
	}

	c.JSON(http.StatusOK, gin.H{"valid": true})
}

// CreateMCPServer creates a new MCP Server from HTTP interfaces
func (h *MCPServerHandler) CreateMCPServer(c *gin.Context) {
	var req CreateMCPServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate server name uniqueness
	if err := h.validator.ValidateName(c.Request.Context(), req.Name, ""); err != nil {
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

	// Get the existing server to check for name changes
	existingServer, err := h.mcpRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == repository.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "MCP Server not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Only validate name if it has changed
	if existingServer.Name != server.Name {
		if err := h.validator.ValidateName(c.Request.Context(), server.Name, id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

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

// DeactivateMCPServer deactivates an MCP Server
func (h *MCPServerHandler) DeactivateMCPServer(c *gin.Context) {
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

	// Check if server is already inactive
	if server.Status != "active" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "MCP Server is not active"})
		return
	}

	// Update status to inactive
	if err := h.mcpRepo.UpdateStatus(c.Request.Context(), id, "inactive"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "MCP Server deactivated successfully"})
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

// GetMCPServerTools provides tool metadata conforming to MCP protocol
func (h *MCPServerHandler) GetMCPServerTools(c *gin.Context) {
	name := c.Param("name")

	// Get MCP Server
	server, err := h.mcpRepo.GetByName(c.Request.Context(), name)
	if err != nil {
		if err == repository.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "MCP Server not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Check if server is active
	if server.Status != "active" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "MCP Server is not active"})
		return
	}

	// Format tools according to MCP protocol specification
	toolsResponse := make([]map[string]interface{}, 0, len(server.Tools))
	for _, tool := range server.Tools {
		// Create a properties map for the parameters
		properties := make(map[string]interface{})
		required := []string{}

		// Extract parameters from URL path
		// Look for parameters in the format {paramName}
		url := tool.RequestTemplate.URL
		urlParams := extractURLParams(url)
		for _, param := range urlParams {
			properties[param] = map[string]interface{}{
				"type":        "string",
				"description": fmt.Sprintf("Path parameter '%s'", param),
			}
			required = append(required, param)
		}

		// Add query parameters if they can be inferred
		// If using a POST/PUT method, add a generic "data" parameter
		if tool.RequestTemplate.Method == "POST" || tool.RequestTemplate.Method == "PUT" || tool.RequestTemplate.Method == "PATCH" {
			properties["data"] = map[string]interface{}{
				"type":        "object",
				"description": "Request body data",
			}
			required = append(required, "data")
		}

		// Define a schema object for the parameters
		toolDef := map[string]interface{}{
			"name":        tool.Name,
			"description": tool.Description,
			"parameters": map[string]interface{}{
				"type":       "object",
				"properties": properties,
				"required":   required,
			},
		}

		toolsResponse = append(toolsResponse, toolDef)
	}

	c.JSON(http.StatusOK, toolsResponse)
}

// extractURLParams extracts parameters from a URL path
// e.g. "/users/{id}/profile" would return ["id"]
func extractURLParams(url string) []string {
	params := []string{}
	parts := strings.Split(url, "/")

	for _, part := range parts {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			// Extract parameter name without braces
			paramName := part[1 : len(part)-1]
			params = append(params, paramName)
		}
	}

	return params
}

// GetMCPServerResources provides resources metadata conforming to MCP protocol
func (h *MCPServerHandler) GetMCPServerResources(c *gin.Context) {
	name := c.Param("name")

	// Get MCP Server
	server, err := h.mcpRepo.GetByName(c.Request.Context(), name)
	if err != nil {
		if err == repository.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "MCP Server not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Check if server is active
	if server.Status != "active" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "MCP Server is not active"})
		return
	}

	// For now, return an empty resources array as placeholder
	// This will be expanded in the future
	c.JSON(http.StatusOK, []map[string]interface{}{})
}

// GetMCPServerPrompts provides prompts metadata conforming to MCP protocol
func (h *MCPServerHandler) GetMCPServerPrompts(c *gin.Context) {
	name := c.Param("name")

	// Get MCP Server
	server, err := h.mcpRepo.GetByName(c.Request.Context(), name)
	if err != nil {
		if err == repository.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "MCP Server not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Check if server is active
	if server.Status != "active" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "MCP Server is not active"})
		return
	}

	// For now, return an empty prompts array as placeholder
	// This will be expanded in the future
	c.JSON(http.StatusOK, []map[string]interface{}{})
}

// InvokeToolMCP provides a MCP protocol compliant endpoint for invoking tools
func (h *MCPServerHandler) InvokeToolMCP(c *gin.Context) {
	name := c.Param("name")
	toolName := c.Param("tool")

	fmt.Printf("INFO: Processing MCP tool invocation request: server=%s, tool=%s\n", name, toolName)

	// Get MCP Server
	server, err := h.mcpRepo.GetByName(c.Request.Context(), name)
	if err != nil {
		if err == repository.ErrNotFound {
			fmt.Printf("ERROR: MCP Server not found: name=%s\n", name)
			c.JSON(http.StatusNotFound, gin.H{"error": "MCP Server not found"})
			return
		}
		fmt.Printf("ERROR: Failed to get MCP server: name=%s, error=%v\n", name, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Check if the server is active
	if server.Status != "active" {
		fmt.Printf("ERROR: MCP Server is not active: name=%s, status=%s\n", name, server.Status)
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
		fmt.Printf("ERROR: Tool not found or not allowed: server=%s, tool=%s\n", name, toolName)
		c.JSON(http.StatusNotFound, gin.H{"error": "Tool not found or not allowed"})
		return
	}

	// Ensure server is registered
	fmt.Printf("INFO: Ensuring server is registered with MCP service: name=%s\n", name)
	err = h.mcpService.RegisterServer(server)
	if err != nil {
		fmt.Printf("ERROR: Failed to register server with MCP service: name=%s, error=%v\n", name, err)
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
	fmt.Printf("INFO: Executing tool request via MCP: server=%s, tool=%s\n", name, toolName)
	result, err := h.mcpService.HandleToolRequest(c.Request.Context(), server.ID, toolName, params)
	if err != nil {
		fmt.Printf("ERROR: Failed to execute tool: server=%s, tool=%s, error=%v\n", name, toolName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to execute tool: " + err.Error()})
		return
	}

	fmt.Printf("INFO: Tool executed successfully: server=%s, tool=%s\n", name, toolName)

	// Format the response according to MCP protocol
	// Try to parse result as JSON
	var jsonResult interface{}
	if json.Valid([]byte(result)) {
		if err := json.Unmarshal([]byte(result), &jsonResult); err == nil {
			c.JSON(http.StatusOK, jsonResult)
			return
		}
	}

	// If not valid JSON, return as text
	c.JSON(http.StatusOK, gin.H{"text": result})
}

// GetMCPServerMetadata returns detailed metadata about an MCP server
func (h *MCPServerHandler) GetMCPServerMetadata(c *gin.Context) {
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

	// Format according to MCP protocol specifications
	metadata := map[string]interface{}{
		"id":             server.ID,
		"name":           server.Name,
		"description":    server.Description,
		"version":        server.Version,
		"status":         server.Status,
		"mcp_compliance": "2025-03-26", // MCP specification version
		"endpoints": map[string]string{
			"tools":     fmt.Sprintf("/api/mcp-server/%s/tools", server.Name),
			"resources": fmt.Sprintf("/api/mcp-server/%s/resources", server.Name),
			"prompts":   fmt.Sprintf("/api/mcp-server/%s/prompts", server.Name),
		},
		"capabilities": map[string]interface{}{
			"tools":     !isEmpty(server.Tools),
			"resources": false, // Not implemented yet
			"prompts":   false, // Not implemented yet
		},
		"created_at": server.CreatedAt,
		"updated_at": server.UpdatedAt,
	}

	// Add tools summary
	toolsSummary := make([]map[string]interface{}, 0, len(server.Tools))
	for _, tool := range server.Tools {
		toolsSummary = append(toolsSummary, map[string]interface{}{
			"name":        tool.Name,
			"description": tool.Description,
			"method":      tool.RequestTemplate.Method,
			"url":         tool.RequestTemplate.URL,
		})
	}
	metadata["tools_summary"] = toolsSummary

	c.JSON(http.StatusOK, metadata)
}

// GetMCPServerUsageGuide returns a comprehensive usage guide for an MCP server
func (h *MCPServerHandler) GetMCPServerUsageGuide(c *gin.Context) {
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

	// Generate a comprehensive usage guide
	guide := map[string]interface{}{
		"server_name":        server.Name,
		"server_description": server.Description,
		"overview": fmt.Sprintf(
			"This MCP Server provides %d tools that can be accessed using the Model Context Protocol standard. "+
				"The server endpoint is available at /api/mcp-server/%s/",
			len(server.Tools),
			server.Name,
		),
		"tools_usage": generateToolsUsageGuide(server),
		"mcp_protocol_info": map[string]interface{}{
			"specification_url": "https://modelcontextprotocol.io/specification/2025-03-26/",
			"server_endpoints": map[string]string{
				"tools_metadata":     fmt.Sprintf("/api/mcp-server/%s/tools", server.Name),
				"resources_metadata": fmt.Sprintf("/api/mcp-server/%s/resources", server.Name),
				"prompts_metadata":   fmt.Sprintf("/api/mcp-server/%s/prompts", server.Name),
				"tool_invocation":    fmt.Sprintf("/api/mcp-server/%s/tools/{tool_name}", server.Name),
			},
			"request_format": map[string]interface{}{
				"content_type": "application/json",
				"parameters":   "Tool-specific parameters according to the tool's schema",
			},
			"response_format": map[string]interface{}{
				"success":      "JSON or text response from the tool",
				"error":        "Error object with message",
				"content_type": "application/json",
			},
		},
		"integration_steps": []string{
			"1. Retrieve tool metadata from the /tools endpoint",
			"2. Examine tool requirements and parameters",
			"3. Call tool endpoints with appropriate parameters",
			"4. Process the tool response according to your application needs",
		},
	}

	c.JSON(http.StatusOK, guide)
}

// GetMCPServerClientExamples returns example client code for different languages
func (h *MCPServerHandler) GetMCPServerClientExamples(c *gin.Context) {
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

	baseUrl := c.Request.Host // Get the current host
	if baseUrl == "" {
		baseUrl = "localhost:8080" // Default if not available
	}

	if !strings.HasPrefix(baseUrl, "http") {
		// Add protocol if not present
		scheme := "http"
		if c.Request.TLS != nil {
			scheme = "https"
		}
		baseUrl = scheme + "://" + baseUrl
	}

	// Generate example code for different programming languages
	examples := map[string]interface{}{
		"python":     generatePythonClientExample(server, baseUrl),
		"javascript": generateJavaScriptClientExample(server, baseUrl),
		"go":         generateGoClientExample(server, baseUrl),
		"java":       generateJavaClientExample(server, baseUrl),
	}

	c.JSON(http.StatusOK, examples)
}

// Helper functions for the new endpoints

// isEmpty checks if a slice is empty
func isEmpty(slice interface{}) bool {
	switch s := slice.(type) {
	case []models.Tool:
		return len(s) == 0
	case []string:
		return len(s) == 0
	default:
		return true
	}
}

// generateToolsUsageGuide creates a detailed guide for each tool
func generateToolsUsageGuide(server *models.MCPServer) []map[string]interface{} {
	guide := make([]map[string]interface{}, 0, len(server.Tools))

	for _, tool := range server.Tools {
		// Extract parameters from URL
		urlParams := extractURLParams(tool.RequestTemplate.URL)

		// Create parameter descriptions
		paramDescriptions := make([]map[string]interface{}, 0)
		for _, param := range urlParams {
			paramDescriptions = append(paramDescriptions, map[string]interface{}{
				"name":        param,
				"type":        "string",
				"description": fmt.Sprintf("Path parameter '%s'", param),
				"required":    true,
			})
		}

		// Add body parameter for POST/PUT methods
		if tool.RequestTemplate.Method == "POST" || tool.RequestTemplate.Method == "PUT" || tool.RequestTemplate.Method == "PATCH" {
			paramDescriptions = append(paramDescriptions, map[string]interface{}{
				"name":        "data",
				"type":        "object",
				"description": "Request body data",
				"required":    true,
			})
		}

		// Add example request
		exampleRequest := generateExampleRequest(tool)

		// Add example response (simplified)
		exampleResponse := ""
		if tool.ResponseTemplate.Body != "" {
			exampleResponse = "Example response depends on the external API response templated with: " +
				truncateString(tool.ResponseTemplate.Body, 100)
		} else {
			exampleResponse = "{\"result\": \"Example response would appear here\"}"
		}

		// Compile the tool usage guide
		toolGuide := map[string]interface{}{
			"name":             tool.Name,
			"description":      tool.Description,
			"endpoint":         fmt.Sprintf("/api/mcp-server/%s/tools/%s", server.Name, tool.Name),
			"method":           "POST", // MCP always uses POST for tool invocation
			"parameters":       paramDescriptions,
			"example_request":  exampleRequest,
			"example_response": exampleResponse,
			"notes": []string{
				"All tools are invoked via POST request regardless of the underlying HTTP method",
				"Parameters should be passed as a JSON object in the request body",
				"Path parameters from the tool URL should be included in the request body",
			},
		}

		guide = append(guide, toolGuide)
	}

	return guide
}

// generateExampleRequest creates an example request body for a tool
func generateExampleRequest(tool models.Tool) string {
	// Extract parameters and create an example request
	params := make(map[string]interface{})

	// Extract URL parameters
	urlParams := extractURLParams(tool.RequestTemplate.URL)
	for _, param := range urlParams {
		params[param] = fmt.Sprintf("<%s>", param)
	}

	// Add example body for POST/PUT
	if tool.RequestTemplate.Method == "POST" || tool.RequestTemplate.Method == "PUT" || tool.RequestTemplate.Method == "PATCH" {
		params["data"] = map[string]string{
			"example_field1": "value1",
			"example_field2": "value2",
		}
	}

	// Convert to JSON string
	jsonBytes, err := json.MarshalIndent(params, "", "  ")
	if err != nil {
		return "{}"
	}

	return string(jsonBytes)
}

// truncateString shortens a string with ellipsis if needed
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// generatePythonClientExample creates Python code to interact with the MCP server
func generatePythonClientExample(server *models.MCPServer, baseUrl string) string {
	// Create a sample tool to use in the example
	var sampleTool models.Tool
	if len(server.Tools) > 0 {
		sampleTool = server.Tools[0]
	} else {
		sampleTool = models.Tool{
			Name:        "example_tool",
			Description: "Example tool",
		}
	}

	return fmt.Sprintf(`
import requests
import json

class MCPClient:
    def __init__(self, base_url="%s"):
        self.base_url = base_url
        self.server_name = "%s"
        
    def get_tools(self):
        """Get the list of available tools on the MCP server"""
        url = f"{self.base_url}/api/mcp-server/{self.server_name}/tools"
        response = requests.get(url)
        if response.status_code == 200:
            return response.json()
        else:
            raise Exception(f"Failed to get tools: {response.text}")
    
    def invoke_tool(self, tool_name, parameters):
        """Invoke a tool on the MCP server"""
        url = f"{self.base_url}/api/mcp-server/{self.server_name}/tools/{tool_name}"
        response = requests.post(url, json=parameters)
        if response.status_code == 200:
            return response.json()
        else:
            raise Exception(f"Failed to invoke tool: {response.text}")

# Example usage
if __name__ == "__main__":
    # Create client instance
    client = MCPClient()
    
    # Get available tools
    tools = client.get_tools()
    print(f"Available tools: {json.dumps(tools, indent=2)}")
    
    # Invoke a specific tool
    try:
        # Example parameters - adjust based on the actual tool requirements
        params = {
            # Add required parameters for the '%s' tool
        }
        result = client.invoke_tool("%s", params)
        print(f"Tool result: {json.dumps(result, indent=2)}")
    except Exception as e:
        print(f"Error: {e}")
`, baseUrl, server.Name, sampleTool.Name, sampleTool.Name)
}

// generateJavaScriptClientExample creates JavaScript code to interact with the MCP server
func generateJavaScriptClientExample(server *models.MCPServer, baseUrl string) string {
	// Create a sample tool to use in the example
	var sampleTool models.Tool
	if len(server.Tools) > 0 {
		sampleTool = server.Tools[0]
	} else {
		sampleTool = models.Tool{
			Name:        "example_tool",
			Description: "Example tool",
		}
	}

	return fmt.Sprintf(`
// MCP Client using modern JavaScript with fetch API
class MCPClient {
  constructor(baseUrl = '%s') {
    this.baseUrl = baseUrl;
    this.serverName = '%s';
  }

  async getTools() {
    const url = this.baseUrl + '/api/mcp-server/' + this.serverName + '/tools';
    const response = await fetch(url);
    
    if (!response.ok) {
      throw new Error('Failed to get tools: ' + response.statusText);
    }
    
    return response.json();
  }

  async invokeTool(toolName, parameters) {
    const url = this.baseUrl + '/api/mcp-server/' + this.serverName + '/tools/' + toolName;
    const response = await fetch(url, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(parameters)
    });
    
    if (!response.ok) {
      throw new Error('Failed to invoke tool: ' + response.statusText);
    }
    
    return response.json();
  }
}

// Example usage
async function run() {
  try {
    // Create client instance
    const client = new MCPClient();
    
    // Get available tools
    const tools = await client.getTools();
    console.log('Available tools:', tools);
    
    // Invoke a specific tool
    const params = {
      // Add required parameters for the '%s' tool
    };
    
    const result = await client.invokeTool('%s', params);
    console.log('Tool result:', result);
  } catch (error) {
    console.error('Error:', error.message);
  }
}

// Run the example
run();
`, baseUrl, server.Name, sampleTool.Name, sampleTool.Name)
}

// generateGoClientExample creates Go code to interact with the MCP server
func generateGoClientExample(server *models.MCPServer, baseUrl string) string {
	// Create a sample tool to use in the example
	var sampleTool models.Tool
	if len(server.Tools) > 0 {
		sampleTool = server.Tools[0]
	} else {
		sampleTool = models.Tool{
			Name:        "example_tool",
			Description: "Example tool",
		}
	}

	return fmt.Sprintf(`
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// MCPClient provides a client for interacting with MCP servers
type MCPClient struct {
	BaseURL    string
	ServerName string
	Client     *http.Client
}

// NewMCPClient creates a new MCP client
func NewMCPClient(baseURL, serverName string) *MCPClient {
	return &MCPClient{
		BaseURL:    baseURL,
		ServerName: serverName,
		Client:     &http.Client{},
	}
}

// GetTools retrieves the list of available tools
func (c *MCPClient) GetTools() ([]map[string]interface{}, error) {
	url := fmt.Sprintf("%s/api/mcp-server/%s/tools", c.BaseURL, c.ServerName)
	
	resp, err := c.Client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get tools: %s", string(body))
	}
	
	var tools []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&tools); err != nil {
		return nil, err
	}
	
	return tools, nil
}

// InvokeTool invokes a tool on the MCP server
func (c *MCPClient) InvokeTool(toolName string, parameters map[string]interface{}) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/api/mcp-server/%s/tools/%s", c.BaseURL, c.ServerName, toolName)
	
	paramJSON, err := json.Marshal(parameters)
	if err != nil {
		return nil, err
	}
	
	resp, err := c.Client.Post(url, "application/json", bytes.NewBuffer(paramJSON))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to invoke tool: %s", string(body))
	}
	
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	
	return result, nil
}

func main() {
	// Create client instance
	client := NewMCPClient("%s", "%s")
	
	// Get available tools
	tools, err := client.GetTools()
	if err != nil {
		fmt.Printf("Error getting tools: %v\n", err)
		return
	}
	
	toolsJSON, _ := json.MarshalIndent(tools, "", "  ")
	fmt.Printf("Available tools: %s\n", string(toolsJSON))
	
	// Invoke a specific tool
	parameters := map[string]interface{}{
		// Add required parameters for the '%s' tool
	}
	
	result, err := client.InvokeTool("%s", parameters)
	if err != nil {
		fmt.Printf("Error invoking tool: %v\n", err)
		return
	}
	
	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	fmt.Printf("Tool result: %s\n", string(resultJSON))
}
`, baseUrl, server.Name, sampleTool.Name, sampleTool.Name)
}

// generateJavaClientExample creates Java code to interact with the MCP server
func generateJavaClientExample(server *models.MCPServer, baseUrl string) string {
	// Create a sample tool to use in the example
	var sampleTool models.Tool
	if len(server.Tools) > 0 {
		sampleTool = server.Tools[0]
	} else {
		sampleTool = models.Tool{
			Name:        "example_tool",
			Description: "Example tool",
		}
	}

	return fmt.Sprintf(`
import java.io.IOException;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.time.Duration;
import org.json.JSONArray;
import org.json.JSONObject;

public class MCPClient {
    private final String baseUrl;
    private final String serverName;
    private final HttpClient httpClient;

    public MCPClient(String baseUrl, String serverName) {
        this.baseUrl = baseUrl;
        this.serverName = serverName;
        this.httpClient = HttpClient.newBuilder()
                .version(HttpClient.Version.HTTP_2)
                .connectTimeout(Duration.ofSeconds(10))
                .build();
    }

    public JSONArray getTools() throws IOException, InterruptedException {
        String url = String.format("%s/api/mcp-server/%s/tools", baseUrl, serverName);
        
        HttpRequest request = HttpRequest.newBuilder()
                .GET()
                .uri(URI.create(url))
                .header("Accept", "application/json")
                .build();
                
        HttpResponse<String> response = httpClient.send(request, HttpResponse.BodyHandlers.ofString());
        
        if (response.statusCode() != 200) {
            throw new IOException("Failed to get tools: " + response.body());
        }
        
        return new JSONArray(response.body());
    }

    public JSONObject invokeTool(String toolName, JSONObject parameters) throws IOException, InterruptedException {
        String url = String.format("%s/api/mcp-server/%s/tools/%s", baseUrl, serverName, toolName);
        
        HttpRequest request = HttpRequest.newBuilder()
                .POST(HttpRequest.BodyPublishers.ofString(parameters.toString()))
                .uri(URI.create(url))
                .header("Content-Type", "application/json")
                .header("Accept", "application/json")
                .build();
                
        HttpResponse<String> response = httpClient.send(request, HttpResponse.BodyHandlers.ofString());
        
        if (response.statusCode() != 200) {
            throw new IOException("Failed to invoke tool: " + response.body());
        }
        
        return new JSONObject(response.body());
    }

    public static void main(String[] args) {
        try {
            // Create client instance
            MCPClient client = new MCPClient("%s", "%s");
            
            // Get available tools
            JSONArray tools = client.getTools();
            System.out.println("Available tools: " + tools.toString(2));
            
            // Invoke a specific tool
            JSONObject parameters = new JSONObject();
            // Add required parameters for the '%s' tool
            
            JSONObject result = client.invokeTool("%s", parameters);
            System.out.println("Tool result: " + result.toString(2));
            
        } catch (Exception e) {
            System.err.println("Error: " + e.getMessage());
            e.printStackTrace();
        }
    }
}
`, baseUrl, server.Name, sampleTool.Name, sampleTool.Name)
}
