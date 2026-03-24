package api

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"project-tachyon/internal/engine"
	"project-tachyon/internal/storage"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// newRequest creates an httptest request for API tests.
func newRequest(t *testing.T, method, path, body string) *http.Request {
	t.Helper()
	return httptest.NewRequest(method, path, strings.NewReader(body))
}

// serve runs a handler and returns the recorder.
func serve(h http.Handler, req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// --- MCP test helpers ---

// newTestMCPServer creates an MCPServer backed by an in-memory DB, writing to buf.
func newTestMCPServer(t *testing.T, buf *bytes.Buffer) *MCPServer {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	db.AutoMigrate(&storage.DownloadTask{}, &storage.DownloadLocation{}, &storage.DailyStat{}, &storage.AppSetting{})
	store := &storage.Storage{DB: db}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	eng := engine.NewEngine(logger, store)

	return NewMCPServerWithIO(eng, buf)
}

// sendRPC feeds a single JSON-RPC line into the MCPServer and returns the response.
func sendRPC(t *testing.T, srv *MCPServer, msg string) JsonRpcResponse {
	t.Helper()
	srv.handleMessage([]byte(msg))

	// Read what the server wrote
	var buf *bytes.Buffer
	if b, ok := srv.writer.(*bytes.Buffer); ok {
		buf = b
	} else {
		t.Fatal("writer is not a *bytes.Buffer")
	}

	var resp JsonRpcResponse
	scanner := bufio.NewScanner(buf)
	if scanner.Scan() {
		if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal response: %v\nraw: %s", err, scanner.Text())
		}
	} else {
		t.Fatal("no response written")
	}
	buf.Reset()
	return resp
}

// --- MCP lifecycle tests ---

func TestMCP_Initialize(t *testing.T) {
	var buf bytes.Buffer
	srv := newTestMCPServer(t, &buf)

	resp := sendRPC(t, srv, `{"jsonrpc":"2.0","method":"initialize","params":{},"id":1}`)
	if resp.Error != nil {
		t.Fatalf("initialize failed: %s", resp.Error.Message)
	}

	result := resp.Result.(map[string]interface{})
	if result["protocolVersion"] == nil {
		t.Error("missing protocolVersion")
	}
	caps := result["capabilities"].(map[string]interface{})
	if caps["tools"] == nil {
		t.Error("missing tools capability")
	}
	info := result["serverInfo"].(map[string]interface{})
	if info["name"] != "tachyon" {
		t.Errorf("expected server name 'tachyon', got %v", info["name"])
	}
}

func TestMCP_Initialized_NoResponse(t *testing.T) {
	var buf bytes.Buffer
	srv := newTestMCPServer(t, &buf)

	srv.handleMessage([]byte(`{"jsonrpc":"2.0","method":"notifications/initialized"}`))
	if buf.Len() != 0 {
		t.Error("notifications/initialized should not produce a response")
	}
}

// --- handleMessage routing tests ---

func TestMCP_HandleMessage_InvalidJSON(t *testing.T) {
	var buf bytes.Buffer
	srv := newTestMCPServer(t, &buf)

	resp := sendRPC(t, srv, "not valid json{{{")
	if resp.Error == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if resp.Error.Code != -32700 {
		t.Errorf("expected parse error code -32700, got %d", resp.Error.Code)
	}
}

func TestMCP_HandleMessage_UnknownMethod(t *testing.T) {
	var buf bytes.Buffer
	srv := newTestMCPServer(t, &buf)

	resp := sendRPC(t, srv, `{"jsonrpc":"2.0","method":"unknown_method","id":1}`)
	if resp.Error == nil {
		t.Fatal("expected error for unknown method")
	}
	if resp.Error.Code != -32601 {
		t.Errorf("expected method-not-found code -32601, got %d", resp.Error.Code)
	}
	if resp.ID != float64(1) {
		t.Errorf("response ID should echo request ID")
	}
}

func TestMCP_HandleMessage_EmptyLine(t *testing.T) {
	var buf bytes.Buffer
	srv := newTestMCPServer(t, &buf)

	input := strings.NewReader("\n\n\n")
	srv.StartWithReader(input)

	if buf.Len() != 0 {
		t.Errorf("empty lines should produce no output, got %q", buf.String())
	}
}

// --- tools/list ---

