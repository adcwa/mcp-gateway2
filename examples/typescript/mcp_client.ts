/**
 * MCP Client Example for TypeScript
 * 
 * This example demonstrates how to use the MCP protocol to interact with an MCP server
 * hosted by MCP Gateway.
 */

import fetch from 'node-fetch';

interface Tool {
  name: string;
  description: string;
  parameters: {
    type: string;
    properties: Record<string, any>;
    required: string[];
  };
}

interface Resource {
  id: string;
  name: string;
  description: string;
  data: any;
}

interface Prompt {
  id: string;
  name: string;
  content: string;
}

class MCPClient {
  private baseUrl: string;
  private serverName: string;
  private serverUrl: string;
  private tools: Tool[] = [];
  private resources: Resource[] = [];
  private prompts: Prompt[] = [];

  /**
   * Initialize the MCP client.
   * 
   * @param baseUrl Base URL of the MCP Gateway (e.g., 'http://localhost:8080')
   * @param serverName Name of the MCP server to connect to
   */
  constructor(baseUrl: string, serverName: string) {
    this.baseUrl = baseUrl.endsWith('/') ? baseUrl.slice(0, -1) : baseUrl;
    this.serverName = serverName;
    this.serverUrl = `${this.baseUrl}/mcp-server/${serverName}`;
  }

  /**
   * Initialize the client by fetching metadata from the MCP server.
   */
  async initialize(): Promise<void> {
    // Fetch available tools
    await this.fetchTools();
    // Fetch available resources
    await this.fetchResources();
    // Fetch available prompts
    await this.fetchPrompts();
  }

  /**
   * Fetch available tools from the MCP server.
   */
  private async fetchTools(): Promise<void> {
    const url = `${this.serverUrl}/tools`;
    try {
      const response = await fetch(url);
      if (response.ok) {
        this.tools = await response.json();
        console.log(`Fetched ${this.tools.length} tools from MCP server`);
      } else {
        console.error(`Failed to fetch tools: ${response.status} - ${await response.text()}`);
      }
    } catch (error) {
      console.error(`Error fetching tools: ${error}`);
    }
  }

  /**
   * Fetch available resources from the MCP server.
   */
  private async fetchResources(): Promise<void> {
    const url = `${this.serverUrl}/resources`;
    try {
      const response = await fetch(url);
      if (response.ok) {
        this.resources = await response.json();
        console.log(`Fetched ${this.resources.length} resources from MCP server`);
      } else {
        console.error(`Failed to fetch resources: ${response.status} - ${await response.text()}`);
      }
    } catch (error) {
      console.error(`Error fetching resources: ${error}`);
    }
  }

  /**
   * Fetch available prompts from the MCP server.
   */
  private async fetchPrompts(): Promise<void> {
    const url = `${this.serverUrl}/prompts`;
    try {
      const response = await fetch(url);
      if (response.ok) {
        this.prompts = await response.json();
        console.log(`Fetched ${this.prompts.length} prompts from MCP server`);
      } else {
        console.error(`Failed to fetch prompts: ${response.status} - ${await response.text()}`);
      }
    } catch (error) {
      console.error(`Error fetching prompts: ${error}`);
    }
  }

  /**
   * Get the list of available tools.
   */
  getTools(): Tool[] {
    return this.tools;
  }

  /**
   * Get the list of available tool names.
   */
  getToolNames(): string[] {
    return this.tools.map(tool => tool.name);
  }

  /**
   * Get information about a specific tool.
   * 
   * @param toolName Name of the tool to get information about
   */
  getToolInfo(toolName: string): Tool | undefined {
    return this.tools.find(tool => tool.name === toolName);
  }

  /**
   * Invoke a tool on the MCP server.
   * 
   * @param toolName Name of the tool to invoke
   * @param params Parameters to pass to the tool (optional)
   */
  async invokeTool(toolName: string, params?: Record<string, any>): Promise<any> {
    // Check if the tool exists
    const toolInfo = this.getToolInfo(toolName);
    if (!toolInfo) {
      console.error(`Tool '${toolName}' not found`);
      return null;
    }

    // Invoke the tool
    const url = `${this.serverUrl}/tools/${toolName}`;
    try {
      const response = await fetch(url, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(params || {}),
      });
      
      if (response.ok) {
        return await response.json();
      } else {
        console.error(`Failed to invoke tool: ${response.status} - ${await response.text()}`);
        return null;
      }
    } catch (error) {
      console.error(`Error invoking tool: ${error}`);
      return null;
    }
  }
}

/**
 * Main entry point for the example.
 */
async function main() {
  // Parse command line arguments
  const args = parseArgs();
  
  // Create MCP client
  const client = new MCPClient(args.baseUrl, args.serverName);
  
  // Initialize client
  await client.initialize();
  
  // Print available tools
  console.log('Available tools:');
  for (const tool of client.getTools()) {
    console.log(`  - ${tool.name}: ${tool.description}`);
  }
  
  // Invoke tool if specified
  if (args.tool) {
    let params = {};
    if (args.params) {
      try {
        params = JSON.parse(args.params);
      } catch (error) {
        console.error(`Invalid JSON parameters: ${args.params}`);
        return;
      }
    }
    
    console.log(`\nInvoking tool '${args.tool}' with parameters:`, params);
    const result = await client.invokeTool(args.tool, params);
    console.log('Result:', JSON.stringify(result, null, 2));
  }
}

/**
 * Parse command line arguments.
 */
function parseArgs() {
  // Default values
  const args = {
    baseUrl: 'http://localhost:8080',
    serverName: 'get-user',
    tool: '',
    params: '',
  };
  
  // Parse command line arguments
  for (let i = 2; i < process.argv.length; i++) {
    const arg = process.argv[i];
    if (arg === '--base-url' && i + 1 < process.argv.length) {
      args.baseUrl = process.argv[++i];
    } else if (arg === '--server-name' && i + 1 < process.argv.length) {
      args.serverName = process.argv[++i];
    } else if (arg === '--tool' && i + 1 < process.argv.length) {
      args.tool = process.argv[++i];
    } else if (arg === '--params' && i + 1 < process.argv.length) {
      args.params = process.argv[++i];
    }
  }
  
  return args;
}

// Run the main function
main().catch(error => {
  console.error('Unhandled error:', error);
  process.exit(1);
}); 