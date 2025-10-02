package rack

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"
)

// MCPMessage represents a JSON-RPC message for MCP protocol
type MCPMessage struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Method  string      `json:"method,omitempty"`
	Params  interface{} `json:"params,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

// MCPError represents an error in MCP protocol
type MCPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// MCPTool represents a tool provided by an MCP server
type MCPTool struct {
	name        string
	description string
	schema      Schema
	server      *MCPServer
}

func (t *MCPTool) Name() string        { return t.name }
func (t *MCPTool) Description() string { return t.description }
func (t *MCPTool) Schema() Schema      { return t.schema }

func (t *MCPTool) Execute(ctx context.Context, input json.RawMessage) (*ToolResult, error) {
	result, err := t.server.CallTool(ctx, t.name, input)
	if err != nil {
		return &ToolResult{Error: err.Error()}, nil
	}

	return &ToolResult{
		Content: result.Content,
		Error:   result.Error,
		Metadata: map[string]any{
			"mcp_server": t.server.name,
			"tool_name":  t.name,
		},
	}, nil
}

// MCPServer manages connection to an MCP server
type MCPServer struct {
	name    string
	command string
	args    []string
	env     []string

	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser

	mu      sync.Mutex
	nextID  int
	pending map[int]chan MCPMessage
	tools   map[string]*MCPTool
	running bool
}

// MCPToolResult represents the result from an MCP tool execution
type MCPToolResult struct {
	Content string `json:"content"`
	Error   string `json:"error,omitempty"`
	IsError bool   `json:"isError,omitempty"`
}

// MCPServerConfig holds configuration for an MCP server
type MCPServerConfig struct {
	Name    string   `json:"name"`
	Command string   `json:"command"`
	Args    []string `json:"args"`
	Env     []string `json:"env"`
}

// NewMCPServer creates a new MCP server instance
func NewMCPServer(name, command string, args []string, env []string) *MCPServer {
	return &MCPServer{
		name:    name,
		command: command,
		args:    args,
		env:     env,
		pending: make(map[int]chan MCPMessage),
		tools:   make(map[string]*MCPTool),
	}
}

// Start initializes and starts the MCP server
func (s *MCPServer) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("server already running")
	}

	// Start the MCP server process
	s.cmd = exec.CommandContext(ctx, s.command, s.args...)
	s.cmd.Env = append(os.Environ(), s.env...)

	stdin, err := s.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	s.stdin = stdin

	stdout, err := s.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	s.stdout = stdout

	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start MCP server: %w", err)
	}

	s.running = true

	// Start reading responses
	go s.readResponses()

	// Initialize the MCP connection
	if err := s.initialize(ctx); err != nil {
		s.Stop()
		return fmt.Errorf("failed to initialize MCP server: %w", err)
	}

	return nil
}

// Stop terminates the MCP server
func (s *MCPServer) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	s.running = false

	if s.stdin != nil {
		s.stdin.Close()
	}
	if s.stdout != nil {
		s.stdout.Close()
	}
	if s.cmd != nil && s.cmd.Process != nil {
		s.cmd.Process.Kill()
		s.cmd.Wait()
	}

	return nil
}

// LoadTools discovers and loads tools from the MCP server
func (s *MCPServer) LoadTools(ctx context.Context) error {
	listReq := MCPMessage{
		JSONRPC: "2.0",
		ID:      s.getNextID(),
		Method:  "tools/list",
		Params:  map[string]interface{}{},
	}

	response, err := s.sendRequest(ctx, listReq)
	if err != nil {
		return fmt.Errorf("failed to list tools: %w", err)
	}

	if response.Error != nil {
		return fmt.Errorf("list tools error: %s", response.Error.Message)
	}

	// Parse tools from response
	result, ok := response.Result.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid tools response format")
	}

	toolsArray, ok := result["tools"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid tools array format")
	}

	for _, toolInterface := range toolsArray {
		toolData, ok := toolInterface.(map[string]interface{})
		if !ok {
			continue
		}

		name, _ := toolData["name"].(string)
		description, _ := toolData["description"].(string)

		// Convert input schema
		schema := Schema{Type: "object", Properties: make(map[string]SchemaField)}
		if inputSchema, ok := toolData["inputSchema"].(map[string]interface{}); ok {
			if props, ok := inputSchema["properties"].(map[string]interface{}); ok {
				for propName, propDef := range props {
					if propDefMap, ok := propDef.(map[string]interface{}); ok {
						field := SchemaField{}
						if propType, ok := propDefMap["type"].(string); ok {
							field.Type = propType
						}
						if propDesc, ok := propDefMap["description"].(string); ok {
							field.Description = propDesc
						}
						schema.Properties[propName] = field
					}
				}
			}
			if required, ok := inputSchema["required"].([]interface{}); ok {
				for _, req := range required {
					if reqStr, ok := req.(string); ok {
						schema.Required = append(schema.Required, reqStr)
					}
				}
			}
		}

		tool := &MCPTool{
			name:        name,
			description: description,
			schema:      schema,
			server:      s,
		}

		s.tools[name] = tool
	}

	return nil
}

// CallTool executes a tool on the MCP server
func (s *MCPServer) CallTool(ctx context.Context, toolName string, arguments json.RawMessage) (*MCPToolResult, error) {
	callReq := MCPMessage{
		JSONRPC: "2.0",
		ID:      s.getNextID(),
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name":      toolName,
			"arguments": json.RawMessage(arguments),
		},
	}

	response, err := s.sendRequest(ctx, callReq)
	if err != nil {
		return nil, fmt.Errorf("tool call failed: %w", err)
	}

	if response.Error != nil {
		return &MCPToolResult{
			Content: response.Error.Message,
			Error:   response.Error.Message,
			IsError: true,
		}, nil
	}

	// Parse the response
	result, ok := response.Result.(map[string]interface{})
	if !ok {
		return &MCPToolResult{
			Content: "Invalid response format",
			Error:   "Invalid response format",
			IsError: true,
		}, nil
	}

	content := ""
	if contentArray, ok := result["content"].([]interface{}); ok && len(contentArray) > 0 {
		if contentItem, ok := contentArray[0].(map[string]interface{}); ok {
			if text, ok := contentItem["text"].(string); ok {
				content = text
			}
		}
	}

	return &MCPToolResult{
		Content: content,
		IsError: false,
	}, nil
}

// GetTools returns all tools from this MCP server
func (s *MCPServer) GetTools() []Tool {
	s.mu.Lock()
	defer s.mu.Unlock()

	tools := make([]Tool, 0, len(s.tools))
	for _, tool := range s.tools {
		tools = append(tools, tool)
	}
	return tools
}

func (s *MCPServer) initialize(ctx context.Context) error {
	initReq := MCPMessage{
		JSONRPC: "2.0",
		ID:      s.getNextID(),
		Method:  "initialize",
		Params: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"clientInfo": map[string]interface{}{
				"name":    "rack-go",
				"version": "1.0.0",
			},
		},
	}

	response, err := s.sendRequest(ctx, initReq)
	if err != nil {
		return fmt.Errorf("initialize failed: %w", err)
	}

	if response.Error != nil {
		return fmt.Errorf("initialize error: %s", response.Error.Message)
	}

	// Send initialized notification
	initNotification := MCPMessage{
		JSONRPC: "2.0",
		Method:  "initialized",
		Params:  map[string]interface{}{},
	}

	return s.sendNotification(initNotification)
}

func (s *MCPServer) sendRequest(ctx context.Context, msg MCPMessage) (*MCPMessage, error) {
	id := msg.ID.(int)

	// Create response channel
	respChan := make(chan MCPMessage, 1)
	s.mu.Lock()
	s.pending[id] = respChan
	s.mu.Unlock()

	// Clean up when done
	defer func() {
		s.mu.Lock()
		delete(s.pending, id)
		s.mu.Unlock()
	}()

	// Send the message
	if err := s.sendMessage(msg); err != nil {
		return nil, err
	}

	// Wait for response
	select {
	case response := <-respChan:
		return &response, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("request timeout")
	}
}

func (s *MCPServer) sendNotification(msg MCPMessage) error {
	return s.sendMessage(msg)
}

func (s *MCPServer) sendMessage(msg MCPMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return fmt.Errorf("server not running")
	}

	_, err = s.stdin.Write(append(data, '\n'))
	return err
}

func (s *MCPServer) readResponses() {
	scanner := bufio.NewScanner(s.stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var msg MCPMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue // Skip invalid messages
		}

		// Handle response
		if msg.ID != nil {
			if id, ok := msg.ID.(float64); ok {
				s.mu.Lock()
				if ch, exists := s.pending[int(id)]; exists {
					select {
					case ch <- msg:
					default:
					}
				}
				s.mu.Unlock()
			}
		}
	}
}

func (s *MCPServer) getNextID() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	return s.nextID
}

// NewMCPToolFromServer creates tools from an MCP server configuration
func NewMCPToolFromServer(serverConfig MCPServerConfig) ([]Tool, error) {
	server := NewMCPServer(
		serverConfig.Name,
		serverConfig.Command,
		serverConfig.Args,
		serverConfig.Env,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start MCP server: %w", err)
	}

	if err := server.LoadTools(ctx); err != nil {
		server.Stop()
		return nil, fmt.Errorf("failed to load tools: %w", err)
	}

	return server.GetTools(), nil
}
