package models

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// HTTPInterface represents an API configuration
type HTTPInterface struct {
	ID          string     `json:"id"`
	Name        string     `json:"name" binding:"required"`
	Description string     `json:"description"`
	Method      string     `json:"method" binding:"required,oneof=GET POST PUT DELETE PATCH"`
	Path        string     `json:"path" binding:"required"`
	Headers     []Header   `json:"headers"`
	Parameters  []Param    `json:"parameters"`
	RequestBody *Body      `json:"requestBody,omitempty"`
	Responses   []Response `json:"responses"`
	Version     int        `json:"version"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

// Header represents an HTTP header
type Header struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Type        string `json:"type" binding:"required,oneof=string integer number boolean array object"`
}

// Param represents a request parameter (query or path)
type Param struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	In          string `json:"in" binding:"required,oneof=query path header"`
	Required    bool   `json:"required"`
	Type        string `json:"type" binding:"required,oneof=string integer number boolean array object"`
	Schema      string `json:"schema,omitempty"`
}

// Body represents a request or response body
type Body struct {
	ContentType string `json:"contentType" binding:"required"`
	Schema      string `json:"schema" binding:"required"`
	Example     string `json:"example,omitempty"`
}

// Response represents an API response
type Response struct {
	StatusCode  int    `json:"statusCode" binding:"required"`
	Description string `json:"description"`
	Body        *Body  `json:"body,omitempty"`
}

// ConvertToOpenAPI converts the HTTP interface to OpenAPI format
func (h *HTTPInterface) ConvertToOpenAPI() map[string]interface{} {
	// Create basic OpenAPI structure
	openAPI := map[string]interface{}{
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title":       h.Name,
			"description": h.Description,
			"version":     "1.0.0",
		},
		"paths": map[string]interface{}{},
	}

	// Add path
	pathData := map[string]interface{}{}
	method := strings.ToLower(h.Method)

	// Build operation object
	operation := map[string]interface{}{
		"summary":     h.Description,
		"description": h.Description,
		"operationId": h.Name,
	}

	// Add parameters
	if len(h.Parameters) > 0 {
		parameters := []map[string]interface{}{}
		for _, param := range h.Parameters {
			paramObj := map[string]interface{}{
				"name":        param.Name,
				"in":          param.In,
				"description": param.Description,
				"required":    param.Required,
				"schema": map[string]interface{}{
					"type": param.Type,
				},
			}
			parameters = append(parameters, paramObj)
		}
		operation["parameters"] = parameters
	}

	// Add headers as parameters
	if len(h.Headers) > 0 {
		if _, ok := operation["parameters"]; !ok {
			operation["parameters"] = []map[string]interface{}{}
		}
		parameters := operation["parameters"].([]map[string]interface{})

		for _, header := range h.Headers {
			headerParam := map[string]interface{}{
				"name":        header.Name,
				"in":          "header",
				"description": header.Description,
				"required":    header.Required,
				"schema": map[string]interface{}{
					"type": header.Type,
				},
			}
			parameters = append(parameters, headerParam)
		}
		operation["parameters"] = parameters
	}

	// Add request body
	if h.RequestBody != nil {
		var schema map[string]interface{}
		if err := json.Unmarshal([]byte(h.RequestBody.Schema), &schema); err != nil {
			schema = map[string]interface{}{"type": "object"}
		}

		var example interface{}
		if h.RequestBody.Example != "" {
			if err := json.Unmarshal([]byte(h.RequestBody.Example), &example); err != nil {
				example = h.RequestBody.Example
			}
		}

		requestBody := map[string]interface{}{
			"description": "Request body",
			"content": map[string]interface{}{
				h.RequestBody.ContentType: map[string]interface{}{
					"schema": schema,
				},
			},
		}

		if example != nil {
			content := requestBody["content"].(map[string]interface{})
			contentType := content[h.RequestBody.ContentType].(map[string]interface{})
			contentType["example"] = example
		}

		operation["requestBody"] = requestBody
	}

	// Add responses
	responses := map[string]interface{}{}
	for _, response := range h.Responses {
		respObj := map[string]interface{}{
			"description": response.Description,
		}

		// Add response body if present
		if response.Body != nil {
			var schema map[string]interface{}
			if err := json.Unmarshal([]byte(response.Body.Schema), &schema); err != nil {
				schema = map[string]interface{}{"type": "object"}
			}

			var example interface{}
			if response.Body.Example != "" {
				if err := json.Unmarshal([]byte(response.Body.Example), &example); err != nil {
					example = response.Body.Example
				}
			}

			content := map[string]interface{}{
				response.Body.ContentType: map[string]interface{}{
					"schema": schema,
				},
			}

			if example != nil {
				contentType := content[response.Body.ContentType].(map[string]interface{})
				contentType["example"] = example
			}

			respObj["content"] = content
		}

		responses[fmt.Sprintf("%d", response.StatusCode)] = respObj
	}
	operation["responses"] = responses

	// Add the operation to the path
	pathData[method] = operation
	paths := openAPI["paths"].(map[string]interface{})
	paths[h.Path] = pathData

	return openAPI
}

// CreateFromOpenAPI creates a new HTTP interface from OpenAPI specification
func CreateFromOpenAPI(name string, description string, openAPI map[string]interface{}) ([]HTTPInterface, error) {
	var interfaces []HTTPInterface

	// Extract paths from OpenAPI
	paths, ok := openAPI["paths"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid OpenAPI format: no paths found")
	}

	// Process each path
	for path, pathItemValue := range paths {
		pathItem, ok := pathItemValue.(map[string]interface{})
		if !ok {
			continue
		}

		// Process each HTTP method
		for method, operationValue := range pathItem {
			operation, ok := operationValue.(map[string]interface{})
			if !ok {
				continue
			}

			// Create the HTTP interface
			httpInterface := HTTPInterface{
				Name:        name + "-" + strings.ToLower(method) + "-" + sanitizePath(path),
				Description: description,
				Method:      strings.ToUpper(method),
				Path:        path,
				Headers:     []Header{},
				Parameters:  []Param{},
				Responses:   []Response{},
			}

			// Extract operation ID if present
			if opID, ok := operation["operationId"].(string); ok && opID != "" {
				httpInterface.Name = opID
			}

			// Extract parameters
			if parameters, ok := operation["parameters"].([]interface{}); ok {
				for _, paramValue := range parameters {
					param, ok := paramValue.(map[string]interface{})
					if !ok {
						continue
					}

					paramName, _ := param["name"].(string)
					paramIn, _ := param["in"].(string)
					paramDesc, _ := param["description"].(string)
					paramRequired, _ := param["required"].(bool)

					if paramIn == "header" {
						// Add header
						header := Header{
							Name:        paramName,
							Description: paramDesc,
							Required:    paramRequired,
							Type:        "string",
						}

						// Extract type from schema if present
						if schema, ok := param["schema"].(map[string]interface{}); ok {
							if paramType, ok := schema["type"].(string); ok {
								header.Type = paramType
							}
						}

						httpInterface.Headers = append(httpInterface.Headers, header)
					} else {
						// Add parameter
						parameter := Param{
							Name:        paramName,
							Description: paramDesc,
							In:          paramIn,
							Required:    paramRequired,
							Type:        "string",
						}

						// Extract type from schema if present
						if schema, ok := param["schema"].(map[string]interface{}); ok {
							if paramType, ok := schema["type"].(string); ok {
								parameter.Type = paramType
							}
						}

						httpInterface.Parameters = append(httpInterface.Parameters, parameter)
					}
				}
			}

			// Extract request body
			if requestBodyValue, ok := operation["requestBody"].(map[string]interface{}); ok {
				if content, ok := requestBodyValue["content"].(map[string]interface{}); ok {
					for contentType, contentValue := range content {
						contentObj, ok := contentValue.(map[string]interface{})
						if !ok {
							continue
						}

						// Create body
						body := &Body{
							ContentType: contentType,
							Schema:      `{"type": "object"}`,
						}

						// Extract schema
						if schema, ok := contentObj["schema"].(map[string]interface{}); ok {
							schemaJSON, err := json.Marshal(schema)
							if err == nil {
								body.Schema = string(schemaJSON)
							}
						}

						// Extract example
						if example, ok := contentObj["example"]; ok {
							exampleJSON, err := json.Marshal(example)
							if err == nil {
								body.Example = string(exampleJSON)
							}
						}

						httpInterface.RequestBody = body
						break // Use the first content type
					}
				}
			}

			// Extract responses
			if responsesValue, ok := operation["responses"].(map[string]interface{}); ok {
				for statusCode, responseValue := range responsesValue {
					responseObj, ok := responseValue.(map[string]interface{})
					if !ok {
						continue
					}

					// Try to parse status code
					code, err := strconv.Atoi(statusCode)
					if err != nil {
						continue
					}

					responseDesc, _ := responseObj["description"].(string)
					response := Response{
						StatusCode:  code,
						Description: responseDesc,
					}

					// Extract response body
					if content, ok := responseObj["content"].(map[string]interface{}); ok {
						for contentType, contentValue := range content {
							contentObj, ok := contentValue.(map[string]interface{})
							if !ok {
								continue
							}

							// Create body
							body := &Body{
								ContentType: contentType,
								Schema:      `{"type": "object"}`,
							}

							// Extract schema
							if schema, ok := contentObj["schema"].(map[string]interface{}); ok {
								schemaJSON, err := json.Marshal(schema)
								if err == nil {
									body.Schema = string(schemaJSON)
								}
							}

							// Extract example
							if example, ok := contentObj["example"]; ok {
								exampleJSON, err := json.Marshal(example)
								if err == nil {
									body.Example = string(exampleJSON)
								}
							}

							response.Body = body
							break // Use the first content type
						}
					}

					httpInterface.Responses = append(httpInterface.Responses, response)
				}
			}

			interfaces = append(interfaces, httpInterface)
		}
	}

	if len(interfaces) == 0 {
		return nil, fmt.Errorf("no valid HTTP interfaces found in OpenAPI spec")
	}

	return interfaces, nil
}

// Helper function to sanitize a path for use in an interface name
func sanitizePath(path string) string {
	// Replace slashes with hyphens and remove query parameters
	path = strings.ReplaceAll(path, "/", "-")
	path = strings.ReplaceAll(path, ":", "")
	path = strings.ReplaceAll(path, "{", "")
	path = strings.ReplaceAll(path, "}", "")
	path = strings.ReplaceAll(path, "?", "")
	path = strings.ReplaceAll(path, "&", "")
	path = strings.ReplaceAll(path, "=", "")
	path = strings.ReplaceAll(path, "*", "")

	// Remove leading and trailing hyphens
	path = strings.Trim(path, "-")

	// Return up to 30 characters
	if len(path) > 30 {
		return path[:30]
	}
	return path
}

// ConvertToMCPServerYAML converts the HTTP interface to MCP Server YAML format
func (h *HTTPInterface) ConvertToMCPServerYAML() string {
	// Implementation will be added later
	return ""
}
