package api

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"project-tachyon/internal/engine"
	"sync"
)

// MCPServer implements a basic JSON-RPC 2.0 handler for Model Context Protocol
// It reads from an io.Reader and writes to an io.Writer (defaults to Stdin/Stdout).
type MCPServer struct {
	engine *engine.TachyonEngine
	mu     sync.Mutex
	writer io.Writer
}

func NewMCPServer(engine *engine.TachyonEngine) *MCPServer {
	return &MCPServer{
		engine: engine,
		writer: os.Stdout,
	}
}

// NewMCPServerWithIO creates an MCPServer that reads/writes to the given streams.
// This is used for testing without touching os.Stdin/os.Stdout.
func NewMCPServerWithIO(engine *engine.TachyonEngine, w io.Writer) *MCPServer {
	return &MCPServer{
		engine: engine,
		writer: w,
	}
}

// Start blocks and processes messages from Stdin
func (s *MCPServer) Start() {
	s.StartWithReader(os.Stdin)
}

// StartWithReader processes messages from the given reader until EOF.
func (s *MCPServer) StartWithReader(r io.Reader) {
	log.SetOutput(os.Stderr)
	log.Printf("MCP Server Started. Listening...")

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		s.handleMessage(line)
	}

	if err := scanner.Err(); err != nil {
		log.Printf("MCP Scan error: %v", err)
	}
}

type JsonRpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	ID      interface{}     `json:"id"`
}

type JsonRpcResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RpcError   `json:"error,omitempty"`
	ID      interface{} `json:"id"`
}

type RpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (s *MCPServer) handleMessage(data []byte) {
	var req JsonRpcRequest
	if err := json.Unmarshal(data, &req); err != nil {
		s.sendError(nil, -32700, "Parse error")
		return
	}

	switch req.Method {
	// --- MCP lifecycle ---
	case "initialize":
		s.handleInitialize(req)
	case "notifications/initialized":
		// Acknowledgement from client — no response required
	// --- MCP tool discovery & invocation ---
	case "tools/list":
		s.handleToolsList(req)
	case "tools/call":
		s.handleToolCall(req)
	default:
		s.sendError(req.ID, -32601, "Method not found")
	}
}

func (s *MCPServer) sendResponse(id interface{}, result interface{}) {
	resp := JsonRpcResponse{
		JSONRPC: "2.0",
		Result:  result,
		ID:      id,
	}
	s.write(resp)
}

func (s *MCPServer) sendError(id interface{}, code int, message string) {
	resp := JsonRpcResponse{
		JSONRPC: "2.0",
		Error: &RpcError{
			Code:    code,
			Message: message,
		},
		ID: id,
	}
	s.write(resp)
}

func (s *MCPServer) write(resp JsonRpcResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()
	bytes, _ := json.Marshal(resp)
	fmt.Fprintf(s.writer, "%s\n", bytes)
}

// Handlers

// handleInitialize responds to the MCP initialize handshake.
func (s *MCPServer) handleInitialize(req JsonRpcRequest) {
	s.sendResponse(req.ID, map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{},
		},
		"serverInfo": map[string]interface{}{
			"name":    "tachyon",
			"version": "1.0.0",
		},
	})
}

// ToolCallParams is the outer envelope for tools/call.
type ToolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// handleToolCall dispatches tools/call to the correct handler.
func (s *MCPServer) handleToolCall(req JsonRpcRequest) {
	var params ToolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.sendError(req.ID, -32602, "Invalid params")
		return
	}

	switch params.Name {
	case "tachyon_download":
		s.handleDownload(req.ID, params.Arguments)
	case "tachyon_list":
		s.handleList(req.ID)
	default:
		s.sendError(req.ID, -32602, "Unknown tool: "+params.Name)
	}
}

// sendToolResult is the MCP-compliant way to return tool output.
func (s *MCPServer) sendToolResult(id interface{}, text string, isError bool) {
	s.sendResponse(id, map[string]interface{}{
		"content": []map[string]interface{}{
			{"type": "text", "text": text},
		},
		"isError": isError,
	})
}

type DownloadParams struct {
	URL      string `json:"url"`
	Path     string `json:"path"`
	Filename string `json:"filename"`
}

func (s *MCPServer) handleDownload(id interface{}, args json.RawMessage) {
	var params DownloadParams
	if err := json.Unmarshal(args, &params); err != nil {
		s.sendToolResult(id, "Invalid params: "+err.Error(), true)
		return
	}

	if params.URL == "" {
		s.sendToolResult(id, "URL is required", true)
		return
	}

	if err := engine.ValidateURL(params.URL); err != nil {
		s.sendToolResult(id, err.Error(), true)
		return
	}
	params.Filename = engine.SanitizeFilename(params.Filename)

	taskID, err := s.engine.StartDownload(params.URL, params.Path, params.Filename, nil)
	if err != nil {
		s.sendToolResult(id, "Download failed: "+err.Error(), true)
		return
	}

	result := fmt.Sprintf("Download queued successfully.\nTask ID: %s\nURL: %s", taskID, params.URL)
	s.sendToolResult(id, result, false)
}

func (s *MCPServer) handleList(id interface{}) {
	tasks, err := s.engine.GetHistory()
	if err != nil {
		s.sendToolResult(id, "Failed to list downloads: "+err.Error(), true)
		return
	}

	var lines []string
	for _, t := range tasks {
		if t.Status == "downloading" || t.Status == "pending" || t.Status == "paused" {
			line := fmt.Sprintf("- [%s] %s — %s (%.1f%%, %.0f B/s, ETA: %s)",
				t.ID[:8], t.Filename, t.Status, t.Progress, t.Speed, t.TimeRemaining)
			lines = append(lines, line)
		}
	}

	if len(lines) == 0 {
		s.sendToolResult(id, "No active downloads.", false)
		return
	}

	text := fmt.Sprintf("Active downloads (%d):\n%s", len(lines), joinLines(lines))
	s.sendToolResult(id, text, false)
}

func joinLines(lines []string) string {
	result := ""
	for i, l := range lines {
		if i > 0 {
			result += "\n"
		}
		result += l
	}
	return result
}

// handleToolsList responds to MCP tool discovery
func (s *MCPServer) handleToolsList(req JsonRpcRequest) {
	tools := []map[string]interface{}{
		{
			"name":        "tachyon_download",
			"description": "Download a file using Tachyon High-Performance Engine",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"url":      map[string]string{"type": "string", "description": "URL to download"},
					"path":     map[string]string{"type": "string", "description": "Destination path (optional)"},
					"filename": map[string]string{"type": "string", "description": "Custom filename (optional)"},
				},
				"required": []string{"url"},
			},
		},
		{
			"name":        "tachyon_list",
			"description": "List active downloads",
			"inputSchema": map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
	}

	s.sendResponse(req.ID, map[string]interface{}{
		"tools": tools,
	})
}
