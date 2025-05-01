# Containerlab MCP Server

This is a quick and dirty MCP (Model Context Protocol) trial for interacting with containerlab using AI. The example is tailored for claude desktop.

1. Running [containerlab API server](https://github.com/srl-labs/clab-api-server)
2. Latest build of [containerlab](https://github.com/srl-labs/containerlab) from the main branch
   - Build it yourself: `make build`
   - Copy the binary: `cp bin/containerlab $(which containerlab)`

## Building the MCP Server

### For Windows
```bash
export GOOS=windows
export GOARCH=amd64
go build -o clab-mcp-server.exe main.go
```

### For Mac
```bash
# For Intel Macs
export GOOS=darwin
export GOARCH=amd64
go build -o clab-mcp-server main.go

# For Apple Silicon (M1/M2/M3)
export GOOS=darwin
export GOARCH=arm64
go build -o clab-mcp-server main.go
```

### For Linux
```bash
export GOOS=linux
export GOARCH=amd64
go build -o clab-mcp-server main.go
```

## Setup for Claude Desktop

1. Place the MCP server executable in an accessible location
   - For Windows: `clab-mcp-server.exe`
   - For Mac/Linux: `clab-mcp-server`
2. Create a configuration file named `claude_desktop_config.json` with the following content:

```json
# For Windows:
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

# For Mac/Linux:
{
  "mcpServers": {
    "clab-api": {
      "command": "/path/to/clab-mcp-server",
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