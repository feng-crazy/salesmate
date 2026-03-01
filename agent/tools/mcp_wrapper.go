package tools

import (
	"context"
	"fmt"
	"salesmate/agent/mcp"
)

// MCPTollWrapper wraps an MCP server tool as a native nanobot tool
type MCPToolWrapper struct {
	session      *mcp.MCPSession
	serverName   string
	toolDef      mcp.ToolDefinition
	timeout      int
	manager      *mcp.MCPServerManager // Need to keep reference to manager to find sessions
	origToolName string                // Original tool name without prefix
}

// NewMCPToolWrapper creates a new MCP tool wrapper
func NewMCPToolWrapper(session *mcp.MCPSession, serverName string, origToolName string, toolDef mcp.ToolDefinition, timeout int, manager *mcp.MCPServerManager) *MCPToolWrapper {
	return &MCPToolWrapper{
		session:      session,
		serverName:   serverName,
		origToolName: origToolName,
		toolDef:      toolDef,
		timeout:      timeout,
		manager:      manager,
	}
}

// Name returns the tool name with server prefix
func (mtw *MCPToolWrapper) Name() string {
	return fmt.Sprintf("mcp_%s_%s", mtw.serverName, mtw.origToolName)
}

// Description returns the tool description
func (mtw *MCPToolWrapper) Description() string {
	return mtw.toolDef.Description
}

// Call executes the tool with the given arguments
func (mtw *MCPToolWrapper) Call(args map[string]interface{}) (string, error) {
	// Create context with timeout
	ctx := context.Background()
	// In a real implementation, we'd use a proper timeout context

	// Find the appropriate session using the manager
	session, exists := mtw.manager.GetSessionByName(mtw.serverName)
	if !exists {
		return fmt.Sprintf("Error: MCP server %s not found", mtw.serverName), nil
	}

	// Call the MCP server
	result, err := session.CallTool(ctx, mtw.origToolName, args)
	if err != nil {
		return fmt.Sprintf("Error calling MCP tool: %v", err), nil
	}

	// Format the result appropriately
	return fmt.Sprintf("MCP tool result: %v", result), nil
}

// ConnectMCPServers connects to configured MCP servers and registers their tools
func ConnectMCPServers(mcpServers map[string]interface{}, registry *ToolRegistry) error {
	manager := mcp.NewMCPServerManager()

	for name, cfg := range mcpServers {
		serverCfg := mcp.MCPServer{
			Name: name,
		}

		// Parse the config based on the type (could be map with command/args or URL)
		if cfgMap, ok := cfg.(map[string]interface{}); ok {
			if cmd, exists := cfgMap["command"].(string); exists {
				serverCfg.Command = cmd
				// Parse args if they exist
				if argsIf, exists := cfgMap["args"]; exists {
					if argsSlice, ok := argsIf.([]interface{}); ok {
						for _, arg := range argsSlice {
							if argStr, ok := arg.(string); ok {
								serverCfg.Args = append(serverCfg.Args, argStr)
							}
						}
					}
				}

				// Parse environment if it exists
				if envIf, exists := cfgMap["env"]; exists {
					if envMap, ok := envIf.(map[string]interface{}); ok {
						serverCfg.Env = make(map[string]string)
						for k, v := range envMap {
							if vStr, ok := v.(string); ok {
								serverCfg.Env[k] = vStr
							}
						}
					}
				}
			} else if url, exists := cfgMap["url"].(string); exists {
				serverCfg.URL = url
				// Parse headers if they exist
				if headersIf, exists := cfgMap["headers"]; exists {
					if headersMap, ok := headersIf.(map[string]interface{}); ok {
						serverCfg.Headers = make(map[string]string)
						for k, v := range headersMap {
							if vStr, ok := v.(string); ok {
								serverCfg.Headers[k] = vStr
							}
						}
					}
				}
			} else {
				continue // Skip if neither command nor URL is provided
			}

			// Set timeout if provided
			if timeoutIf, exists := cfgMap["toolTimeout"]; exists {
				if timeoutFloat, ok := timeoutIf.(float64); ok {
					serverCfg.Timeout = int(timeoutFloat)
				}
			} else {
				serverCfg.Timeout = 30 // Default timeout of 30 seconds
			}
		}

		// Add server to manager
		if err := manager.AddServer(serverCfg); err != nil {
			continue
		}
	}

	// Connect all servers
	ctx := context.Background()
	if err := manager.ConnectAll(ctx); err != nil {
		return fmt.Errorf("failed to connect to MCP servers: %v", err)
	}

	// Get tools from all servers and register them
	tools, err := manager.GetTools(ctx)
	if err != nil {
		return fmt.Errorf("failed to get tools from MCP servers: %v", err)
	}

	// For each tool, register it with the appropriate session
	// We'll iterate through each server separately to ensure proper session references
	for serverName, session := range manager.GetSessions() {
		// Get tools specifically for this server by filtering
		serverTools := getToolsForServer(tools, serverName)

		for _, toolDef := range serverTools {
			// Extract original tool name (remove server prefix)
			origToolName := toolDef.Name
			if len(origToolName) > len(serverName)+5 { // mcp_ + serverName + _
				origToolName = toolDef.Name[len("mcp_"+serverName+"_"):]
			}

			wrapper := NewMCPToolWrapper(session, serverName, origToolName, toolDef, 30, manager)
			registry.Register(wrapper)
		}
	}

	return nil
}

// getToolsForServer filters tools for a specific server
func getToolsForServer(allTools []mcp.ToolDefinition, serverName string) []mcp.ToolDefinition {
	var serverTools []mcp.ToolDefinition
	prefix := fmt.Sprintf("mcp_%s_", serverName)

	for _, tool := range allTools {
		if len(tool.Name) >= len(prefix) && tool.Name[:len(prefix)] == prefix {
			serverTools = append(serverTools, tool)
		}
	}

	return serverTools
}
