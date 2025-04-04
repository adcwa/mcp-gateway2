# MCP-Gateway

MCP-Gateway is a service that provides MCP Server unified management capabilities, helping AI Agents quickly connect to various data sources. Through MCP Server, AI Agents can easily access databases, REST APIs, and other external services without worrying about specific connection details.

## Features

- HTTP interface management: Support for exporting OpenAPI structures, converting HTTP to MCP Server YAML format, and versioning of interfaces.
  - Convert curl commands to HTTP interfaces: Easily transform curl commands into properly formatted HTTP interfaces.
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

## API Documentation

### HTTP Interfaces

- `GET /api/http-interfaces`: List all HTTP interfaces
- `GET /api/http-interfaces/:id`: Get a specific HTTP interface
- `POST /api/http-interfaces`: Create a new HTTP interface
- `PUT /api/http-interfaces/:id`: Update an HTTP interface
- `DELETE /api/http-interfaces/:id`: Delete an HTTP interface
- `GET /api/http-interfaces/:id/versions`: Get all versions of an HTTP interface
- `GET /api/http-interfaces/:id/versions/:version`: Get a specific version of an HTTP interface
- `GET /api/http-interfaces/:id/openapi`: Get OpenAPI specification for an HTTP interface
- `POST /api/http-interfaces/from-curl`: Create a new HTTP interface from a curl command

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

## License

MIT

## References

- [Higress MCP Quick Start](https://higress.cn/ai/mcp-quick-start/)
- [Higress MCP Server](https://higress.cn/ai/mcp-server/)
- [GJSON Template](https://github.com/higress-group/gjson_template) 