package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	mcp_golang "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/stdio"
)

// Configuration for the API server
type Config struct {
	ApiServerURL string `json:"apiServerURL"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	Token        string `json:"token"`
	TokenExpiry  time.Time
}

// Global config
var config Config

// API client
type ApiClient struct {
	client *http.Client
}

// Create a new API client
func NewApiClient() *ApiClient {
	return &ApiClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type deployLabRequest struct {
	TopologyContent map[string]interface{} `json:"topologyContent"`
}

// Helper function to create error responses
func createErrorResponse(message string) *mcp_golang.ToolResponse {
	return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(fmt.Sprintf("ERROR: %s", message)))
}

// Ensure we have a valid token
func (c *ApiClient) ensureAuth() error {
	// If token is valid, return
	if config.Token != "" && config.TokenExpiry.After(time.Now()) {
		return nil
	}

	// Otherwise, login
	loginURL := fmt.Sprintf("%s/login", config.ApiServerURL)
	loginReq := struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}{
		Username: config.Username,
		Password: config.Password,
	}

	reqBody, err := json.Marshal(loginReq)
	if err != nil {
		return fmt.Errorf("failed to marshal login request: %w", err)
	}

	resp, err := c.client.Post(loginURL, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("login failed with status %d: %s", resp.StatusCode, body)
	}

	var loginResp struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return fmt.Errorf("failed to decode login response: %w", err)
	}

	config.Token = loginResp.Token
	config.TokenExpiry = time.Now().Add(50 * time.Minute) // Token usually valid for 60 minutes
	return nil
}

// Make authenticated API request
func (c *ApiClient) makeRequest(method, path string, body interface{}, queryParams map[string]string) (*http.Response, error) {
	if err := c.ensureAuth(); err != nil {
		return nil, err
	}

	// Build URL with query parameters
	baseURL, err := url.Parse(config.ApiServerURL)
	if err != nil {
		return nil, fmt.Errorf("invalid API server URL: %w", err)
	}

	// Add the path
	baseURL.Path = path

	// Add query parameters if provided
	if len(queryParams) > 0 {
		q := url.Values{}
		for key, value := range queryParams {
			q.Add(key, value)
		}
		baseURL.RawQuery = q.Encode()
	}

	finalURL := baseURL.String()

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, finalURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+config.Token)

	return c.client.Do(req)
}

func init() {
	// Load config from environment variables
	config.ApiServerURL = getEnv("API_SERVER_URL", "http://localhost:8080")
	config.Username = getEnv("API_USERNAME", "admin")
	config.Password = getEnv("API_PASSWORD", "password")
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// Authentication arguments
type AuthArgs struct {
	ApiServerURL string `json:"apiServerURL" jsonschema:"required,description=The URL of the API server (e.g., http://localhost:8080/login)"`
	Username     string `json:"username" jsonschema:"required,description=Username for authentication"`
	Password     string `json:"password" jsonschema:"required,description=Password for authentication"`
}

// Lab listing arguments (might include filters)
type ListLabsArgs struct {
	// Optional filters could be added here
}

// Lab deployment arguments
type DeployLabArgs struct {
	TopologyContent map[string]interface{} `json:"topologyContent" jsonschema:"required,type=object,description=A JSON object containing the topology"`
	Reconfigure     bool                   `json:"reconfigure" jsonschema:"description=Whether to reconfigure an existing lab"`
}

// Lab inspection arguments
type InspectLabArgs struct {
	LabName string `json:"labName" jsonschema:"required,description=Name of the lab to inspect"`
	Details bool   `json:"details" jsonschema:"description=Include detailed container information"`
}

// Command execution arguments
type ExecCommandArgs struct {
	LabName  string `json:"labName" jsonschema:"required,description=Name of the lab"`
	NodeName string `json:"nodeName" jsonschema:"description=Name of the specific node (optional)"`
	Command  string `json:"command" jsonschema:"required,description=Command to execute"`
}

// Lab destruction arguments
type DestroyLabArgs struct {
	LabName  string `json:"labName" jsonschema:"required,description=Name of the lab to destroy"`
	Cleanup  bool   `json:"cleanup" jsonschema:"description=Remove lab directory after destroy"`
	Graceful bool   `json:"graceful" jsonschema:"description=Attempt graceful shutdown of containers"`
}

func main() {
	// Print startup message to stderr, not stdout
	fmt.Fprintln(os.Stderr, "Containerlab MCP Server started")
	fmt.Fprintln(os.Stderr, "Configured API URL:", config.ApiServerURL)
	fmt.Fprintln(os.Stderr, "Waiting for Claude to connect...")

	apiClient := NewApiClient()
	server := mcp_golang.NewServer(stdio.NewStdioServerTransport())

	// Register tools
	registerTools(server, apiClient)

	// Start the server
	err := server.Serve()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error starting server: %v\n", err)
		os.Exit(1)
	}

	// Keep the server running
	select {}
}

// Register all tools with the server
func registerTools(server *mcp_golang.Server, apiClient *ApiClient) {
	// Authentication tool
	// Authentication tool
	err := server.RegisterTool("authenticate",
		`Authenticate with the containerlab API server.

    The tool will make a POST request to [apiServerURL]/login with the provided credentials.

    Example input:
    {
        "apiServerURL": "http://localhost:8080",
        "username": "admin",
        "password": "password"
    }

    API endpoint format:
    POST http://localhost:8080/login
    {
        "username": "string",
        "password": "string"
    }`,
		func(args AuthArgs) (*mcp_golang.ToolResponse, error) {
			// Update global config
			config.ApiServerURL = args.ApiServerURL
			config.Username = args.Username
			config.Password = args.Password
			config.Token = "" // Clear existing token

			// Test authentication
			err := apiClient.ensureAuth()
			if err != nil {
				return createErrorResponse(fmt.Sprintf("Authentication failed: %v", err)), nil
			}

			return mcp_golang.NewToolResponse(mcp_golang.NewTextContent("Successfully authenticated with the API server")), nil
		})

	// List Labs tool
	err = server.RegisterTool("listLabs", "List all available labs",
		func(args ListLabsArgs) (*mcp_golang.ToolResponse, error) {
			resp, err := apiClient.makeRequest("GET", "/api/v1/labs", nil, nil)
			if err != nil {
				return createErrorResponse(fmt.Sprintf("Failed to list labs: %v", err)), nil
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				return createErrorResponse(fmt.Sprintf("Failed to list labs: %s", body)), nil
			}

			var labsResponse interface{}
			if err := json.NewDecoder(resp.Body).Decode(&labsResponse); err != nil {
				return createErrorResponse(fmt.Sprintf("Failed to parse labs response: %v", err)), nil
			}

			// Format the response nicely
			labsJSON, _ := json.MarshalIndent(labsResponse, "", "  ")
			return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(string(labsJSON))), nil
		})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error registering list labs tool: %v\n", err)
	}

	// Deploy Lab tool
	err = server.RegisterTool("deployLab",
		`Deploy a new lab with the provided topology.

    IMPORTANT: topologyContent must be a proper JSON object, NOT a string with escaped quotes.

    Example input:
    {"topologyContent":{"name":"srl01","topology":{"kinds":{"nokia_srlinux":{"type":"ixrd3","image":"ghcr.io/nokia/srlinux"}},"nodes":{"srl1":{"kind":"nokia_srlinux"},"srl2":{"kind":"nokia_srlinux"}},"links":[{"endpoints":["srl1:e1-1","srl2:e1-1"]}]}}}`,
		func(args DeployLabArgs) (*mcp_golang.ToolResponse, error) {
			// Build query parameters
			queryParams := make(map[string]string)
			if args.Reconfigure {
				queryParams["reconfigure"] = "true"
			}

			reqBody := deployLabRequest{
				TopologyContent: args.TopologyContent,
			}
			resp, err := apiClient.makeRequest("POST", "/api/v1/labs", reqBody, queryParams)
			if err != nil {
				return createErrorResponse(fmt.Sprintf("Failed to deploy lab: %v", err)), nil
			}
			defer resp.Body.Close()

			body, _ := io.ReadAll(resp.Body)
			if resp.StatusCode != http.StatusOK {
				return createErrorResponse(fmt.Sprintf("Failed to deploy lab: %s", body)), nil
			}

			return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(fmt.Sprintf("Successfully deployed lab.\nResponse: %s", body))), nil
		})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error registering deploy lab tool: %v\n", err)
	}

	// Inspect Lab tool
	err = server.RegisterTool("inspectLab", "Get details about a specific lab",
		func(args InspectLabArgs) (*mcp_golang.ToolResponse, error) {
			path := fmt.Sprintf("/api/v1/labs/%s", args.LabName)

			queryParams := make(map[string]string)
			if args.Details {
				queryParams["details"] = "true"
			}

			resp, err := apiClient.makeRequest("GET", path, nil, queryParams)
			if err != nil {
				return createErrorResponse(fmt.Sprintf("Failed to inspect lab: %v", err)), nil
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				return createErrorResponse(fmt.Sprintf("Failed to inspect lab: %s", body)), nil
			}

			var labInfo interface{}
			if err := json.NewDecoder(resp.Body).Decode(&labInfo); err != nil {
				return createErrorResponse(fmt.Sprintf("Failed to parse lab info: %v", err)), nil
			}

			labJSON, _ := json.MarshalIndent(labInfo, "", "  ")
			return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(string(labJSON))), nil
		})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error registering inspect lab tool: %v\n", err)
	}

	// Execute Command tool
	err = server.RegisterTool("execCommand", "Execute a command on lab nodes",
		func(args ExecCommandArgs) (*mcp_golang.ToolResponse, error) {
			path := fmt.Sprintf("/api/v1/labs/%s/exec", args.LabName)

			queryParams := make(map[string]string)
			if args.NodeName != "" {
				queryParams["nodeFilter"] = args.NodeName
			}

			commandReq := struct {
				Command string `json:"command"`
			}{
				Command: args.Command,
			}

			resp, err := apiClient.makeRequest("POST", path, commandReq, queryParams)
			if err != nil {
				return createErrorResponse(fmt.Sprintf("Failed to execute command: %v", err)), nil
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				return createErrorResponse(fmt.Sprintf("Failed to execute command: %s", body)), nil
			}

			var cmdResponse interface{}
			if err := json.NewDecoder(resp.Body).Decode(&cmdResponse); err != nil {
				body, _ := io.ReadAll(resp.Body)
				return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(string(body))), nil
			}

			cmdJSON, _ := json.MarshalIndent(cmdResponse, "", "  ")
			return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(string(cmdJSON))), nil
		})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error registering execute command tool: %v\n", err)
	}

	// Destroy Lab tool
	err = server.RegisterTool("destroyLab", "Destroy and clean up a lab",
		func(args DestroyLabArgs) (*mcp_golang.ToolResponse, error) {
			path := fmt.Sprintf("/api/v1/labs/%s", args.LabName)

			// Add query parameters
			queryParams := make(map[string]string)
			if args.Cleanup {
				queryParams["cleanup"] = "true"
			}
			if args.Graceful {
				queryParams["graceful"] = "true"
			}

			resp, err := apiClient.makeRequest("DELETE", path, nil, queryParams)
			if err != nil {
				return createErrorResponse(fmt.Sprintf("Failed to destroy lab: %v", err)), nil
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				return createErrorResponse(fmt.Sprintf("Failed to destroy lab: %s", body)), nil
			}

			var destroyResponse struct {
				Message string `json:"message"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&destroyResponse); err != nil {
				return createErrorResponse(fmt.Sprintf("Failed to parse destroy response: %v", err)), nil
			}

			return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(destroyResponse.Message)), nil
		})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error registering destroy lab tool: %v\n", err)
	}
}
