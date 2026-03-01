package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// MCPServer represents an MCP server configuration
type MCPServer struct {
	Name    string
	Command string
	Args    []string
	URL     string
	Headers map[string]string
	Env     map[string]string
	Timeout int
}

// TransportType defines the type of transport to use for MCP connections
type TransportType string

const (
	StdioTransport   TransportType = "stdio"
	HTTPTransport    TransportType = "http"
	WebSocketTransport TransportType = "websocket"
)

// MCPSession represents a connection to an MCP server
type MCPSession struct {
	Server     *MCPServer
	transport  TransportType
	stdinCmd   *exec.Cmd // For stdio transport
	wsConn     *websocket.Conn // For websocket transport
	httpClient *http.Client // For http transport
	writer     io.Writer
	reader     io.Reader
	reqID      int
	mu         sync.Mutex
	activeRequests map[int]chan json.RawMessage
}

// Connect connects to an MCP server using the appropriate transport
func (ms *MCPSession) Connect(ctx context.Context) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if ms.Server.Command != "" {
		ms.transport = StdioTransport
		return ms.connectViaStdio(ctx)
	} else if ms.Server.URL != "" {
		// Determine transport based on URL scheme
		if strings.HasPrefix(ms.Server.URL, "ws://") || strings.HasPrefix(ms.Server.URL, "wss://") {
			ms.transport = WebSocketTransport
			return ms.connectViaWebSocket(ctx)
		} else {
			ms.transport = HTTPTransport
			return ms.connectViaHTTP(ctx)
		}
	}

	return fmt.Errorf("neither command nor URL provided for MCP server %s", ms.Server.Name)
}

// connectViaStdio connects to an MCP server via stdio
func (ms *MCPSession) connectViaStdio(ctx context.Context) error {
	log.Printf("Connecting to MCP server %s via stdio", ms.Server.Name)

	// Set up the command with proper environment
	cmd := exec.CommandContext(ctx, ms.Server.Command, ms.Server.Args...)

	// Set environment variables if provided
	if len(ms.Server.Env) > 0 {
		env := make([]string, 0, len(ms.Server.Env))
		for k, v := range ms.Server.Env {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		cmd.Env = append(cmd.Environ(), env...)
	}

	// Get stdin and stdout pipes
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start MCP server: %w", err)
	}

	// Set up reader and writer
	ms.stdinCmd = cmd
	ms.reader = stdout
	ms.writer = stdin
	ms.activeRequests = make(map[int]chan json.RawMessage)

	// Start reading responses in a goroutine
	go ms.readStdioResponses(stdout)

	return nil
}

// readStdioResponses reads responses from the MCP server (stdio version)
func (ms *MCPSession) readStdioResponses(reader io.Reader) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		var response map[string]interface{}
		line := scanner.Bytes()

		if err := json.Unmarshal(line, &response); err != nil {
			log.Printf("Error decoding MCP response: %v", err)
			continue
		}

		// Handle the response - check if it's a response to a request we made
		if idVal, ok := response["id"]; ok {
			idFloat, ok := idVal.(float64)
			if !ok {
				continue
			}
			id := int(idFloat)

			ms.mu.Lock()
			if ch, exists := ms.activeRequests[id]; exists {
				ch <- line
				delete(ms.activeRequests, id)
			}
			ms.mu.Unlock()
		}
	}
}

// connectViaWebSocket connects to an MCP server via WebSocket
func (ms *MCPSession) connectViaWebSocket(ctx context.Context) error {
	log.Printf("Connecting to MCP server %s via WebSocket: %s", ms.Server.Name, ms.Server.URL)

	headers := make(http.Header)
	for k, v := range ms.Server.Headers {
		headers.Set(k, v)
	}

	conn, _, err := websocket.DefaultDialer.Dial(ms.Server.URL, headers)
	if err != nil {
		return fmt.Errorf("failed to dial WebSocket: %w", err)
	}

	ms.wsConn = conn
	ms.activeRequests = make(map[int]chan json.RawMessage)

	// Start reading responses in a goroutine
	go ms.readWebSocketResponses(conn)

	return nil
}

// readWebSocketResponses reads responses from the MCP server (WebSocket version)
func (ms *MCPSession) readWebSocketResponses(conn *websocket.Conn) {
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading WebSocket message: %v", err)
			return
		}

		var response map[string]interface{}
		if err := json.Unmarshal(message, &response); err != nil {
			log.Printf("Error decoding MCP WebSocket response: %v", err)
			continue
		}

		// Handle the response - check if it's a response to a request we made
		if idVal, ok := response["id"]; ok {
			idFloat, ok := idVal.(float64)
			if !ok {
				continue
			}
			id := int(idFloat)

			ms.mu.Lock()
			if ch, exists := ms.activeRequests[id]; exists {
				ch <- message
				delete(ms.activeRequests, id)
			}
			ms.mu.Unlock()
		}
	}
}

