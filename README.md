# MCP-Gateway

MCP-Gateway is a service that provides MCP Server unified management capabilities, helping AI Agents quickly connect to various data sources. Through MCP Server, AI Agents can easily access databases, REST APIs, and other external services without worrying about specific connection details.

## Features

- HTTP interface management: Support for exporting OpenAPI structures, converting HTTP to MCP Server YAML format, and versioning of interfaces.
  - Convert curl commands to HTTP interfaces: Easily transform curl commands into properly formatted HTTP interfaces.
  - Import/Export OpenAPI: Convert between HTTP interfaces and OpenAPI specifications for easy integration with existing API frameworks.
- MCP Server management: Support for managing MCP Server metadata, selecting multiple HTTP structures to update metadata, publishing MCP Servers (compiling to WebAssembly for dynamic loading), and version control.
- Routing management: Support for route configuration, such as matching `xxx/mcp-server/{name}` to MCP Server with name `{name}`.

## Architecture

MCP-Gateway consists of three core modules:

1. **HTTP Interface Management**: Defines and manages API configurations.
2. **MCP Server Management**: Manages MCP Server instances, compiles them to WebAssembly, and handles runtime execution.
3. **Routing Management**: Manages routing rules for MCP Servers.

## Getting Started

### Prerequisites

- Go 1.19 or higher
- Git

### Installation

1. Clone the repository:
   ```
   git clone https://github.com/wangfeng/mcp-gateway2.git
   cd mcp-gateway2
   ```

2. Install dependencies:
   ```
   go mod tidy
   ```

3. Run the server:
   ```
   go run cmd/server/main.go
   ```

4. The server will start on port 8080 by default. You can customize the port by setting the `PORT` environment variable.

### Testing

To test the API, run the test client:

```
go run test/client.go
```

This will:
1. List available HTTP interfaces
2. Create an MCP Server using one of the interfaces
3. Compile the MCP Server to WebAssembly
4. Activate the MCP Server
5. Invoke a tool on the MCP Server

To test the curl conversion feature, run the curl test client:

```
go run test/curl/curl_client.go
```

This will:
1. Convert several curl commands to HTTP interfaces
2. Create an MCP Server using the converted interfaces
3. Compile and activate the MCP Server
4. Invoke a tool from the MCP Server

To test the OpenAPI conversion feature, run the OpenAPI test client:

```
go run test/openapi/openapi_client.go
```

This will:
1. Export an existing HTTP interface to OpenAPI format
2. Import a sample OpenAPI specification to create new HTTP interfaces
3. Perform a round-trip conversion (export to OpenAPI and import back)

## API Documentation

### HTTP Interfaces

- `GET /api/http-interfaces`: List all HTTP interfaces
- `GET /api/http-interfaces/:id`: Get a specific HTTP interface
- `POST /api/http-interfaces`: Create a new HTTP interface
- `PUT /api/http-interfaces/:id`: Update an HTTP interface
- `DELETE /api/http-interfaces/:id`: Delete an HTTP interface
- `GET /api/http-interfaces/:id/versions`: Get all versions of an HTTP interface
- `GET /api/http-interfaces/:id/versions/:version`: Get a specific version of an HTTP interface
- `GET /api/http-interfaces/:id/openapi`: Export an HTTP interface to OpenAPI format
- `POST /api/http-interfaces/from-curl`: Create a new HTTP interface from a curl command
- `POST /api/http-interfaces/from-openapi`: Create new HTTP interfaces from an OpenAPI specification

### MCP Servers

- `GET /api/mcp-servers`: List all MCP Servers
- `GET /api/mcp-servers/:id`: Get a specific MCP Server
- `POST /api/mcp-servers`: Create a new MCP Server from HTTP interfaces
- `PUT /api/mcp-servers/:id`: Update an MCP Server
- `DELETE /api/mcp-servers/:id`: Delete an MCP Server
- `GET /api/mcp-servers/:id/versions`: Get all versions of an MCP Server
- `GET /api/mcp-servers/:id/versions/:version`: Get a specific version of an MCP Server
- `POST /api/mcp-servers/:id/compile`: Compile an MCP Server to WebAssembly
- `POST /api/mcp-servers/:id/activate`: Activate an MCP Server
- `POST /api/mcp-servers/:id/tools/:tool`: Invoke a tool in an MCP Server

## Curl to HTTP Interface Conversion

The system supports converting curl commands to HTTP interfaces. Simply send a POST request to `/api/http-interfaces/from-curl` with the following JSON body:

```json
{
  "command": "curl -H \"Content-Type: application/json\" https://api.example.com/resource",
  "name": "example-api",
  "description": "Example API endpoint"
}
```

The system will parse the curl command and create a properly formatted HTTP interface that can be used to create MCP Servers.

## OpenAPI Conversion

### Export to OpenAPI

You can export any HTTP interface to OpenAPI format by sending a GET request to `/api/http-interfaces/:id/openapi`. The response will be a properly formatted OpenAPI 3.0.0 specification that can be used with other OpenAPI tools.

### Import from OpenAPI

You can create new HTTP interfaces from an OpenAPI specification by sending a POST request to `/api/http-interfaces/from-openapi` with the following JSON body:
```shell
 go run test/openapi/openapi_client.go
```
```json
{
  "name": "my-api",
  "description": "My API description",
  "spec": {
    "openapi": "3.0.0",
    "info": {
      "title": "Sample API",
      "description": "A sample API",
      "version": "1.0.0"
    },
    "paths": {
      "/users": {
        "get": {
          "summary": "Get all users",
          "responses": {
            "200": {
              "description": "A list of users"
            }
          }
        }
      }
    }
  }
}
```

The system will parse the OpenAPI specification and create HTTP interfaces for each path/operation combination.

## License

MIT

## References

- [Higress MCP Quick Start](https://higress.cn/ai/mcp-quick-start/)
- [Higress MCP Server](https://higress.cn/ai/mcp-server/)
- [GJSON Template](https://github.com/higress-group/gjson_template) 