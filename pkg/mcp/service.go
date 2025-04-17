package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/tidwall/gjson"
	"github.com/wangfeng/mcp-gateway2/pkg/models"
	"gopkg.in/yaml.v3"
)

var (
	ErrServerNotFound  = errors.New("MCP Server not found")
	ErrToolNotFound    = errors.New("tool not found")
	ErrInvalidResponse = errors.New("invalid response from MCP Server")
)

// MCPService provides functionality for managing MCP Servers
type MCPService struct {
	configDir  string
	servers    map[string]*models.MCPServer
	httpClient *http.Client
	mu         sync.RWMutex
}

// NewMCPService creates a new MCP Service
func NewMCPService(configDir string) (*MCPService, error) {
	// Create configuration directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, err
	}

	return &MCPService{
		configDir:  configDir,
		servers:    make(map[string]*models.MCPServer),
		httpClient: &http.Client{},
	}, nil
}

// GenerateYAML generates a YAML configuration for a MCP Server
func (s *MCPService) GenerateYAML(mcpServer *models.MCPServer) (string, error) {
	if mcpServer == nil {
		fmt.Printf("ERROR: Cannot generate YAML for nil MCP server\n")
		return "", fmt.Errorf("nil MCP server")
	}

	fmt.Printf("INFO: Generating YAML for MCP server: id=%s, name=%s\n", mcpServer.ID, mcpServer.Name)

	// Convert MCP Server model to a map
	yamlData := map[string]interface{}{
		"server": map[string]interface{}{
			"name":       mcpServer.Name,
			"allowTools": mcpServer.AllowTools,
		},
		"tools": []map[string]interface{}{},
	}

	// Convert tools
	for _, tool := range mcpServer.Tools {
		toolMap := map[string]interface{}{
			"name":        tool.Name,
			"description": tool.Description,
			"requestTemplate": map[string]interface{}{
				"method": tool.RequestTemplate.Method,
				"url":    tool.RequestTemplate.URL,
			},
			"responseTemplate": map[string]interface{}{
				"body": tool.ResponseTemplate.Body,
			},
		}

		// Add headers if present
		if len(tool.RequestTemplate.Headers) > 0 {
			toolMap["requestTemplate"].(map[string]interface{})["headers"] = tool.RequestTemplate.Headers
		}

		// Add body if present
		if tool.RequestTemplate.Body != "" {
			toolMap["requestTemplate"].(map[string]interface{})["body"] = tool.RequestTemplate.Body
		}

		yamlData["tools"] = append(yamlData["tools"].([]map[string]interface{}), toolMap)
	}

	// Marshal to YAML
	yamlBytes, err := yaml.Marshal(yamlData)
	if err != nil {
		fmt.Printf("ERROR: Failed to marshal YAML: %v\n", err)
		return "", err
	}

	fmt.Printf("INFO: Successfully generated YAML for MCP server: id=%s\n", mcpServer.ID)
	return string(yamlBytes), nil
}

// SaveYAML saves the YAML configuration for a MCP Server to disk
func (s *MCPService) SaveYAML(mcpServer *models.MCPServer) (string, error) {
	if mcpServer == nil {
		fmt.Printf("ERROR: Cannot save YAML for nil MCP server\n")
		return "", fmt.Errorf("nil MCP server")
	}

	fmt.Printf("INFO: Saving YAML for MCP server: id=%s\n", mcpServer.ID)

	yaml, err := s.GenerateYAML(mcpServer)
	if err != nil {
		fmt.Printf("ERROR: Failed to generate YAML: %v\n", err)
		return "", err
	}

	// Create directory if it doesn't exist
	configPath := filepath.Join(s.configDir, "config")
	if err := os.MkdirAll(configPath, 0755); err != nil {
		fmt.Printf("ERROR: Failed to create config directory: %v\n", err)
		return "", err
	}

	// Write YAML to file
	filePath := filepath.Join(configPath, fmt.Sprintf("%s.yaml", mcpServer.ID))
	if err := os.WriteFile(filePath, []byte(yaml), 0644); err != nil {
		fmt.Printf("ERROR: Failed to write YAML file: %v\n", err)
		return "", err
	}

	fmt.Printf("INFO: Saved YAML file to: %s\n", filePath)
	return filePath, nil
}