// connectViaHTTP connects to an MCP server via HTTP (SSE)
func (ms *MCPSession) connectViaHTTP(ctx context.Context) error {
	log.Printf("Connecting to MCP server %s via HTTP: %s", ms.Server.Name, ms.Server.URL)

	client := &http.Client{
		Timeout: time.Duration(ms.Server.Timeout) * time.Second,
	}

	ms.httpClient = client
	ms.activeRequests = make(map[int]chan json.RawMessage)

	return nil
}

// sendRequest sends a JSON-RPC request to the MCP server
func (ms *MCPSession) sendRequest(method string, params interface{}) (json.RawMessage, error) {
	ms.mu.Lock()
	ms.reqID++
	id := ms.reqID

	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
		"params":  params,
	}

	// Create channel to receive response
	responseChan := make(chan json.RawMessage, 1)
	ms.activeRequests[id] = responseChan
	ms.mu.Unlock()

	// Send the request based on transport type
	var sendErr error
	switch ms.transport {
	case StdioTransport:
		sendErr = ms.sendStdioRequest(req, responseChan)
	case WebSocketTransport:
		sendErr = ms.sendWebSocketRequest(req, responseChan)
	case HTTPTransport:
		sendErr = ms.sendHTTPRequest(req, responseChan)
	default:
		return nil, fmt.Errorf("unsupported transport type: %s", ms.transport)
	}

	if sendErr != nil {
		ms.mu.Lock()
		delete(ms.activeRequests, id)
		ms.mu.Unlock()
		return nil, fmt.Errorf("failed to send request: %w", sendErr)
	}

	// Wait for response with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(ms.Server.Timeout)*time.Second)
	defer cancel()

	select {
	case response := <-responseChan:
		return response, nil
	case <-ctx.Done():
		ms.mu.Lock()
		delete(ms.activeRequests, id)
		ms.mu.Unlock()
		return nil, fmt.Errorf("request timeout after %ds", ms.Server.Timeout)
	}
}

// sendStdioRequest sends a request via stdio transport
func (ms *MCPSession) sendStdioRequest(req map[string]interface{}, responseChan chan json.RawMessage) error {
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}

	// Add newline as MCP expects line-delimited JSON
	data = append(data, '\n')

	_, err = ms.writer.Write(data)
	return err
}

// sendWebSocketRequest sends a request via WebSocket transport
func (ms *MCPSession) sendWebSocketRequest(req map[string]interface{}, responseChan chan json.RawMessage) error {
	return ms.wsConn.WriteJSON(req)
}

// sendHTTPRequest sends a request via HTTP transport
func (ms *MCPSession) sendHTTPRequest(req map[string]interface{}, responseChan chan json.RawMessage) error {
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequest("POST", ms.Server.URL, strings.NewReader(string(data)))
	if err != nil {
		return err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	for k, v := range ms.Server.Headers {
		httpReq.Header.Set(k, v)
	}

	resp, err := ms.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	responseData, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var response map[string]interface{}
	if err := json.Unmarshal(responseData, &response); err != nil {
		return err
	}

	// Send response to the waiting channel
	responseChan <- responseData

	return nil
}

// ListTools lists available tools from the MCP server
func (ms *MCPSession) ListTools(ctx context.Context) ([]ToolDefinition, error) {
	response, err := ms.sendRequest("tools/list", map[string]interface{}{})
	if err != nil {
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}

	var result struct {
		Result struct {
			Tools []struct {
				Name        string                 `json:"name"`
				Description string                 `json:"description"`
				InputSchema map[string]interface{} `json:"inputSchema"`
			} `json:"tools"`
		} `json:"result"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
		Jsonrpc string `json:"jsonrpc"`
	}

	if err := json.Unmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("failed to decode list tools response: %w", err)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("MCP server error %d: %s", result.Error.Code, result.Error.Message)
	}

	// Convert to our internal format
	tools := make([]ToolDefinition, len(result.Result.Tools))
	for i, tool := range result.Result.Tools {
		tools[i] = ToolDefinition{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: tool.InputSchema,
		}
	}

	return tools, nil
}

// CallTool calls a specific tool on the MCP server
func (ms *MCPSession) CallTool(ctx context.Context, toolName string, arguments map[string]interface{}) (interface{}, error) {
	params := map[string]interface{}{
		"name":      toolName,
		"arguments": arguments,
	}

	response, err := ms.sendRequest("tools/call", params)
	if err != nil {
		return nil, fmt.Errorf("failed to call tool %s: %w", toolName, err)
	}

	var result struct {
		Result interface{} `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
		Jsonrpc string `json:"jsonrpc"`
	}

	if err := json.Unmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("failed to decode call tool response: %w", err)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("MCP server error %d: %s", result.Error.Code, result.Error.Message)
	}

	return result.Result, nil
}

