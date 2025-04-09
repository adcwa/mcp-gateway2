package api

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/wangfeng/mcp-gateway2/internal/repository"
	"github.com/wangfeng/mcp-gateway2/pkg/mcp"
)

// WasmFileHandler handles API requests for WASM files
type WasmFileHandler struct {
	mcpRepo    repository.MCPServerRepository
	mcpService *mcp.MCPService
}

// NewWasmFileHandler creates a new WASM file handler
func NewWasmFileHandler(mcpRepo repository.MCPServerRepository, mcpService *mcp.MCPService) *WasmFileHandler {
	return &WasmFileHandler{
		mcpRepo:    mcpRepo,
		mcpService: mcpService,
	}
}

// RegisterRoutes registers the WASM file API routes
func (h *WasmFileHandler) RegisterRoutes(router *gin.Engine) {
	wasmGroup := router.Group("/api/wasm-files")
	{
		wasmGroup.GET("", h.GetAllWasmFiles)
		wasmGroup.GET("/:id", h.GetWasmFile)
		wasmGroup.DELETE("/:id", h.DeleteWasmFile)
		wasmGroup.GET("/:id/download", h.DownloadWasmFile)
	}
}

// GetAllWasmFiles returns all WASM files
func (h *WasmFileHandler) GetAllWasmFiles(c *gin.Context) {
	// This would normally retrieve all WASM files from a database
	// For this MVP, we'll scan the WASM directory for .wasm files
	wasmDir := h.mcpService.GetWasmDir()
	files, err := os.ReadDir(wasmDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read WASM directory: " + err.Error()})
		return
	}

	wasmFiles := []gin.H{}
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(strings.ToLower(file.Name()), ".wasm") {
			continue
		}

		fileInfo, err := file.Info()
		if err != nil {
			continue
		}

		wasmFiles = append(wasmFiles, gin.H{
			"id":        file.Name(), // Use filename as ID
			"name":      file.Name(),
			"path":      filepath.Join(wasmDir, file.Name()),
			"size":      fileInfo.Size(),
			"createdAt": fileInfo.ModTime(),
			"updatedAt": fileInfo.ModTime(),
		})
	}

	c.JSON(http.StatusOK, wasmFiles)
}

// GetWasmFile returns a specific WASM file
func (h *WasmFileHandler) GetWasmFile(c *gin.Context) {
	id := c.Param("id")

	// Validate that the file exists
	wasmDir := h.mcpService.GetWasmDir()
	filePath := filepath.Join(wasmDir, id)
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			c.JSON(http.StatusNotFound, gin.H{"error": "WASM file not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":        id,
		"name":      id,
		"path":      filePath,
		"size":      fileInfo.Size(),
		"createdAt": fileInfo.ModTime(),
		"updatedAt": fileInfo.ModTime(),
	})
}

// DeleteWasmFile deletes a WASM file
func (h *WasmFileHandler) DeleteWasmFile(c *gin.Context) {
	id := c.Param("id")

	// Validate that the file exists
	wasmDir := h.mcpService.GetWasmDir()
	filePath := filepath.Join(wasmDir, id)
	if _, err := os.Stat(filePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			c.JSON(http.StatusNotFound, gin.H{"error": "WASM file not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Delete the file
	if err := os.Remove(filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete WASM file: " + err.Error()})
		return
	}

	// Remove references to this WASM file in any MCPServers
	// This would require more complex database operations in a real implementation
	// For this MVP, we'll skip this step

	c.Status(http.StatusNoContent)
}

// DownloadWasmFile allows downloading a WASM file
func (h *WasmFileHandler) DownloadWasmFile(c *gin.Context) {
	id := c.Param("id")

	// Validate that the file exists
	wasmDir := h.mcpService.GetWasmDir()
	filePath := filepath.Join(wasmDir, id)
	if _, err := os.Stat(filePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			c.JSON(http.StatusNotFound, gin.H{"error": "WASM file not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Set appropriate headers
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Content-Disposition", "attachment; filename="+id)
	c.Header("Content-Type", "application/wasm")

	// Serve the file
	c.File(filePath)
}