func TestMCP_ToolsList(t *testing.T) {
	var buf bytes.Buffer
	srv := newTestMCPServer(t, &buf)

	resp := sendRPC(t, srv, `{"jsonrpc":"2.0","method":"tools/list","id":10}`)
	if resp.Error != nil {
		t.Fatalf("tools/list returned error: %s", resp.Error.Message)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatal("result should be a map")
	}
	tools, ok := result["tools"].([]interface{})
	if !ok {
		t.Fatal("tools key should be an array")
	}
	if len(tools) < 2 {
		t.Errorf("expected at least 2 tools, got %d", len(tools))
	}

	// Verify tool names
	names := make(map[string]bool)
	for _, tool := range tools {
		tm := tool.(map[string]interface{})
		names[tm["name"].(string)] = true
	}
	if !names["tachyon_download"] {
		t.Error("missing tachyon_download tool")
	}
	if !names["tachyon_list"] {
		t.Error("missing tachyon_list tool")
	}
}

func TestMCP_ToolsList_HasInputSchema(t *testing.T) {
	var buf bytes.Buffer
	srv := newTestMCPServer(t, &buf)

	resp := sendRPC(t, srv, `{"jsonrpc":"2.0","method":"tools/list","id":1}`)
	result := resp.Result.(map[string]interface{})
	tools := result["tools"].([]interface{})

	for _, tool := range tools {
		tm := tool.(map[string]interface{})
		schema, ok := tm["inputSchema"]
		if !ok || schema == nil {
			t.Errorf("tool %s missing inputSchema", tm["name"])
		}
	}
}

// --- tools/call: tachyon_download ---

// toolCall is a helper to build a tools/call JSON-RPC message.
func toolCall(id int, name string, args string) string {
	return fmt.Sprintf(`{"jsonrpc":"2.0","method":"tools/call","params":{"name":%q,"arguments":%s},"id":%d}`, name, args, id)
}

func TestMCP_Download_MissingURL(t *testing.T) {
	var buf bytes.Buffer
	srv := newTestMCPServer(t, &buf)

	resp := sendRPC(t, srv, toolCall(2, "tachyon_download", `{"url":""}`))
	result := resp.Result.(map[string]interface{})
	if result["isError"] != true {
		t.Fatal("expected isError for empty URL")
	}
}

func TestMCP_Download_InvalidToolCallParams(t *testing.T) {
	var buf bytes.Buffer
	srv := newTestMCPServer(t, &buf)

	resp := sendRPC(t, srv, `{"jsonrpc":"2.0","method":"tools/call","params":"bad","id":3}`)
	if resp.Error == nil {
		t.Fatal("expected error for non-object params")
	}
}

func TestMCP_Download_InvalidURLScheme(t *testing.T) {
	var buf bytes.Buffer
	srv := newTestMCPServer(t, &buf)

	resp := sendRPC(t, srv, toolCall(4, "tachyon_download", `{"url":"ftp://bad.com/file"}`))
	result := resp.Result.(map[string]interface{})
	if result["isError"] != true {
		t.Fatal("expected isError for ftp scheme")
	}
}

func TestMCP_Download_ValidURL(t *testing.T) {
	var buf bytes.Buffer
	srv := newTestMCPServer(t, &buf)

	resp := sendRPC(t, srv, toolCall(5, "tachyon_download", `{"url":"https://example.com/file.zip","path":".","filename":"test.zip"}`))
	if resp.Error != nil {
		t.Fatalf("unexpected error: %s", resp.Error.Message)
	}

	result := resp.Result.(map[string]interface{})
	if result["isError"] == true {
		t.Error("expected success")
	}
	content := result["content"].([]interface{})
	text := content[0].(map[string]interface{})["text"].(string)
	if !strings.Contains(text, "queued") {
		t.Errorf("expected 'queued' in result text, got: %s", text)
	}
}

func TestMCP_Download_PathTraversalFilename(t *testing.T) {
	var buf bytes.Buffer
	srv := newTestMCPServer(t, &buf)

	resp := sendRPC(t, srv, toolCall(6, "tachyon_download", `{"url":"https://example.com/file.zip","filename":"../../etc/passwd"}`))
	result := resp.Result.(map[string]interface{})
	if result["isError"] == true {
		t.Fatal("sanitized filename should succeed")
	}
}

func TestMCP_Download_LoopbackBlocked(t *testing.T) {
	var buf bytes.Buffer
	srv := newTestMCPServer(t, &buf)

	resp := sendRPC(t, srv, toolCall(7, "tachyon_download", `{"url":"http://127.0.0.1/secret"}`))
	result := resp.Result.(map[string]interface{})
	if result["isError"] != true {
		t.Fatal("expected isError for loopback URL")
	}
}

func TestMCP_Download_UnknownTool(t *testing.T) {
	var buf bytes.Buffer
	srv := newTestMCPServer(t, &buf)

	resp := sendRPC(t, srv, toolCall(8, "nonexistent_tool", `{}`))
	if resp.Error == nil {
		t.Fatal("expected error for unknown tool")
	}
}

// --- tools/call: tachyon_list ---

