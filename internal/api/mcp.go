package api

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"project-tachyon/internal/engine"
	"sync"
)

// MCPServer implements a basic JSON-RPC 2.0 handler for Model Context Protocol
// It listens on Stdin and writes to Stdout
type MCPServer struct {
	engine *engine.TachyonEngine
	mu     sync.Mutex
}

func NewMCPServer(engine *engine.TachyonEngine) *MCPServer {
	return &MCPServer{
		engine: engine,
	}
}

// Start blocks and processes messages from Stdin
func (s *MCPServer) Start() {
	// Disable standard logger to avoid polluting stdout (which is used for RPC)
	log.SetOutput(os.Stderr)
	log.Printf("MCP Server Started. Listening on Stdin...")

	scanner := bufio.NewScanner(os.Stdin)
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

	// Handle Tools
	switch req.Method {
	case "tachyon_download":
		s.handleDownload(req)
	case "tachyon_list":
		s.handleList(req)
	case "tools/list": // Standard MCP discovery
		s.handleToolsList(req)
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
	fmt.Fprintf(os.Stdout, "%s\n", bytes)
}

// Handlers

type DownloadParams struct {
	URL      string `json:"url"`
	Path     string `json:"path"`
	Filename string `json:"filename"`
}

func (s *MCPServer) handleDownload(req JsonRpcRequest) {
	var params DownloadParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.sendError(req.ID, -32602, "Invalid params")
		return
	}

	if params.URL == "" {
		s.sendError(req.ID, -32602, "URL is required")
		return
	}

	id, err := s.engine.StartDownload(params.URL, params.Path, params.Filename, nil)
	if err != nil {
		s.sendError(req.ID, -32000, err.Error())
		return
	}

	s.sendResponse(req.ID, map[string]string{
		"status":  "queued",
		"task_id": id,
		"message": "Download started successfully",
	})
}

func (s *MCPServer) handleList(req JsonRpcRequest) {
	tasks, err := s.engine.GetHistory()
	if err != nil {
		s.sendError(req.ID, -32000, err.Error())
		return
	}

	// Filter for simplified view
	var simplified []map[string]interface{}
	for _, t := range tasks {
		if t.Status == "downloading" || t.Status == "pending" || t.Status == "paused" {
			simplified = append(simplified, map[string]interface{}{
				"id":       t.ID,
				"filename": t.Filename,
				"status":   t.Status,
				"progress": t.Progress,
				"speed":    t.Speed,
				"eta":      t.TimeRemaining,
			})
		}
	}
	s.sendResponse(req.ID, simplified)
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

