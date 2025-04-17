package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/wangfeng/mcp-gateway2/internal/repository"
	"github.com/wangfeng/mcp-gateway2/pkg/models"
	"gopkg.in/yaml.v3"
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
		httpGroup.GET("/:id/openapi", h.ExportToOpenAPI)
		httpGroup.POST("/from-curl", h.CreateFromCurl)
		httpGroup.POST("/from-openapi", h.CreateFromOpenAPI)
		httpGroup.POST("/from-openapi-file", h.CreateFromOpenAPIFile)
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
func parseCurlCommand(curlCommand string, name string, description string) (*models.HTTPInterface, error) {
	// Clean up the curl command
	curlCommand = strings.TrimSpace(curlCommand)
	curlCommand = strings.Replace(curlCommand, "\\\n", " ", -1) // Handle line continuations

	// Extract the HTTP method
	methodMatch := regexp.MustCompile(`-X\s+([A-Z]+)`).FindStringSubmatch(curlCommand)
	method := "GET" // Default to GET
	if len(methodMatch) > 1 {
		method = methodMatch[1]
	}

	// Extract the URL
	urlMatch := regexp.MustCompile(`curl\s+['"]?([^'"]*?)['"]?(\s+-|\s*$)`).FindStringSubmatch(curlCommand)
	if len(urlMatch) < 2 {
		return nil, errors.New("no URL found in curl command")
	}
	url := urlMatch[1]

	// Extract headers
	headerRegex := regexp.MustCompile(`-H\s+['"]([^'"]*)['"]`)
	headerMatches := headerRegex.FindAllStringSubmatch(curlCommand, -1)
	var headers []models.Header

	for _, match := range headerMatches {
		if len(match) < 2 {
			continue
		}

		headerLine := match[1]
		if headerLine == "" {
			continue
		}

		parts := strings.SplitN(headerLine, ":", 2)
		if len(parts) < 2 {
			continue
		}

		name := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Skip empty headers
		if name == "" {
			continue
		}

		// Set some common headers as required if they have values
		isRequired := false
		nameLower := strings.ToLower(name)
		if value != "" && (nameLower == "content-type" || nameLower == "accept" || nameLower == "authorization") {
			isRequired = true
		}

		// Special handling for authorization header
		if nameLower == "authorization" && value == "" {
			// Skip empty authorization headers completely
			continue
		}

		headers = append(headers, models.Header{
			Name:         name,
			Type:         "string", // Assuming string type for headers
			Required:     isRequired,
			Description:  fmt.Sprintf("The %s header", name),
			DefaultValue: value, // Adding default value from the curl command
		})
	}

	// Extract the request body
	var requestBody *models.Body
	// Use a regex pattern that works in Go's regexp package (no backreferences)
	// First try to match single-quoted data
	singleQuoteMatch := regexp.MustCompile(`--data(?:-raw)?\s+'([^']*)'`).FindStringSubmatch(curlCommand)
	// Then try to match double-quoted data
	doubleQuoteMatch := regexp.MustCompile(`--data(?:-raw)?\s+"([^"]*)"(?:\s+|$)`).FindStringSubmatch(curlCommand)

	// Use whichever match succeeded
	var data string
	if len(singleQuoteMatch) > 1 {
		data = singleQuoteMatch[1]
	} else if len(doubleQuoteMatch) > 1 {
		data = doubleQuoteMatch[1]
	}

	if data != "" {
		// If we find a request body with data, set method to POST if not explicitly specified
		if methodMatch == nil || len(methodMatch) <= 1 {
			method = "POST"
		}

		// Try to determine content type from headers
		var contentType string
		for _, header := range headers {
			if strings.ToLower(header.Name) == "content-type" {
				contentType = header.DefaultValue
				break
			}
		}

		// Default to application/json if content-type not specified
		if contentType == "" {
			contentType = "application/json"
		}

		// Log the extracted data for debugging
		fmt.Printf("Extracted request body data: %s\n", data)

		// If we have JSON content but the body isn't valid JSON, try to fix it
		isJSON := strings.Contains(strings.ToLower(contentType), "json")
		if isJSON && !json.Valid([]byte(data)) {
			// Try to see if it's a JSON string that needs to be unescaped
			unescapedData := strings.ReplaceAll(data, `\"`, `"`)

			// Check if the unescaped data is valid JSON
			if json.Valid([]byte(unescapedData)) {
				data = unescapedData
			}
		}

		requestBody = &models.Body{
			ContentType: contentType,
			Schema:      data,
			Example:     data,
		}
	}

	// Create the HTTP interface with the extracted information
	httpInterface := &models.HTTPInterface{
		Name:        name,
		Description: description,
		Method:      method,
		Path:        url,
		Headers:     headers,
		Parameters:  []models.Param{},
		RequestBody: requestBody,
		Responses:   []models.Response{},
	}

	return httpInterface, nil
}