// RegisterServer registers an MCP Server with the service
func (s *MCPService) RegisterServer(mcpServer *models.MCPServer) error {
	if mcpServer == nil {
		fmt.Printf("ERROR: Cannot register nil MCP server\n")
		return fmt.Errorf("nil MCP server")
	}

	fmt.Printf("INFO: Registering MCP server: id=%s, name=%s\n", mcpServer.ID, mcpServer.Name)

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if the server has tools
	if len(mcpServer.Tools) == 0 {
		fmt.Printf("WARNING: MCP server has no tools: id=%s\n", mcpServer.ID)
	} else {
		fmt.Printf("INFO: MCP server has %d tools\n", len(mcpServer.Tools))
		for i, tool := range mcpServer.Tools {
			fmt.Printf("INFO: Tool %d: name=%s, method=%s, url=%s\n",
				i, tool.Name, tool.RequestTemplate.Method, tool.RequestTemplate.URL)
		}
	}

	// Cache the server
	s.servers[mcpServer.ID] = mcpServer
	fmt.Printf("INFO: Successfully registered MCP server in cache: id=%s\n", mcpServer.ID)

	return nil
}

// HandleToolRequest handles a tool request for an MCP Server
func (s *MCPService) HandleToolRequest(ctx context.Context, serverID, toolName string, params map[string]interface{}) (string, error) {
	// Get the server definition
	s.mu.RLock()
	server, ok := s.servers[serverID]
	s.mu.RUnlock()

	if !ok {
		fmt.Printf("ERROR: Server not found: %s\n", serverID)
		return "", ErrServerNotFound
	}

	// Find the tool definition
	var toolDef *models.Tool
	for _, tool := range server.Tools {
		if tool.Name == toolName {
			toolDef = &tool
			break
		}
	}

	if toolDef == nil {
		fmt.Printf("ERROR: Tool not found: %s for server: %s\n", toolName, serverID)
		return "", ErrToolNotFound
	}

	fmt.Printf("INFO: Executing tool request: %s for server: %s with params: %+v\n", toolName, serverID, params)

	// Execute the tool request using the tool definition
	resp, err := s.executeToolRequest(ctx, server, toolDef, params)
	if err != nil {
		fmt.Printf("ERROR: Failed to execute tool request: %s - %v\n", toolName, err)
		return "", err
	}

	fmt.Printf("INFO: Tool request completed successfully: %s\n", toolName)
	return resp, nil
}

// executeToolRequest executes a tool request using the tool definition
func (s *MCPService) executeToolRequest(ctx context.Context, server *models.MCPServer, tool *models.Tool, params map[string]interface{}) (string, error) {
	// Create request based on the tool's request template
	req, err := s.createRequest(ctx, tool, params)
	if err != nil {
		fmt.Printf("ERROR: Failed to create request for tool %s: %v\n", tool.Name, err)
		return "", err
	}

	fmt.Printf("INFO: Sending request to: %s %s\n", req.Method, req.URL.String())

	// Execute request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		fmt.Printf("ERROR: HTTP request failed for tool %s: %v\n", tool.Name, err)
		return "", err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("ERROR: Failed to read response body for tool %s: %v\n", tool.Name, err)
		return "", err
	}

	// 打印详细的响应信息
	fmt.Printf("INFO: ======== RESPONSE DETAILS ========\n")
	fmt.Printf("INFO: Status: %d %s\n", resp.StatusCode, resp.Status)
	fmt.Printf("INFO: Headers:\n")
	for key, values := range resp.Header {
		for _, value := range values {
			fmt.Printf("INFO:   %s: %s\n", key, value)
		}
	}
	fmt.Printf("INFO: Body: %s\n", string(body))
	fmt.Printf("INFO: ================================\n")

	// If the status code is not successful, return an error
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errMessage := fmt.Sprintf("request failed with status code %d: %s", resp.StatusCode, string(body))
		fmt.Printf("ERROR: %s\n", errMessage)
		return "", fmt.Errorf(errMessage)
	}

	// Process response according to the tool's response template
	result, err := s.processResponse(tool, body)
	if err != nil {
		fmt.Printf("ERROR: Failed to process response for tool %s: %v\n", tool.Name, err)
		return "", err
	}

	// 打印处理后的结果
	fmt.Printf("INFO: Processed response result: %s\n", result)
	return result, nil
}

