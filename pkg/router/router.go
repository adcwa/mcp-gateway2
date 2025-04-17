package router

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/wangfeng/mcp-gateway2/internal/repository"
	"github.com/wangfeng/mcp-gateway2/pkg/mcp"
	"github.com/wangfeng/mcp-gateway2/pkg/models"
)

// MCPServerRouter handles routing requests to MCP servers
type MCPServerRouter struct {
	mcpRepo    repository.MCPServerRepository
	mcpService *mcp.MCPService
}

// NewMCPServerRouter creates a new MCP server router
func NewMCPServerRouter(mcpRepo repository.MCPServerRepository, mcpService *mcp.MCPService) *MCPServerRouter {
	return &MCPServerRouter{
		mcpRepo:    mcpRepo,
		mcpService: mcpService,
	}
}

// RegisterRoutes registers the routes for MCP servers
func (r *MCPServerRouter) RegisterRoutes(router *gin.Engine) {
	// Main MCP server endpoint for dynamic routing by server name
	mcpServerGroup := router.Group("/router/mcp-servers")
	mcpServerGroup.Any("/:name/*path", r.HandleMCPServerByNameRequest)
}

// HandleMCPServerByNameRequest handles all requests to MCP servers by their name
func (r *MCPServerRouter) HandleMCPServerByNameRequest(c *gin.Context) {
	serverName := c.Param("name")
	path := c.Param("path")

	fmt.Printf("INFO: Handling MCP server request by name: server=%s, path=%s\n", serverName, path)

	// Get all MCP servers
	servers, err := r.mcpRepo.GetAll(c.Request.Context())
	if err != nil {
		fmt.Printf("ERROR: Failed to get MCP servers: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// Find the server by name
	var targetServer *models.MCPServer
	for _, server := range servers {
		if server.Name == serverName {
			copyServer := server // Create a copy to avoid issues with loop variable
			targetServer = &copyServer
			break
		}
	}

	if targetServer == nil {
		fmt.Printf("ERROR: MCP server not found: %s\n", serverName)
		c.JSON(http.StatusNotFound, gin.H{"error": "MCP server not found"})
		return
	}

	// Check if server is active
	if targetServer.Status != "active" {
		fmt.Printf("ERROR: MCP server is not active: %s, status=%s\n", serverName, targetServer.Status)
		c.JSON(http.StatusBadRequest, gin.H{"error": "MCP server is not active"})
		return
	}

	// Register server with MCP service if not already registered
	server, err := r.mcpRepo.GetByID(c.Request.Context(), targetServer.ID)
	if err != nil {
		fmt.Printf("ERROR: Failed to get MCP server: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	err = r.mcpService.RegisterServer(server)
	if err != nil {
		fmt.Printf("ERROR: Failed to register server with MCP service: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register server"})
		return
	}

	// Handle request based on path
	// Remove leading slash from path
	if path != "" && path[0] == '/' {
		path = path[1:]
	}

	// Route the request based on path
	if path == "tools" && c.Request.Method == http.MethodGet {
		// Handle get tools request
		r.handleGetTools(c, server)
	} else if path == "resources" && c.Request.Method == http.MethodGet {
		// Handle get resources request
		r.handleGetResources(c, server)
	} else if path == "prompts" && c.Request.Method == http.MethodGet {
		// Handle get prompts request
		r.handleGetPrompts(c, server)
	} else if strings.HasPrefix(path, "tools/") && c.Request.Method == http.MethodPost {
		// Handle tool invocation
		toolName := strings.TrimPrefix(path, "tools/")
		r.handleToolInvocation(c, server, toolName)
	} else {
		// Unknown path
		fmt.Printf("ERROR: Unknown path: %s\n", path)
		c.JSON(http.StatusNotFound, gin.H{"error": "Unknown path"})
	}
}

// handleGetTools handles requests to get tools metadata
func (r *MCPServerRouter) handleGetTools(c *gin.Context, server *models.MCPServer) {
	// Format tools according to MCP protocol specification
	toolsResponse := make([]map[string]interface{}, 0, len(server.Tools))
	for _, tool := range server.Tools {
		// Create a properties map for the parameters
		properties := make(map[string]interface{})
		required := []string{}

		// Extract parameters from URL path
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

// handleGetResources handles requests to get resources metadata
func (r *MCPServerRouter) handleGetResources(c *gin.Context, server *models.MCPServer) {
	// For now, return an empty resources array as placeholder
	c.JSON(http.StatusOK, []map[string]interface{}{})
}

// handleGetPrompts handles requests to get prompts metadata
func (r *MCPServerRouter) handleGetPrompts(c *gin.Context, server *models.MCPServer) {
	// For now, return an empty prompts array as placeholder
	c.JSON(http.StatusOK, []map[string]interface{}{})
}

// handleToolInvocation handles tool invocation requests
func (r *MCPServerRouter) handleToolInvocation(c *gin.Context, server *models.MCPServer, toolName string) {
	fmt.Printf("INFO: Handling tool invocation: server=%s, tool=%s\n", server.Name, toolName)

	// Check if the tool exists and is allowed
	toolExists := false
	for _, allowed := range server.AllowTools {
		if allowed == toolName {
			toolExists = true
			break
		}
	}

	if !toolExists {
		fmt.Printf("ERROR: Tool not found or not allowed: %s\n", toolName)
		c.JSON(http.StatusNotFound, gin.H{"error": "Tool not found or not allowed"})
		return
	}

	// Get tool parameters
	var params map[string]interface{}
	if err := c.ShouldBindJSON(&params); err != nil {
		fmt.Printf("WARNING: Could not parse request body, using empty params: %v\n", err)
		params = make(map[string]interface{})
	} else {
		fmt.Printf("INFO: Parsed parameters: %v\n", params)
	}

	// Execute the tool
	fmt.Printf("INFO: Executing tool: server=%s, tool=%s\n", server.Name, toolName)
	result, err := r.mcpService.HandleToolRequest(c.Request.Context(), server.ID, toolName, params)
	if err != nil {
		fmt.Printf("ERROR: Failed to execute tool: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to execute tool: " + err.Error()})
		return
	}

	fmt.Printf("INFO: Tool executed successfully\n")

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
