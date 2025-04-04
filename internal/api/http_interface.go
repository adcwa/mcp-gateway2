package api

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/wangfeng/mcp-gateway2/internal/repository"
	"github.com/wangfeng/mcp-gateway2/pkg/models"
)

// HTTPInterfaceHandler handles API requests for HTTP interfaces
type HTTPInterfaceHandler struct {
	repo repository.HTTPInterfaceRepository
}

// NewHTTPInterfaceHandler creates a new HTTP interface handler
func NewHTTPInterfaceHandler(repo repository.HTTPInterfaceRepository) *HTTPInterfaceHandler {
	return &HTTPInterfaceHandler{
		repo: repo,
	}
}

// RegisterRoutes registers the HTTP interface API routes
func (h *HTTPInterfaceHandler) RegisterRoutes(router *gin.Engine) {
	httpGroup := router.Group("/api/http-interfaces")
	{
		httpGroup.GET("", h.GetAllHTTPInterfaces)
		httpGroup.GET("/:id", h.GetHTTPInterface)
		httpGroup.POST("", h.CreateHTTPInterface)
		httpGroup.PUT("/:id", h.UpdateHTTPInterface)
		httpGroup.DELETE("/:id", h.DeleteHTTPInterface)
		httpGroup.GET("/:id/versions", h.GetHTTPInterfaceVersions)
		httpGroup.GET("/:id/versions/:version", h.GetHTTPInterfaceByVersion)
		httpGroup.GET("/:id/openapi", h.GetOpenAPI)
		httpGroup.POST("/from-curl", h.CreateFromCurl)
	}
}

// GetAllHTTPInterfaces returns all HTTP interfaces
func (h *HTTPInterfaceHandler) GetAllHTTPInterfaces(c *gin.Context) {
	interfaces, err := h.repo.GetAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, interfaces)
}

// GetHTTPInterface returns a specific HTTP interface
func (h *HTTPInterfaceHandler) GetHTTPInterface(c *gin.Context) {
	id := c.Param("id")
	httpInterface, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == repository.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "HTTP interface not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, httpInterface)
}

// CreateHTTPInterface creates a new HTTP interface
func (h *HTTPInterfaceHandler) CreateHTTPInterface(c *gin.Context) {
	var httpInterface models.HTTPInterface
	if err := c.ShouldBindJSON(&httpInterface); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.repo.Create(c.Request.Context(), &httpInterface); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, httpInterface)
}