// createRequest creates an HTTP request based on the tool definition and parameters
func (s *MCPService) createRequest(ctx context.Context, tool *models.Tool, params map[string]interface{}) (*http.Request, error) {
	// Get URL and method from the tool definition
	url := tool.RequestTemplate.URL
	method := tool.RequestTemplate.Method

	fmt.Printf("DEBUG: Creating request with URL template: %s\n", url)

	// Replace URL parameters with values from params
	// Example: If URL is "https://api.example.com/{param1}/{param2}"
	// and params has {"param1": "value1", "param2": "value2"},
	// the result should be "https://api.example.com/value1/value2"
	for key, value := range params {
		placeholder := fmt.Sprintf("{%s}", key)
		if !strings.Contains(url, placeholder) {
			fmt.Printf("DEBUG: Parameter '%s' not found in URL template\n", key)
			continue
		}

		strValue := fmt.Sprintf("%v", value)
		url = strings.ReplaceAll(url, placeholder, strValue)
		fmt.Printf("DEBUG: Replaced '%s' with '%s' in URL\n", placeholder, strValue)
	}

	fmt.Printf("DEBUG: Final URL after parameter replacement: %s\n", url)

	// Extract user-provided headers, body, and other parameters from params
	userHeaders := map[string]string{}
	userBody := map[string]interface{}{}

	// Check if headers are provided in the params
	if headersParam, ok := params["headers"]; ok {
		if headersMap, ok := headersParam.(map[string]interface{}); ok {
			for k, v := range headersMap {
				userHeaders[k] = fmt.Sprintf("%v", v)
			}
			// Remove headers from params to avoid confusion with URL or query params
			delete(params, "headers")
		}
	}

	// Check if body is provided in the params
	if bodyParam, ok := params["body"]; ok {
		if bodyMap, ok := bodyParam.(map[string]interface{}); ok {
			userBody = bodyMap
			// Remove body from params to avoid confusion with URL or query params
			delete(params, "body")
		} else if bodyStr, ok := bodyParam.(string); ok && bodyStr != "" {
			// Try to parse as JSON if it's a string
			if err := json.Unmarshal([]byte(bodyStr), &userBody); err != nil {
				// If not valid JSON, treat it as a raw string body
				userBody = map[string]interface{}{"raw": bodyStr}
			}
			// Remove body from params
			delete(params, "body")
		}
	}

	// Create request body if method is not GET
	var reqBody io.Reader
	var bodyJson string
	if method != "GET" {
		if len(userBody) > 0 {
			// User provided a body
			jsonData, err := json.Marshal(userBody)
			if err != nil {
				fmt.Printf("ERROR: Failed to marshal user body: %v\n", err)
				return nil, err
			}
			bodyJson = string(jsonData)
			fmt.Printf("DEBUG: Using user-provided body: %s\n", bodyJson)
			reqBody = bytes.NewBuffer(jsonData)
		} else if tool.RequestTemplate.Body != "" {
			// Use template body with parameter replacement
			bodyTemplate := tool.RequestTemplate.Body
			var err error
			bodyJson, err = replaceParams(bodyTemplate, params)
			if err != nil {
				fmt.Printf("ERROR: Failed to replace parameters in request body: %v\n", err)
				return nil, err
			}
			fmt.Printf("DEBUG: Request body after parameter replacement: %s\n", bodyJson)
			reqBody = bytes.NewBuffer([]byte(bodyJson))
		}
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		fmt.Printf("ERROR: Failed to create HTTP request: %v\n", err)
		return nil, err
	}

	// Add default headers from tool definition first
	for key, value := range tool.RequestTemplate.Headers {
		req.Header.Set(key, value)
		fmt.Printf("DEBUG: Added default header: %s: %s\n", key, value)
	}

	// Override with user-provided headers
	for key, value := range userHeaders {
		req.Header.Set(key, value)
		fmt.Printf("DEBUG: Overrode with user header: %s: %s\n", key, value)
	}

	// Set default Content-Type if not provided and body exists
	if reqBody != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
		fmt.Printf("DEBUG: Added default Content-Type: application/json\n")
	}

	// Handle query parameters for GET requests (or other methods if URL contains query params)
	if len(params) > 0 {
		q := req.URL.Query()
		for key, value := range params {
			// Skip parameters that were used in the URL template
			placeholder := fmt.Sprintf("{%s}", key)
			if strings.Contains(tool.RequestTemplate.URL, placeholder) {
				continue
			}

			q.Add(key, fmt.Sprintf("%v", value))
			fmt.Printf("DEBUG: Added query parameter: %s=%v\n", key, value)
		}
		req.URL.RawQuery = q.Encode()
		fmt.Printf("DEBUG: Final query string: %s\n", req.URL.RawQuery)
	}

	// 打印完整的请求信息
	fmt.Printf("INFO: ======== REQUEST DETAILS ========\n")
	fmt.Printf("INFO: Method: %s\n", req.Method)
	fmt.Printf("INFO: URL: %s\n", req.URL.String())
	fmt.Printf("INFO: Headers:\n")
	for key, values := range req.Header {
		for _, value := range values {
			fmt.Printf("INFO:   %s: %s\n", key, value)
		}
	}
	if reqBody != nil {
		fmt.Printf("INFO: Body: %s\n", bodyJson)
	} else {
		fmt.Printf("INFO: Body: <none>\n")
	}
	fmt.Printf("INFO: ================================\n")

	return req, nil
}