func TestMCP_List_EmptyHistory(t *testing.T) {
	var buf bytes.Buffer
	srv := newTestMCPServer(t, &buf)

	resp := sendRPC(t, srv, toolCall(9, "tachyon_list", `{}`))
	if resp.Error != nil {
		t.Fatalf("unexpected error: %s", resp.Error.Message)
	}
	result := resp.Result.(map[string]interface{})
	content := result["content"].([]interface{})
	text := content[0].(map[string]interface{})["text"].(string)
	if !strings.Contains(text, "No active") {
		t.Errorf("expected 'No active' for empty list, got: %s", text)
	}
}

func TestMCP_List_AfterDownload(t *testing.T) {
	var buf bytes.Buffer
	srv := newTestMCPServer(t, &buf)

	// Queue a download first
	sendRPC(t, srv, toolCall(100, "tachyon_download", `{"url":"https://example.com/file.zip","path":"."}`))

	// Now list
	resp := sendRPC(t, srv, toolCall(101, "tachyon_list", `{}`))
	if resp.Error != nil {
		t.Fatalf("list error: %s", resp.Error.Message)
	}
}

// --- StartWithReader integration ---

func TestMCP_StartWithReader_MultipleMessages(t *testing.T) {
	var buf bytes.Buffer
	srv := newTestMCPServer(t, &buf)

	// Send three JSON-RPC messages: initialize + tools/list + unknown
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","method":"initialize","params":{},"id":1}`,
		`{"jsonrpc":"2.0","method":"tools/list","id":2}`,
		`{"jsonrpc":"2.0","method":"unknown","id":3}`,
	}, "\n")

	srv.StartWithReader(strings.NewReader(input))

	// Should have 3 lines of output
	scanner := bufio.NewScanner(&buf)
	count := 0
	for scanner.Scan() {
		count++
		var resp JsonRpcResponse
		if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
			t.Errorf("line %d: invalid JSON response: %v", count, err)
		}
	}
	if count != 3 {
		t.Errorf("expected 3 responses, got %d", count)
	}
}

func TestMCP_StartWithReader_SkipsEmptyLines(t *testing.T) {
	var buf bytes.Buffer
	srv := newTestMCPServer(t, &buf)

	input := "\n\n" + `{"jsonrpc":"2.0","method":"tools/list","id":1}` + "\n\n"
	srv.StartWithReader(strings.NewReader(input))

	scanner := bufio.NewScanner(&buf)
	count := 0
	for scanner.Scan() {
		count++
	}
	if count != 1 {
		t.Errorf("expected 1 response (empty lines skipped), got %d", count)
	}
}

// --- Response format verification ---

func TestMCP_ResponseFormat_JSONRPC(t *testing.T) {
	var buf bytes.Buffer
	srv := newTestMCPServer(t, &buf)

	resp := sendRPC(t, srv, `{"jsonrpc":"2.0","method":"tools/list","id":"str-id"}`)
	if resp.JSONRPC != "2.0" {
		t.Errorf("jsonrpc field = %q, want 2.0", resp.JSONRPC)
	}
	if resp.ID != "str-id" {
		t.Errorf("response ID should echo request ID, got %v", resp.ID)
	}
}

func TestMCP_ResponseFormat_ErrorPreservesID(t *testing.T) {
	var buf bytes.Buffer
	srv := newTestMCPServer(t, &buf)

	resp := sendRPC(t, srv, `{"jsonrpc":"2.0","method":"nonexistent","id":42}`)
	if resp.ID != float64(42) {
		t.Errorf("error response should preserve ID, got %v", resp.ID)
	}
	if resp.Error == nil {
		t.Fatal("expected error")
	}
	if resp.Result != nil {
		t.Error("error response should not have result")
	}
}

func TestParseCookieString_Whitespace(t *testing.T) {
	cookies := ParseCookieString("  a=1 ;  b=2  ")
	if len(cookies) != 2 {
		t.Fatalf("expected 2 cookies, got %d", len(cookies))
	}
}

func TestParseCookieString_ValueWithEquals(t *testing.T) {
	cookies := ParseCookieString("token=abc=def==")
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}
	if cookies[0].Value != "abc=def==" {
		t.Errorf("expected value with = signs preserved, got %q", cookies[0].Value)
	}
}

func TestParseCookieString_NoValue(t *testing.T) {
	cookies := ParseCookieString("orphan")
	// Should handle gracefully (either skip or set empty value)
	t.Logf("parsed %d cookies from bare key", len(cookies))
}

func TestParseCookieString_ManyPairs(t *testing.T) {
	raw := "a=1; b=2; c=3; d=4; e=5; f=6; g=7; h=8; i=9; j=10"
	cookies := ParseCookieString(raw)
	if len(cookies) != 10 {
		t.Errorf("expected 10 cookies, got %d", len(cookies))
	}
}
