package api

import (
	"fmt"
	"net/http"

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
