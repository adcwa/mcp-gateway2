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
	baseURL = "http://localhost:8080"
)

func main() {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Step 1: List HTTP interfaces
	fmt.Println("Step 1: Listing HTTP interfaces...")
	httpInterfaces, err := getHTTPInterfaces(client)
	if err != nil {
		log.Fatalf("Failed to get HTTP interfaces: %v", err)
	}

	fmt.Printf("Found %d HTTP interfaces\n", len(httpInterfaces))
	for i, httpInterface := range httpInterfaces {
		fmt.Printf("%d. %s (%s)\n", i+1, httpInterface["name"], httpInterface["id"])
	}

	if len(httpInterfaces) == 0 {
		log.Fatalf("No HTTP interfaces found")
	}

	// Step 2: Create MCP server
	fmt.Println("\nStep 2: Creating MCP server...")

	// Get first HTTP interface ID
	httpID := httpInterfaces[0]["id"].(string)

	// Create request body
	createServerReq := map[string]interface{}{
		"name":        "random-user-server",
		"description": "MCP Server for random user API",
		"httpIds":     []string{httpID},
	}

	mcpServer, err := createMCPServer(client, createServerReq)
	if err != nil {
		log.Fatalf("Failed to create MCP server: %v", err)
	}

	fmt.Printf("MCP server created with ID: %s\n", mcpServer["id"])

	// Step 3: Compile MCP server
	fmt.Println("\nStep 3: Compiling MCP server...")
	compileResp, err := compileMCPServer(client, mcpServer["id"].(string))
	if err != nil {
		log.Fatalf("Failed to compile MCP server: %v", err)
	}

	fmt.Printf("MCP server compiled successfully: %s\n", compileResp["message"])
	fmt.Printf("WASM path: %s\n", compileResp["wasmPath"])

	// Step 4: Activate MCP server
	fmt.Println("\nStep 4: Activating MCP server...")
	activateResp, err := activateMCPServer(client, mcpServer["id"].(string))
	if err != nil {
		log.Fatalf("Failed to activate MCP server: %v", err)
	}

	fmt.Printf("MCP server activated successfully: %s\n", activateResp["message"])

	// Step 5: Invoke tool
	fmt.Println("\nStep 5: Invoking tool...")
	toolName := mcpServer["tools"].([]interface{})[0].(map[string]interface{})["name"].(string)

	toolResp, err := invokeTool(client, mcpServer["id"].(string), toolName, map[string]interface{}{})
	if err != nil {
		log.Fatalf("Failed to invoke tool: %v", err)
	}

	fmt.Printf("\nTool response:\n%s\n", toolResp["result"])

	fmt.Println("\nTest completed successfully!")
}

// Helper functions
func getHTTPInterfaces(client *http.Client) ([]map[string]interface{}, error) {
	resp, err := client.Get(baseURL + "/api/http-interfaces")
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

func createMCPServer(client *http.Client, reqBody map[string]interface{}) (map[string]interface{}, error) {
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	resp, err := client.Post(baseURL+"/api/mcp-servers", "application/json", bytes.NewBuffer(jsonBody))
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

func compileMCPServer(client *http.Client, serverID string) (map[string]interface{}, error) {
	resp, err := client.Post(baseURL+"/api/mcp-servers/"+serverID+"/compile", "application/json", nil)
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

func activateMCPServer(client *http.Client, serverID string) (map[string]interface{}, error) {
	resp, err := client.Post(baseURL+"/api/mcp-servers/"+serverID+"/activate", "application/json", nil)
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

func invokeTool(client *http.Client, serverID string, toolName string, params map[string]interface{}) (map[string]interface{}, error) {
	jsonBody, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	resp, err := client.Post(baseURL+"/api/mcp-servers/"+serverID+"/tools/"+toolName, "application/json", bytes.NewBuffer(jsonBody))
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
