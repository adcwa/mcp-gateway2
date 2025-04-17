package com.mcpgateway.client;

import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStreamReader;
import java.io.OutputStream;
import java.net.HttpURLConnection;
import java.net.URL;
import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

import org.json.JSONArray;
import org.json.JSONException;
import org.json.JSONObject;

/**
 * MCP Client Example for Java
 * 
 * This example demonstrates how to use the MCP protocol to interact with an MCP server
 * hosted by MCP Gateway.
 */
public class MCPClient {
    private final String baseUrl;
    private final String serverName;
    private final String serverUrl;
    private final List<Tool> tools = new ArrayList<>();
    private final List<Resource> resources = new ArrayList<>();
    private final List<Prompt> prompts = new ArrayList<>();

    /**
     * Initialize the MCP client.
     * 
     * @param baseUrl    Base URL of the MCP Gateway (e.g., 'http://localhost:8080')
     * @param serverName Name of the MCP server to connect to
     */
    public MCPClient(String baseUrl, String serverName) {
        this.baseUrl = baseUrl.endsWith("/") ? baseUrl.substring(0, baseUrl.length() - 1) : baseUrl;
        this.serverName = serverName;
        this.serverUrl = this.baseUrl + "/mcp-server/" + serverName;
    }

    /**
     * Initialize the client by fetching metadata from the MCP server.
     * 
     * @throws IOException if an I/O error occurs
     */
    public void initialize() throws IOException {
        // Fetch available tools
        fetchTools();
        // Fetch available resources
        fetchResources();
        // Fetch available prompts
        fetchPrompts();
    }

    /**
     * Fetch available tools from the MCP server.
     * 
     * @throws IOException if an I/O error occurs
     */
    private void fetchTools() throws IOException {
        String url = serverUrl + "/tools";
        String response = sendRequest(url, "GET", null);
        try {
            JSONArray toolsArray = new JSONArray(response);
            tools.clear();
            for (int i = 0; i < toolsArray.length(); i++) {
                JSONObject toolObj = toolsArray.getJSONObject(i);
                Tool tool = new Tool();
                tool.name = toolObj.getString("name");
                tool.description = toolObj.getString("description");
                tools.add(tool);
            }
            System.out.println("Fetched " + tools.size() + " tools from MCP server");
        } catch (JSONException e) {
            System.err.println("Failed to parse tools: " + e.getMessage());
        }
    }

    /**
     * Fetch available resources from the MCP server.
     * 
     * @throws IOException if an I/O error occurs
     */
    private void fetchResources() throws IOException {
        String url = serverUrl + "/resources";
        String response = sendRequest(url, "GET", null);
        try {
            JSONArray resourcesArray = new JSONArray(response);
            resources.clear();
            for (int i = 0; i < resourcesArray.length(); i++) {
                JSONObject resourceObj = resourcesArray.getJSONObject(i);
                Resource resource = new Resource();
                resource.id = resourceObj.getString("id");
                resource.name = resourceObj.getString("name");
                resource.description = resourceObj.getString("description");
                resources.add(resource);
            }
            System.out.println("Fetched " + resources.size() + " resources from MCP server");
        } catch (JSONException e) {
            System.err.println("Failed to parse resources: " + e.getMessage());
        }
    }

    /**
     * Fetch available prompts from the MCP server.
     * 
     * @throws IOException if an I/O error occurs
     */
    private void fetchPrompts() throws IOException {
        String url = serverUrl + "/prompts";
        String response = sendRequest(url, "GET", null);
        try {
            JSONArray promptsArray = new JSONArray(response);
            prompts.clear();
            for (int i = 0; i < promptsArray.length(); i++) {
                JSONObject promptObj = promptsArray.getJSONObject(i);
                Prompt prompt = new Prompt();
                prompt.id = promptObj.getString("id");
                prompt.name = promptObj.getString("name");
                prompt.content = promptObj.getString("content");
                prompts.add(prompt);
            }
            System.out.println("Fetched " + prompts.size() + " prompts from MCP server");
        } catch (JSONException e) {
            System.err.println("Failed to parse prompts: " + e.getMessage());
        }
    }

    /**
     * Get the list of available tools.
     * 
     * @return the list of available tools
     */
    public List<Tool> getTools() {
        return tools;
    }

    /**
     * Get the list of available tool names.
     * 
     * @return the list of available tool names
     */
    public List<String> getToolNames() {
        List<String> names = new ArrayList<>();
        for (Tool tool : tools) {
            names.add(tool.name);
        }
        return names;
    }

    /**
     * Get information about a specific tool.
     * 
     * @param toolName Name of the tool to get information about
     * @return the tool information, or null if not found
     */
    public Tool getToolInfo(String toolName) {
        for (Tool tool : tools) {
            if (tool.name.equals(toolName)) {
                return tool;
            }
        }
        return null;
    }

