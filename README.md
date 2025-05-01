# Containerlab MCP Server

This is a quick and dirty MCP (Model Control Protocol) trial for interacting with containerlab using Claude. This integration allows Claude to create, manage, and interact with network labs through the containerlab API server.

## Requirements

1. Running [containerlab API server](https://github.com/srl-labs/clab-api-server)
2. Latest build of [containerlab](https://github.com/srl-labs/containerlab) from the main branch
   - Build it yourself: `make build`
   - Copy the binary: `cp bin/containerlab $(which containerlab)`

## Setup for Claude Desktop

1. Place the MCP server executable (`clab-mcp-server.exe`) in an accessible location
2. Create a configuration file named `claude_desktop_config.json` with the following content:

```json
{
  "mcpServers": {
    "clab-api": {
      "command": "C:\\clab-mcp-server.exe",
      "args": [],
      "env": {
        "API_SERVER_URL": "http://localhost:8080"
      }
    }
  }
}
```

3. Ensure the containerlab API server is running at http://localhost:8080 or update the URL as needed

## Usage

Once configured, Claude will be able to:
- List available labs
- Deploy new network topologies
- Inspect lab details
- Execute commands on lab nodes
- Destroy and clean up labs

You can ask Claude to perform these operations using natural language, and it will use the appropriate MCP tools to interact with your containerlab environment.