// Initialize performs the MCP initialization handshake
func (ms *MCPSession) Initialize(ctx context.Context) error {
	params := map[string]interface{}{
		"capabilities": map[string]interface{}{},
	}

	response, err := ms.sendRequest("initialize", params)
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	var result struct {
		Result struct {
			ServerCapabilities map[string]interface{} `json:"serverCapabilities"`
		} `json:"result"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
		Jsonrpc string `json:"jsonrpc"`
	}

	if err := json.Unmarshal(response, &result); err != nil {
		return fmt.Errorf("failed to decode initialize response: %w", err)
	}

	if result.Error != nil {
		return fmt.Errorf("MCP server initialization error %d: %s", result.Error.Code, result.Error.Message)
	}

	log.Printf("MCP server %s initialized successfully", ms.Server.Name)
	return nil
}

// Close closes the MCP session
func (ms *MCPSession) Close() error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	switch ms.transport {
	case StdioTransport:
		if ms.stdinCmd != nil {
			return ms.stdinCmd.Process.Kill()
		}
	case WebSocketTransport:
		if ms.wsConn != nil {
			return ms.wsConn.Close()
		}
	case HTTPTransport:
		if ms.httpClient != nil {
			// HTTP clients don't need explicit closing for individual requests
		}
	}

	return nil
}

// ToolDefinition represents an MCP tool definition
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// MCPServerManager manages multiple MCP servers
type MCPServerManager struct {
	servers map[string]*MCPSession
	mu      sync.RWMutex
}

// NewMCPServerManager creates a new MCP server manager
func NewMCPServerManager() *MCPServerManager {
	return &MCPServerManager{
		servers: make(map[string]*MCPSession),
	}
}

// AddServer adds an MCP server to the manager
func (mm *MCPServerManager) AddServer(server MCPServer) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	session := &MCPSession{
		Server: &server,
	}

	mm.servers[server.Name] = session
	return nil
}

// ConnectAll connects to all configured MCP servers
func (mm *MCPServerManager) ConnectAll(ctx context.Context) error {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	var lastErr error
	for name, session := range mm.servers {
		if err := session.Connect(ctx); err != nil {
			log.Printf("Failed to connect to MCP server %s: %v", name, err)
			lastErr = err
			continue
		}

		// Perform initialization handshake
		if err := session.Initialize(ctx); err != nil {
			log.Printf("Failed to initialize MCP server %s: %v", name, err)
			session.Close()
			lastErr = err
			continue
		}
	}
	return lastErr
}

// GetTools gets all tools from all connected MCP servers
func (mm *MCPServerManager) GetTools(ctx context.Context) ([]ToolDefinition, error) {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	var allTools []ToolDefinition

	for name, session := range mm.servers {
		tools, err := session.ListTools(ctx)
		if err != nil {
			log.Printf("Failed to list tools from MCP server %s: %v", name, err)
			continue
		}

		// Add server name prefix to tool names to make them unique
		for _, tool := range tools {
			tool.Name = fmt.Sprintf("mcp_%s_%s", name, tool.Name)
			allTools = append(allTools, tool)
		}
	}

	return allTools, nil
}

// CallTool calls a tool on the appropriate MCP server
func (mm *MCPServerManager) CallTool(ctx context.Context, fullToolName string, arguments map[string]interface{}) (interface{}, error) {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	// Extract server name and actual tool name from fullToolName
	// Expected format: "mcp_{serverName}_{actualToolName}"
	if len(fullToolName) < 5 || !hasPrefix(fullToolName, "mcp_") {
		return nil, fmt.Errorf("invalid MCP tool name format: %s", fullToolName)
	}

	// Remove the "mcp_" prefix and find the next underscore to extract server name
	remaining := fullToolName[4:] // Remove "mcp_"
	parts := splitN(remaining, "_", 2)

	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid MCP tool name format: %s", fullToolName)
	}

	serverName := parts[0]
	actualToolName := parts[1]

	session, exists := mm.servers[serverName]
	if !exists {
		return nil, fmt.Errorf("MCP server %s not found", serverName)
	}

	return session.CallTool(ctx, actualToolName, arguments)
}

// GetSessionByName returns a session by server name
func (mm *MCPServerManager) GetSessionByName(name string) (*MCPSession, bool) {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	session, exists := mm.servers[name]
	return session, exists
}

// GetSessions returns all sessions
func (mm *MCPServerManager) GetSessions() map[string]*MCPSession {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	// Create a copy of the map to prevent race conditions
	sessions := make(map[string]*MCPSession, len(mm.servers))
	for k, v := range mm.servers {
		sessions[k] = v
	}
	return sessions
}

// CloseAll closes all MCP sessions
func (mm *MCPServerManager) CloseAll() {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	for _, session := range mm.servers {
		session.Close()
	}
}

// Helper functions for string operations
func splitN(s, sep string, n int) []string {
	if n == 0 {
		return nil
	}
	if n == 1 {
		return []string{s}
	}

	var result []string
	start := 0
	for i := 0; i < n-1; i++ {
		pos := indexOf(s[start:], sep)
		if pos == -1 {
			break
		}
		pos += start
		result = append(result, s[start:pos])
		start = pos + len(sep)
	}
	result = append(result, s[start:])
	return result
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}