// processResponse processes the response according to the tool's response template
func (s *MCPService) processResponse(tool *models.Tool, responseBody []byte) (string, error) {
	// If there's no response template, return the raw response
	if tool.ResponseTemplate.Body == "" {
		return string(responseBody), nil
	}

	// Parse the response JSON
	result := gjson.ParseBytes(responseBody)

	// Replace template variables with values from the response
	responseTemplate := tool.ResponseTemplate.Body
	formattedResponse, err := replaceResponseVars(responseTemplate, result)
	if err != nil {
		return "", err
	}

	return formattedResponse, nil
}

// replaceParams replaces parameter placeholders in a template string with actual values
func replaceParams(template string, params map[string]interface{}) (string, error) {
	// Check if the template is a valid JSON
	var jsonObj interface{}
	if json.Valid([]byte(template)) {
		fmt.Printf("DEBUG: Template is valid JSON\n")
		// Try to replace parameters in the JSON template
		// First decode the JSON template
		if err := json.Unmarshal([]byte(template), &jsonObj); err != nil {
			fmt.Printf("ERROR: Failed to unmarshal JSON template: %v\n", err)
			return "", err
		}

		// Replace parameters in the JSON structure
		jsonObj = replaceJSONParams(jsonObj, params)

		// Marshal back to JSON
		result, err := json.Marshal(jsonObj)
		if err != nil {
			fmt.Printf("ERROR: Failed to marshal JSON after parameter replacement: %v\n", err)
			return "", err
		}

		return string(result), nil
	}

	// If not valid JSON, treat as string template
	fmt.Printf("DEBUG: Template is not a valid JSON, treating as string template\n")
	result := template

	// Replace parameters in the string template
	for key, value := range params {
		placeholder := fmt.Sprintf("{%s}", key)
		strValue := fmt.Sprintf("%v", value)
		result = strings.ReplaceAll(result, placeholder, strValue)
		fmt.Printf("DEBUG: Replaced '%s' with '%s' in string template\n", placeholder, strValue)
	}

	return result, nil
}

// replaceJSONParams recursively replaces parameters in a JSON structure
func replaceJSONParams(value interface{}, params map[string]interface{}) interface{} {
	// If the value is a string, check if it contains parameter placeholders
	if strValue, ok := value.(string); ok {
		for key, paramValue := range params {
			placeholder := fmt.Sprintf("{%s}", key)
			if strings.Contains(strValue, placeholder) {
				strValue = strings.ReplaceAll(strValue, placeholder, fmt.Sprintf("%v", paramValue))
				fmt.Printf("DEBUG: Replaced '%s' with '%v' in JSON string\n", placeholder, paramValue)
			}
		}
		return strValue
	}

	// If the value is a map, process each key-value pair
	if mapValue, ok := value.(map[string]interface{}); ok {
		for k, v := range mapValue {
			mapValue[k] = replaceJSONParams(v, params)
		}
		return mapValue
	}

	// If the value is a slice, process each element
	if sliceValue, ok := value.([]interface{}); ok {
		for i, v := range sliceValue {
			sliceValue[i] = replaceJSONParams(v, params)
		}
		return sliceValue
	}

	// Otherwise, return the value as is
	return value
}

// replaceResponseVars replaces template variables with values from the response
func replaceResponseVars(template string, result gjson.Result) (string, error) {
	// In a real implementation, you'd parse the template and replace variables
	// For now, let's simulate a simple format based on the Random User API example

	// Format the response according to a template similar to the example
	if result.Get("results").Exists() {
		name := result.Get("results.0.name.first").String() + " " + result.Get("results.0.name.last").String()
		email := result.Get("results.0.email").String()
		location := result.Get("results.0.location.city").String() + ", " + result.Get("results.0.location.country").String()
		phone := result.Get("results.0.phone").String()

		response := fmt.Sprintf("# User Information\n- **Name**: %s\n- **Email**: %s\n- **Location**: %s\n- **Phone**: %s",
			name, email, location, phone)

		return response, nil
	}

	// If the response doesn't match our expected format, return it as-is
	return result.Raw, nil
}

// GetConfigDir returns the directory where configuration files are stored
func (s *MCPService) GetConfigDir() string {
	return s.configDir
}