    /**
     * Invoke a tool on the MCP server.
     * 
     * @param toolName Name of the tool to invoke
     * @param params   Parameters to pass to the tool (optional)
     * @return the tool result as a JSON object, or null on error
     * @throws IOException if an I/O error occurs
     */
    public JSONObject invokeTool(String toolName, Map<String, Object> params) throws IOException {
        // Check if the tool exists
        Tool toolInfo = getToolInfo(toolName);
        if (toolInfo == null) {
            System.err.println("Tool '" + toolName + "' not found");
            return null;
        }

        // Invoke the tool
        String url = serverUrl + "/tools/" + toolName;
        String jsonParams = params != null ? new JSONObject(params).toString() : "{}";
        String response = sendRequest(url, "POST", jsonParams);
        try {
            return new JSONObject(response);
        } catch (JSONException e) {
            System.err.println("Failed to parse tool response: " + e.getMessage());
            return null;
        }
    }

    /**
     * Send an HTTP request to the MCP server.
     * 
     * @param url     URL to send the request to
     * @param method  HTTP method (GET or POST)
     * @param payload Request payload (for POST requests)
     * @return the response as a string
     * @throws IOException if an I/O error occurs
     */
    private String sendRequest(String url, String method, String payload) throws IOException {
        HttpURLConnection connection = (HttpURLConnection) new URL(url).openConnection();
        connection.setRequestMethod(method);
        connection.setRequestProperty("Accept", "application/json");

        if ("POST".equals(method) && payload != null) {
            connection.setRequestProperty("Content-Type", "application/json");
            connection.setDoOutput(true);
            try (OutputStream os = connection.getOutputStream()) {
                byte[] input = payload.getBytes(StandardCharsets.UTF_8);
                os.write(input, 0, input.length);
            }
        }

        int responseCode = connection.getResponseCode();
        if (responseCode >= 200 && responseCode < 300) {
            StringBuilder response = new StringBuilder();
            try (BufferedReader br = new BufferedReader(
                    new InputStreamReader(connection.getInputStream(), StandardCharsets.UTF_8))) {
                String line;
                while ((line = br.readLine()) != null) {
                    response.append(line);
                }
            }
            return response.toString();
        } else {
            StringBuilder errorResponse = new StringBuilder();
            try (BufferedReader br = new BufferedReader(
                    new InputStreamReader(connection.getErrorStream(), StandardCharsets.UTF_8))) {
                String line;
                while ((line = br.readLine()) != null) {
                    errorResponse.append(line);
                }
            }
            throw new IOException("HTTP error code: " + responseCode + ", message: " + errorResponse.toString());
        }
    }

    /**
     * Main entry point for the example.
     * 
     * @param args Command line arguments
     */
    public static void main(String[] args) {
        // Parse command line arguments
        String baseUrl = "http://localhost:8080";
        String serverName = "get-user";
        String toolName = null;
        String params = null;

        for (int i = 0; i < args.length; i++) {
            switch (args[i]) {
                case "--base-url":
                    if (i + 1 < args.length) {
                        baseUrl = args[++i];
                    }
                    break;
                case "--server-name":
                    if (i + 1 < args.length) {
                        serverName = args[++i];
                    }
                    break;
                case "--tool":
                    if (i + 1 < args.length) {
                        toolName = args[++i];
                    }
                    break;
                case "--params":
                    if (i + 1 < args.length) {
                        params = args[++i];
                    }
                    break;
            }
        }

        try {
            // Create MCP client
            MCPClient client = new MCPClient(baseUrl, serverName);

            // Initialize client
            client.initialize();

            // Print available tools
            System.out.println("Available tools:");
            for (Tool tool : client.getTools()) {
                System.out.println("  - " + tool.name + ": " + tool.description);
            }

            // Invoke tool if specified
            if (toolName != null) {
                Map<String, Object> toolParams = new HashMap<>();
                if (params != null) {
                    try {
                        JSONObject jsonParams = new JSONObject(params);
                        for (String key : jsonParams.keySet()) {
                            toolParams.put(key, jsonParams.get(key));
                        }
                    } catch (JSONException e) {
                        System.err.println("Invalid JSON parameters: " + params);
                        return;
                    }
                }

                System.out.println("\nInvoking tool '" + toolName + "' with parameters: " + toolParams);
                JSONObject result = client.invokeTool(toolName, toolParams);
                System.out.println("Result: " + (result != null ? result.toString(2) : "null"));
            }
        } catch (Exception e) {
            System.err.println("Error: " + e.getMessage());
            e.printStackTrace();
        }
    }

    /**
     * Tool model class.
     */
    public static class Tool {
        public String name;
        public String description;
    }

    /**
     * Resource model class.
     */
    public static class Resource {
        public String id;
        public String name;
        public String description;
    }

    /**
     * Prompt model class.
     */
    public static class Prompt {
        public String id;
        public String name;
        public String content;
    }
} 