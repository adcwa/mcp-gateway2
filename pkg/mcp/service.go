package mcp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/tetratelabs/wazero"
	"github.com/tidwall/gjson"
	"github.com/wangfeng/mcp-gateway2/pkg/models"
	"gopkg.in/yaml.v3"
)

var (
	ErrServerNotFound    = errors.New("MCP Server not found")
	ErrToolNotFound      = errors.New("tool not found")
	ErrServerNotCompiled = errors.New("MCP Server not compiled to WASM")
	ErrInvalidResponse   = errors.New("invalid response from MCP Server")
)

// MCPService provides functionality for managing MCP Servers
type MCPService struct {
	wasmDir    string
	wasmCache  map[string]wazero.CompiledModule
	wasmRT     wazero.Runtime
	httpClient *http.Client
	mu         sync.RWMutex
}

// NewMCPService creates a new MCP Service
func NewMCPService(wasmDir string) (*MCPService, error) {
	// Create WASM directory if it doesn't exist
	if err := os.MkdirAll(wasmDir, 0755); err != nil {
		return nil, err
	}

	// Initialize WASM runtime
	r := wazero.NewRuntime(context.Background())

	return &MCPService{
		wasmDir:    wasmDir,
		wasmCache:  make(map[string]wazero.CompiledModule),
		wasmRT:     r,
		httpClient: &http.Client{},
	}, nil
}

// GenerateYAML generates a YAML configuration for a MCP Server
func (s *MCPService) GenerateYAML(mcpServer *models.MCPServer) (string, error) {
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
		return "", err
	}

	return string(yamlBytes), nil
}

// SaveYAML saves the YAML configuration for a MCP Server to disk
func (s *MCPService) SaveYAML(mcpServer *models.MCPServer) (string, error) {
	yaml, err := s.GenerateYAML(mcpServer)
	if err != nil {
		return "", err
	}

	// Create directory if it doesn't exist
	configPath := filepath.Join(s.wasmDir, "config")
	if err := os.MkdirAll(configPath, 0755); err != nil {
		return "", err
	}

	// Write YAML to file
	filePath := filepath.Join(configPath, fmt.Sprintf("%s.yaml", mcpServer.ID))
	if err := os.WriteFile(filePath, []byte(yaml), 0644); err != nil {
		return "", err
	}

	return filePath, nil
}

// CompileToWasm compiles an MCP Server configuration to a WebAssembly module
// In a real implementation, this would compile the configuration to WASM.
// For this MVP, we'll simulate this by using a pre-compiled WASM file.
func (s *MCPService) CompileToWasm(mcpServer *models.MCPServer) (string, error) {
	// Save the YAML configuration
	_, err := s.SaveYAML(mcpServer)
	if err != nil {
		return "", err
	}

	// In a real implementation, this would compile the configuration to WASM.
	// For this MVP, we'll use a mock WASM file
	wasmPath := filepath.Join(s.wasmDir, fmt.Sprintf("%s.wasm", mcpServer.ID))

	// Create a dummy WASM file (in a real implementation, this would be generated)
	// In this MVP, we'll just copy a pre-compiled WASM file if available, or create an empty one
	// For real functionality, you'd need a compiler that converts the YAML to WASM
	dummyWasm := []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00} // Magic number + version 1
	if err := os.WriteFile(wasmPath, dummyWasm, 0644); err != nil {
		return "", err
	}

	return wasmPath, nil
}

// LoadWasmModule loads a WebAssembly module for an MCP Server
func (s *MCPService) LoadWasmModule(ctx context.Context, mcpServer *models.MCPServer) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if the WASM path exists
	if mcpServer.WasmPath == "" {
		return ErrServerNotCompiled
	}

	// Read the WASM file
	wasmBytes, err := os.ReadFile(mcpServer.WasmPath)
	if err != nil {
		return err
	}

	// Compile the WASM module
	module, err := s.wasmRT.CompileModule(ctx, wasmBytes)
	if err != nil {
		return err
	}

	// Cache the compiled module
	s.wasmCache[mcpServer.ID] = module

	return nil
}

// HandleToolRequest handles a tool request for an MCP Server
func (s *MCPService) HandleToolRequest(ctx context.Context, serverID, toolName string, params map[string]interface{}) (string, error) {
	// In a real implementation, this would use the WASM module to handle the request.
	// For this MVP, we'll simulate this by making an HTTP request directly.

	// Check if we have the WASM module for this server
	s.mu.RLock()
	_, ok := s.wasmCache[serverID]
	s.mu.RUnlock()

	if !ok {
		return "", ErrServerNotFound
	}

	// Simulate executing the WASM module by making an HTTP request
	// In a real implementation, this would execute the WASM module
	// and use the result to make a request.

	// This is just a placeholder for the MVP to demonstrate the concept
	resp, err := s.executeToolRequest(ctx, serverID, toolName, params)
	if err != nil {
		return "", err
	}

	return resp, nil
}

// executeToolRequest executes a tool request for an MCP Server
// This is a simplified simulation for the MVP
func (s *MCPService) executeToolRequest(ctx context.Context, serverID, toolName string, params map[string]interface{}) (string, error) {
	// In a real implementation, this would use the WASM module to execute the request.
	// For this MVP, we'll simulate this by making a GET request to a test API.

	// For demo purposes, let's use the Random User API
	resp, err := s.httpClient.Get("https://randomuser.me/api/")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Parse the JSON response
	result := gjson.ParseBytes(body)

	// Apply a simple template for the response
	name := result.Get("results.0.name.first").String() + " " + result.Get("results.0.name.last").String()
	email := result.Get("results.0.email").String()
	location := result.Get("results.0.location.city").String() + ", " + result.Get("results.0.location.country").String()
	phone := result.Get("results.0.phone").String()

	// Format the response according to a template similar to the example
	response := fmt.Sprintf("# User Information\n- **Name**: %s\n- **Email**: %s\n- **Location**: %s\n- **Phone**: %s",
		name, email, location, phone)

	return response, nil
}

// Close closes the MCP Service and cleans up resources
func (s *MCPService) Close(ctx context.Context) error {
	return s.wasmRT.Close(ctx)
}
