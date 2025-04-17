# MCP Client Examples

This directory contains example clients for accessing MCP servers through the MCP Gateway in different programming languages.

## Overview

These clients demonstrate how to interact with MCP servers using the Model Context Protocol. Each client can:

1. Fetch MCP server metadata (tools, resources, prompts)
2. Invoke tools on the MCP server
3. Process results from tool invocations

## Prerequisites

- Running instance of MCP Gateway (default: http://localhost:8080)
- At least one active MCP server in the gateway

## Examples

### Python Client

#### Requirements
- Python 3.6+
- `requests` library

#### Installation
```bash
cd python
pip install requests
```

#### Usage
```bash
python mcp_client.py --base-url http://localhost:8080 --server-name get-user --tool get-user
```

### TypeScript Client

#### Requirements
- Node.js 14+
- npm

#### Installation
```bash
cd typescript
npm install
npm run build
```

#### Usage
```bash
node mcp_client.js --base-url http://localhost:8080 --server-name get-user --tool get-user
```

### Java Client

#### Requirements
- Java 11+
- Maven

#### Installation
```bash
cd java
mvn clean package
```

#### Usage
```bash
java -jar target/mcp-client-example-1.0.0.jar --base-url http://localhost:8080 --server-name get-user --tool get-user
```

## Command Line Options

All clients support the following command line options:

- `--base-url <url>`: Base URL of the MCP Gateway (default: http://localhost:8080)
- `--server-name <name>`: Name of the MCP server to connect to (default: get-user)
- `--tool <name>`: Name of the tool to invoke
- `--params <json>`: JSON parameters for the tool invocation

## Example Usage

### Invoke the Random User API

```bash
# Python
python mcp_client.py --tool get-user

# TypeScript
node mcp_client.js --tool get-user

# Java
java -jar target/mcp-client-example-1.0.0.jar --tool get-user
```

### Invoke the Weather API with Parameters

```bash
# Python
python mcp_client.py --server-name get-weather --tool get-weather --params '{"q":"New York","appid":"your_api_key"}'

# TypeScript
node mcp_client.js --server-name get-weather --tool get-weather --params '{"q":"New York","appid":"your_api_key"}'

# Java
java -jar target/mcp-client-example-1.0.0.jar --server-name get-weather --tool get-weather --params '{"q":"New York","appid":"your_api_key"}'
```

## MCP Protocol Reference

For more information on the Model Context Protocol, see:
- https://modelcontextprotocol.io/specification/2025-03-26/server/tools
- https://modelcontextprotocol.io/specification/2025-03-26/server/resources
- https://modelcontextprotocol.io/specification/2025-03-26/server/prompts