// UpdateHTTPInterface updates an HTTP interface
func (h *HTTPInterfaceHandler) UpdateHTTPInterface(c *gin.Context) {
	id := c.Param("id")
	var httpInterface models.HTTPInterface
	if err := c.ShouldBindJSON(&httpInterface); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Ensure ID matches
	httpInterface.ID = id

	if err := h.repo.Update(c.Request.Context(), &httpInterface); err != nil {
		if err == repository.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "HTTP interface not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, httpInterface)
}

// DeleteHTTPInterface deletes an HTTP interface
func (h *HTTPInterfaceHandler) DeleteHTTPInterface(c *gin.Context) {
	id := c.Param("id")
	if err := h.repo.Delete(c.Request.Context(), id); err != nil {
		if err == repository.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "HTTP interface not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetHTTPInterfaceVersions returns all versions of an HTTP interface
func (h *HTTPInterfaceHandler) GetHTTPInterfaceVersions(c *gin.Context) {
	id := c.Param("id")
	versions, err := h.repo.GetVersions(c.Request.Context(), id)
	if err != nil {
		if err == repository.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "HTTP interface not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, versions)
}

// GetHTTPInterfaceByVersion returns a specific version of an HTTP interface
func (h *HTTPInterfaceHandler) GetHTTPInterfaceByVersion(c *gin.Context) {
	id := c.Param("id")
	version := c.Param("version")
	versionInt := 0
	if _, err := fmt.Sscanf(version, "%d", &versionInt); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid version number"})
		return
	}

	httpInterface, err := h.repo.GetByVersion(c.Request.Context(), id, versionInt)
	if err != nil {
		if err == repository.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "HTTP interface version not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, httpInterface)
}

// GetOpenAPI returns the OpenAPI specification for an HTTP interface
func (h *HTTPInterfaceHandler) GetOpenAPI(c *gin.Context) {
	id := c.Param("id")
	httpInterface, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == repository.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "HTTP interface not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	openAPI := httpInterface.ConvertToOpenAPI()
	c.JSON(http.StatusOK, openAPI)
}

// CurlCommand represents a curl command to be converted to an HTTP interface
type CurlCommand struct {
	Command     string `json:"command" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

// CreateFromCurl creates a new HTTP interface from a curl command
func (h *HTTPInterfaceHandler) CreateFromCurl(c *gin.Context) {
	var curlCmd CurlCommand
	if err := c.ShouldBindJSON(&curlCmd); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse the curl command
	httpInterface, err := parseCurlCommand(curlCmd.Command, curlCmd.Name, curlCmd.Description)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse curl command: " + err.Error()})
		return
	}

	// Persist the new interface
	if err := h.repo.Create(c.Request.Context(), httpInterface); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, httpInterface)
}

// parseCurlCommand parses a curl command and converts it to an HTTP interface
func parseCurlCommand(curlCmd, name, description string) (*models.HTTPInterface, error) {
	// Initialize HTTP interface
	httpInterface := &models.HTTPInterface{
		Name:        name,
		Description: description,
		Headers:     []models.Header{},
		Parameters:  []models.Param{},
	}

	// Clean up the command (remove "curl" prefix if present)
	curlCmd = strings.TrimSpace(curlCmd)
	if strings.HasPrefix(curlCmd, "curl ") {
		curlCmd = strings.TrimPrefix(curlCmd, "curl ")
	}

	// Extract method
	methodRegex := regexp.MustCompile(`-X\s+([A-Z]+)`)
	if methodMatch := methodRegex.FindStringSubmatch(curlCmd); len(methodMatch) > 1 {
		httpInterface.Method = methodMatch[1]
	} else {
		// Default to GET if no method specified
		httpInterface.Method = "GET"
		// Check if there's a data flag, which implies POST
		if strings.Contains(curlCmd, " -d ") || strings.Contains(curlCmd, " --data ") {
			httpInterface.Method = "POST"
		}
	}

	// Extract URL
	urlRegex := regexp.MustCompile(`[^-]('|")?(https?://[^'"]+)('|")?`)
	if urlMatch := urlRegex.FindStringSubmatch(curlCmd); len(urlMatch) > 2 {
		httpInterface.Path = urlMatch[2]
	} else {
		return nil, fmt.Errorf("no URL found in curl command")
	}

	// Extract headers
	headerRegex := regexp.MustCompile(`-H\s+['"]([^:]+):\s*([^'"]+)['"]`)
	headerMatches := headerRegex.FindAllStringSubmatch(curlCmd, -1)
	for _, match := range headerMatches {
		if len(match) > 2 {
			header := models.Header{
				Name:        match[1],
				Description: "",
				Required:    true,
				Type:        "string",
			}
			httpInterface.Headers = append(httpInterface.Headers, header)
		}
	}

	// Extract data/body if present
	dataRegex := regexp.MustCompile(`-d\s+['"]([^'"]+)['"]`)
	if dataMatch := dataRegex.FindStringSubmatch(curlCmd); len(dataMatch) > 1 {
		// Check if data is JSON
		if strings.HasPrefix(dataMatch[1], "{") && strings.HasSuffix(dataMatch[1], "}") {
			httpInterface.RequestBody = &models.Body{
				ContentType: "application/json",
				Schema:      `{"type": "object"}`,
				Example:     dataMatch[1],
			}
		} else {
			httpInterface.RequestBody = &models.Body{
				ContentType: "application/x-www-form-urlencoded",
				Schema:      `{"type": "string"}`,
				Example:     dataMatch[1],
			}
		}
	}

	// Add a default response
	httpInterface.Responses = []models.Response{
		{
			StatusCode:  200,
			Description: "Successful response",
			Body: &models.Body{
				ContentType: "application/json",
				Schema:      `{"type": "object"}`,
			},
		},
	}

	return httpInterface, nil
}
