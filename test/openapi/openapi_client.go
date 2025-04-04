package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

const (
	apiBaseURL = "http://localhost:8080"
)

func main() {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Test 1: Export an existing HTTP interface to OpenAPI
	fmt.Println("\n=== Test 1: Export HTTP Interface to OpenAPI ===")
	// Use a fixed ID for testing
	interfaceID := "http-20250404-2"
	fmt.Printf("Using interface ID: %s\n", interfaceID)

	// Export to OpenAPI
	openAPI, err := exportToOpenAPI(client, interfaceID)
	if err != nil {
		log.Fatalf("Failed to export interface to OpenAPI: %v", err)
	}

	fmt.Println("Successfully exported to OpenAPI:")
	fmt.Printf("  OpenAPI Version: %s\n", openAPI["openapi"])

	// Display info
	if info, ok := openAPI["info"].(map[string]interface{}); ok {
		fmt.Printf("  Title: %s\n", info["title"])
		fmt.Printf("  Description: %s\n", info["description"])
		fmt.Printf("  Version: %s\n", info["version"])
	}

	// Display paths
	if paths, ok := openAPI["paths"].(map[string]interface{}); ok {
		fmt.Printf("  Paths: %d\n", len(paths))
		for path, _ := range paths {
			fmt.Printf("    - %s\n", path)
		}
	}

	// Save the OpenAPI spec to a file for the next test
	openAPIJSON, err := json.MarshalIndent(openAPI, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal OpenAPI spec: %v", err)
	}

	if err := os.WriteFile("openapi-export.json", openAPIJSON, 0644); err != nil {
		log.Printf("Warning: Failed to save OpenAPI spec to file: %v", err)
	} else {
		fmt.Println("OpenAPI spec saved to openapi-export.json")
	}

	// Test 2: Import OpenAPI spec to HTTP interfaces
	fmt.Println("\n=== Test 2: Import OpenAPI to HTTP Interfaces ===")

	// Create a sample OpenAPI spec
	sampleOpenAPI := map[string]interface{}{
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title":       "Sample API",
			"description": "A sample API for testing OpenAPI import",
			"version":     "1.0.0",
		},
		"paths": map[string]interface{}{
			"/users": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Get all users",
					"description": "Returns a list of users",
					"operationId": "getUsers",
					"parameters": []map[string]interface{}{
						{
							"name":        "limit",
							"in":          "query",
							"description": "Maximum number of users to return",
							"required":    false,
							"schema": map[string]interface{}{
								"type": "integer",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "A list of users",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "array",
										"items": map[string]interface{}{
											"type": "object",
											"properties": map[string]interface{}{
												"id":   map[string]interface{}{"type": "string"},
												"name": map[string]interface{}{"type": "string"},
											},
										},
									},
									"example": []map[string]interface{}{
										{"id": "1", "name": "John Doe"},
										{"id": "2", "name": "Jane Smith"},
									},
								},
							},
						},
					},
				},
				"post": map[string]interface{}{
					"summary":     "Create a user",
					"description": "Creates a new user",
					"operationId": "createUser",
					"requestBody": map[string]interface{}{
						"description": "User object",
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"name":  map[string]interface{}{"type": "string"},
										"email": map[string]interface{}{"type": "string"},
									},
									"required": []string{"name", "email"},
								},
								"example": map[string]string{
									"name":  "New User",
									"email": "user@example.com",
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"201": map[string]interface{}{
							"description": "User created",
						},
						"400": map[string]interface{}{
							"description": "Invalid input",
						},
					},
				},
			},
			"/users/{id}": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Get user by ID",
					"description": "Returns a single user",
					"operationId": "getUserById",
					"parameters": []map[string]interface{}{
						{
							"name":        "id",
							"in":          "path",
							"description": "User ID",
							"required":    true,
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "A user object",
						},
						"404": map[string]interface{}{
							"description": "User not found",
						},
					},
				},
			},
		},
	}

	// Import OpenAPI spec
	importResult, err := importFromOpenAPI(client, "sample-api", "Sample API for testing", sampleOpenAPI)
	if err != nil {
		log.Fatalf("Failed to import OpenAPI spec: %v", err)
	}

	fmt.Printf("Successfully imported OpenAPI spec: %s\n", importResult["message"])

	// Display imported interfaces
	if interfaces, ok := importResult["interfaces"].([]interface{}); ok {
		fmt.Printf("Created interfaces: %d\n", len(interfaces))
		for i, intf := range interfaces {
			interfaceMap := intf.(map[string]interface{})
			fmt.Printf("  %d. %s - %s %s\n", i+1,
				interfaceMap["name"],
				interfaceMap["method"],
				interfaceMap["path"])
		}
	}

	// Test 3: Use the previously exported OpenAPI spec (if available) to create interfaces
	openAPIFile, err := os.ReadFile("openapi-export.json")
	if err == nil {
		fmt.Println("\n=== Test 3: Import Previously Exported OpenAPI ===")

		var exportedSpec map[string]interface{}
		if err := json.Unmarshal(openAPIFile, &exportedSpec); err != nil {
			log.Printf("Warning: Failed to parse exported OpenAPI spec: %v", err)
		} else {
			importResult, err := importFromOpenAPI(client, "imported-api", "Imported from exported OpenAPI", exportedSpec)
			if err != nil {
				log.Printf("Warning: Failed to import exported OpenAPI spec: %v", err)
			} else {
				fmt.Printf("Successfully imported previously exported OpenAPI spec: %s\n", importResult["message"])
			}
		}
	}

	fmt.Println("\nTests completed successfully!")
}

// Helper function to get all HTTP interfaces
func getHTTPInterfaces(client *http.Client) ([]map[string]interface{}, error) {
	resp, err := client.Get(apiBaseURL + "/api/http-interfaces")
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

// Helper function to export an HTTP interface to OpenAPI
func exportToOpenAPI(client *http.Client, id string) (map[string]interface{}, error) {
	url := apiBaseURL + "/api/http-interfaces/" + id + "/openapi"
	fmt.Printf("Making request to: %s\n", url)

	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	fmt.Printf("Response status: %d\n", resp.StatusCode)
	fmt.Printf("Response body: %s\n", string(body))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, body)
	}

	var openAPI map[string]interface{}
	if err := json.Unmarshal(body, &openAPI); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}

	return openAPI, nil
}

// Helper function to import OpenAPI spec to HTTP interfaces
func importFromOpenAPI(client *http.Client, name, description string, spec map[string]interface{}) (map[string]interface{}, error) {
	reqBody := map[string]interface{}{
		"name":        name,
		"description": description,
		"spec":        spec,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	url := apiBaseURL + "/api/http-interfaces/from-openapi"
	fmt.Printf("Making POST request to: %s\n", url)
	fmt.Printf("Request body: %s\n", string(jsonBody))

	resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	fmt.Printf("Response status: %d\n", resp.StatusCode)
	fmt.Printf("Response body: %s\n", string(body))

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, body)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}

	return result, nil
}
