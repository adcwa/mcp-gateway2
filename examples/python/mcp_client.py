#!/usr/bin/env python3
"""
MCP Client Example for Python

This example demonstrates how to use the MCP protocol to interact with an MCP server
hosted by MCP Gateway.
"""

import argparse
import json
import requests
from typing import Dict, List, Any, Optional


class MCPClient:
    """Simple MCP client implementation for Python."""

    def __init__(self, base_url: str, server_name: str):
        """
        Initialize the MCP client.
        
        Args:
            base_url: Base URL of the MCP Gateway (e.g., 'http://localhost:8080')
            server_name: Name of the MCP server to connect to
        """
        self.base_url = base_url.rstrip('/')
        self.server_name = server_name
        self.server_url = f"{self.base_url}/mcp-server/{server_name}"
        self.tools: List[Dict[str, Any]] = []
        self.resources: List[Dict[str, Any]] = []
        self.prompts: List[Dict[str, Any]] = []
        
    def initialize(self) -> None:
        """Initialize the client by fetching metadata from the MCP server."""
        # Fetch available tools
        self._fetch_tools()
        # Fetch available resources
        self._fetch_resources()
        # Fetch available prompts
        self._fetch_prompts()
        
    def _fetch_tools(self) -> None:
        """Fetch available tools from the MCP server."""
        url = f"{self.server_url}/tools"
        response = requests.get(url)
        if response.status_code == 200:
            self.tools = response.json()
            print(f"Fetched {len(self.tools)} tools from MCP server")
        else:
            print(f"Failed to fetch tools: {response.status_code} - {response.text}")
            
    def _fetch_resources(self) -> None:
        """Fetch available resources from the MCP server."""
        url = f"{self.server_url}/resources"
        response = requests.get(url)
        if response.status_code == 200:
            self.resources = response.json()
            print(f"Fetched {len(self.resources)} resources from MCP server")
        else:
            print(f"Failed to fetch resources: {response.status_code} - {response.text}")
            
    def _fetch_prompts(self) -> None:
        """Fetch available prompts from the MCP server."""
        url = f"{self.server_url}/prompts"
        response = requests.get(url)
        if response.status_code == 200:
            self.prompts = response.json()
            print(f"Fetched {len(self.prompts)} prompts from MCP server")
        else:
            print(f"Failed to fetch prompts: {response.status_code} - {response.text}")
            
    def get_tools(self) -> List[Dict[str, Any]]:
        """Get the list of available tools."""
        return self.tools
        
    def get_tool_names(self) -> List[str]:
        """Get the list of available tool names."""
        return [tool["name"] for tool in self.tools]
    
    def get_tool_info(self, tool_name: str) -> Optional[Dict[str, Any]]:
        """Get information about a specific tool."""
        for tool in self.tools:
            if tool["name"] == tool_name:
                return tool
        return None
    
    def invoke_tool(self, tool_name: str, params: Dict[str, Any] = None) -> Any:
        """
        Invoke a tool on the MCP server.
        
        Args:
            tool_name: Name of the tool to invoke
            params: Parameters to pass to the tool (optional)
            
        Returns:
            The tool result
        """
        if params is None:
            params = {}
            
        # Check if the tool exists
        tool_info = self.get_tool_info(tool_name)
        if tool_info is None:
            print(f"Tool '{tool_name}' not found")
            return None
            
        # Invoke the tool
        url = f"{self.server_url}/tools/{tool_name}"
        try:
            response = requests.post(url, json=params)
            if response.status_code == 200:
                return response.json()
            else:
                print(f"Failed to invoke tool: {response.status_code} - {response.text}")
                return None
        except Exception as e:
            print(f"Error invoking tool: {e}")
            return None


def main():
    """Main entry point for the example."""
    parser = argparse.ArgumentParser(description="MCP Client Example")
    parser.add_argument("--base-url", default="http://localhost:8080", help="Base URL of the MCP Gateway")
    parser.add_argument("--server-name", default="get-user", help="Name of the MCP server to connect to")
    parser.add_argument("--tool", help="Name of the tool to invoke")
    parser.add_argument("--params", help="JSON parameters for the tool")
    args = parser.parse_args()
    
    # Create MCP client
    client = MCPClient(args.base_url, args.server_name)
    
    # Initialize client
    client.initialize()
    
    # Print available tools
    print("Available tools:")
    for tool in client.get_tools():
        print(f"  - {tool['name']}: {tool['description']}")
        
    # Invoke tool if specified
    if args.tool:
        params = {}
        if args.params:
            try:
                params = json.loads(args.params)
            except json.JSONDecodeError:
                print(f"Invalid JSON parameters: {args.params}")
                return
                
        print(f"\nInvoking tool '{args.tool}' with parameters: {params}")
        result = client.invoke_tool(args.tool, params)
        print(f"Result: {json.dumps(result, indent=2)}")


if __name__ == "__main__":
    main() 