package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const (
	curlBaseURL = "http://localhost:8080"
)

func main() {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Test cases with different curl commands
	testCases := []struct {
		name        string
		description string
		curlCmd     string
	}{
		{
			name:        "github-api",
			description: "GitHub API to fetch user information",
			curlCmd:     `curl -H "Accept: application/vnd.github.v3+json" https://api.github.com/users/octocat`,
		},
		{
			name:        "post-example",
			description: "Example POST request with JSON data",
			curlCmd:     `curl -X POST -H "Content-Type: application/json" -d '{"name":"John","age":30}' https://example.com/api/users`,
		},
		{
			name:        "weather-api",
			description: "Weather API request with query parameters",
			curlCmd:     `curl "https://api.openweathermap.org/data/2.5/weather?q=London&appid=YOUR_API_KEY"`,
		},
	}

	// Process each test case
	for i, tc := range testCases {
		fmt.Printf("\n--- Test Case %d: %s ---\n", i+1, tc.name)

		// Convert curl command to HTTP interface
		httpInterface, err := convertCurlToHTTPInterface(client, tc.curlCmd, tc.name, tc.description)
		if err != nil {
			log.Fatalf("Failed to convert curl command: %v", err)
		}

		fmt.Printf("Successfully created HTTP interface with ID: %s\n", httpInterface["id"])
		fmt.Printf("  Name: %s\n", httpInterface["name"])
		fmt.Printf("  Method: %s\n", httpInterface["method"])
		fmt.Printf("  Path: %s\n", httpInterface["path"])

		// Print headers if present
		if headers, ok := httpInterface["headers"].([]interface{}); ok && len(headers) > 0 {
			fmt.Println("  Headers:")
			for _, header := range headers {
				headerMap := header.(map[string]interface{})
				fmt.Printf("    - %s\n", headerMap["name"])
			}
		}

		// Print request body if present
		if requestBody, ok := httpInterface["requestBody"].(map[string]interface{}); ok {
			fmt.Println("  Request Body:")
			fmt.Printf("    ContentType: %s\n", requestBody["contentType"])
			if example, ok := requestBody["example"].(string); ok && example != "" {
				fmt.Printf("    Example: %s\n", example)
			}
		}

		fmt.Println("\nHTTP interface created and saved successfully!")
	}

	// Create MCP server from the first HTTP interface
	fmt.Println("\n\n--- Creating MCP Server from HTTP Interfaces ---")

	// List all HTTP interfaces
	interfaces, err := getHTTPInterfacesList(client)
	if err != nil {
		log.Fatalf("Failed to get HTTP interfaces: %v", err)
	}

	if len(interfaces) == 0 {
		log.Fatalf("No HTTP interfaces found")
	}

	// Get IDs of the interfaces created from curl commands
	httpIDs := []string{}
	for _, httpInterface := range interfaces {
		for _, tc := range testCases {
			if httpInterface["name"] == tc.name {
				httpIDs = append(httpIDs, httpInterface["id"].(string))
				break
			}
		}
	}

	if len(httpIDs) == 0 {
		log.Fatalf("None of the curl-created interfaces found")
	}

	// Create MCP server from the interfaces
	createServerReq := map[string]interface{}{
		"name":        "curl-examples-server",
		"description": "MCP Server for curl example APIs",
		"httpIds":     httpIDs,
	}

	mcpServer, err := createServerFromInterfaces(client, createServerReq)
	if err != nil {
		log.Fatalf("Failed to create MCP server: %v", err)
	}

	fmt.Printf("MCP server created with ID: %s\n", mcpServer["id"])
	fmt.Printf("Tools in MCP server: %d\n", len(mcpServer["tools"].([]interface{})))

	// Compile and activate the MCP server
	fmt.Println("\n--- Compiling and Activating MCP Server ---")

	// Compile
	compileResp, err := compileServer(client, mcpServer["id"].(string))
	if err != nil {
		log.Fatalf("Failed to compile MCP server: %v", err)
	}
	fmt.Printf("MCP server compiled successfully: %s\n", compileResp["message"])

	// Activate
	activateResp, err := activateServer(client, mcpServer["id"].(string))
	if err != nil {
		log.Fatalf("Failed to activate MCP server: %v", err)
	}
	fmt.Printf("MCP server activated successfully: %s\n", activateResp["message"])

	// Test invoking a tool from the MCP server
	fmt.Println("\n--- Invoking Tool from MCP Server ---")

	// Use the first tool
	tools := mcpServer["tools"].([]interface{})
	if len(tools) == 0 {
		log.Fatalf("No tools found in MCP server")
	}

	toolName := tools[0].(map[string]interface{})["name"].(string)

	// Invoke tool
	toolResp, err := invokeServerTool(client, mcpServer["id"].(string), toolName, map[string]interface{}{})
	if err != nil {
		log.Fatalf("Failed to invoke tool: %v", err)
	}

	fmt.Printf("\nTool response:\n%s\n", toolResp["result"])

	fmt.Println("\nTest completed successfully!")
}

// Helper function to convert curl to HTTP interface
func convertCurlToHTTPInterface(client *http.Client, curlCmd, name, description string) (map[string]interface{}, error) {
	// Create request body
	requestBody := map[string]interface{}{
		"command":     curlCmd,
		"name":        name,
		"description": description,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	resp, err := client.Post(curlBaseURL+"/api/http-interfaces/from-curl", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, body)
	}

	var httpInterface map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&httpInterface); err != nil {
		return nil, err
	}

	return httpInterface, nil
}

// Helper functions below are renamed to avoid conflicts with client.go

func getHTTPInterfacesList(client *http.Client) ([]map[string]interface{}, error) {
	resp, err := client.Get(curlBaseURL + "/api/http-interfaces")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var interfaces []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&interfaces); err != nil {
		return nil, err
	}

	return interfaces, nil
}

func createServerFromInterfaces(client *http.Client, reqBody map[string]interface{}) (map[string]interface{}, error) {
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	resp, err := client.Post(curlBaseURL+"/api/mcp-servers", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, body)
	}

	var server map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&server); err != nil {
		return nil, err
	}

	return server, nil
}

func compileServer(client *http.Client, serverID string) (map[string]interface{}, error) {
	resp, err := client.Post(curlBaseURL+"/api/mcp-servers/"+serverID+"/compile", "application/json", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, body)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

func activateServer(client *http.Client, serverID string) (map[string]interface{}, error) {
	resp, err := client.Post(curlBaseURL+"/api/mcp-servers/"+serverID+"/activate", "application/json", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, body)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

func invokeServerTool(client *http.Client, serverID string, toolName string, params map[string]interface{}) (map[string]interface{}, error) {
	jsonBody, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	resp, err := client.Post(curlBaseURL+"/api/mcp-servers/"+serverID+"/tools/"+toolName, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, body)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}