// OpenAPIImport represents an OpenAPI spec to be converted to HTTP interfaces
type OpenAPIImport struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Spec        map[string]interface{} `json:"spec" binding:"required"`
}

// CreateFromOpenAPI creates new HTTP interfaces from an OpenAPI specification
func (h *HTTPInterfaceHandler) CreateFromOpenAPI(c *gin.Context) {
	var importReq OpenAPIImport
	if err := c.ShouldBindJSON(&importReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set default name if empty
	name := importReq.Name
	if name == "" {
		name = "api"
		// Try to get title from OpenAPI info
		if info, ok := importReq.Spec["info"].(map[string]interface{}); ok {
			if title, ok := info["title"].(string); ok && title != "" {
				name = title
			}
		}
	}

	// 尝试从OpenAPI info部分获取description，如果没有提供的话
	description := importReq.Description
	if description == "" {
		if info, ok := importReq.Spec["info"].(map[string]interface{}); ok {
			if desc, ok := info["description"].(string); ok {
				description = desc
			}
		}
	}

	// Convert OpenAPI to HTTP interfaces
	interfaces, err := models.CreateFromOpenAPI(name, description, importReq.Spec)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse OpenAPI spec: " + err.Error()})
		return
	}

	// Save each interface
	savedInterfaces := []models.HTTPInterface{}
	for _, httpInterface := range interfaces {
		if err := h.repo.Create(c.Request.Context(), &httpInterface); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save interfaces: " + err.Error()})
			return
		}
		savedInterfaces = append(savedInterfaces, httpInterface)
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":    fmt.Sprintf("Successfully created %d HTTP interfaces from OpenAPI spec", len(savedInterfaces)),
		"interfaces": savedInterfaces,
	})
}

// ExportToOpenAPI exports an HTTP interface to OpenAPI format
func (h *HTTPInterfaceHandler) ExportToOpenAPI(c *gin.Context) {
	id := c.Param("id")
	fmt.Printf("Exporting OpenAPI for interface with ID: %s\n", id)

	httpInterface, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		fmt.Printf("Error getting HTTP interface: %v\n", err)
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("HTTP interface not found: %s", err.Error())})
		return
	}

	fmt.Printf("HTTP interface found: %+v\n", httpInterface)
	fmt.Printf("Converting to OpenAPI...\n")

	openAPISpec := httpInterface.ConvertToOpenAPI()
	fmt.Printf("OpenAPI conversion result: %+v\n", openAPISpec)

	c.JSON(http.StatusOK, openAPISpec)
	fmt.Printf("Response sent to client\n")
}

// CreateFromOpenAPIFile handles OpenAPI file uploads and creates HTTP interfaces
func (h *HTTPInterfaceHandler) CreateFromOpenAPIFile(c *gin.Context) {
	// Get the uploaded file
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded: " + err.Error()})
		return
	}

	// Open the file
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open uploaded file: " + err.Error()})
		return
	}
	defer src.Close()

	// Read file content
	fileBytes, err := io.ReadAll(src)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file: " + err.Error()})
		return
	}

	// Check file extension to determine format
	var openAPISpec map[string]interface{}

	fileName := strings.ToLower(file.Filename)
	isYAML := strings.HasSuffix(fileName, ".yaml") || strings.HasSuffix(fileName, ".yml")

	if isYAML {
		// Parse YAML
		var yamlData interface{}
		if err := yaml.Unmarshal(fileBytes, &yamlData); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid YAML: " + err.Error()})
			return
		}

		// Convert YAML to JSON format
		jsonBytes, err := json.Marshal(yamlData)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to convert YAML to JSON: " + err.Error()})
			return
		}

		if err := json.Unmarshal(jsonBytes, &openAPISpec); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid OpenAPI format: " + err.Error()})
			return
		}
	} else {
		// Parse JSON directly
		if err := json.Unmarshal(fileBytes, &openAPISpec); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON: " + err.Error()})
			return
		}
	}

	// Get default name from info section or use "api" as fallback
	name := "api"
	description := ""

	if info, ok := openAPISpec["info"].(map[string]interface{}); ok {
		if title, ok := info["title"].(string); ok && title != "" {
			name = title
		}
		if desc, ok := info["description"].(string); ok {
			description = desc
		}
	}

	// Convert OpenAPI to HTTP interfaces
	interfaces, err := models.CreateFromOpenAPI(name, description, openAPISpec)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse OpenAPI spec: " + err.Error()})
		return
	}

	// Save each interface
	savedInterfaces := []models.HTTPInterface{}
	for _, httpInterface := range interfaces {
		if err := h.repo.Create(c.Request.Context(), &httpInterface); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save interfaces: " + err.Error()})
			return
		}
		savedInterfaces = append(savedInterfaces, httpInterface)
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":    fmt.Sprintf("Successfully created %d HTTP interfaces from OpenAPI file", len(savedInterfaces)),
		"interfaces": savedInterfaces,
	})
}
