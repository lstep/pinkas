package mcp

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mostdoc/mostdoc/internal/db"
	"github.com/mostdoc/mostdoc/internal/directories"
	"github.com/mostdoc/mostdoc/internal/mcptokens"
	"github.com/mostdoc/mostdoc/internal/pages"
	"github.com/mostdoc/mostdoc/internal/permissions"
	"github.com/mostdoc/mostdoc/internal/spaces"
)

// testEnv holds the test environment for MCP tests.
type testEnv struct {
	Srv    *Server
	Repos  *testRepos
	Logger *slog.Logger
	Token  *mcptokens.MCPToken
}

type testRepos struct {
	Pages   *pages.Repository
	Spaces  *spaces.Repository
	Dirs    *directories.Repository
	Perm    *permissions.Repository
	MCPTok  *mcptokens.Repository
	MCPTokSvc *mcptokens.Service
	PermRes *permissions.Resolver
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	conn, err := db.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	if err := db.Migrate(conn, "file://../../migrations"); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	spacesRepo := spaces.NewRepository(conn)
	dirsRepo := directories.NewRepository(conn)
	pagesRepo := pages.NewRepository(conn)
	permRepo := permissions.NewRepository(conn)
	mcpTokenRepo := mcptokens.NewRepository(conn)
	mcpTokenSvc := mcptokens.NewService(mcpTokenRepo)

	err = spacesRepo.Create(context.Background(), "space-1", "Test Space", "test-space",
		"editor", true, nil)
	if err != nil {
		t.Fatalf("create space: %v", err)
	}

	err = pagesRepo.CreatePage(context.Background(), "page-1", "space-1", "Test Page", "test-page",
		0, nil, "user-1", "📄")
	if err != nil {
		t.Fatalf("create page: %v", err)
	}

	permRes := permissions.NewResolver(
		permRepo,
		dirsRepo.GetDirectory,
		spacesRepo.Get,
		pagesRepo.GetPage,
		nil,
		logger,
	)

	mcpSrv := NewServer(mcpTokenSvc, pagesRepo, spacesRepo, dirsRepo, permRes, logger)

	return &testEnv{
		Srv:    mcpSrv,
		Repos: &testRepos{
			Pages:    pagesRepo,
			Spaces:   spacesRepo,
			Dirs:     dirsRepo,
			Perm:     permRepo,
			MCPTok:   mcpTokenRepo,
			MCPTokSvc: mcpTokenSvc,
			PermRes:  permRes,
		},
		Logger: logger,
	}
}

// createTestSession initializes the MCP protocol handshake and returns a context
// with a registered session and auth info (userID=user-1, write+read scopes).
func (e *testEnv) createTestSession(t *testing.T) context.Context {
	t.Helper()

	// Create token
	tokenResp, err := e.Repos.MCPTokSvc.CreateToken(context.Background(), "user-1", mcptokens.CreateTokenRequest{
		Name:   "Test Token",
		Scopes: []string{mcptokens.ScopeRead, mcptokens.ScopeWrite},
	})
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	// Validate to get the actual token object with correct state
	token, err := e.Repos.MCPTokSvc.ValidateToken(context.Background(), tokenResp.Secret)
	if err != nil {
		t.Fatalf("validate token: %v", err)
	}
	e.Token = token

	// Create an in-process session
	session := server.NewInProcessSession("test-session-1", nil)
	session.SetClientCapabilities(mcp.ClientCapabilities{})
	session.Initialize()

	if err := e.Srv.mcpServer.RegisterSession(context.Background(), session); err != nil {
		t.Fatalf("register session: %v", err)
	}

	// Build context with auth info (simulating what authMiddleware does)
	ctx := e.Srv.mcpServer.WithContext(context.Background(), session)
	ctx = context.WithValue(ctx, ctxKeyUserID, token.UserID)
	ctx = context.WithValue(ctx, ctxKeyToken, token)
	ctx = context.WithValue(ctx, ctxKeyIsAdmin, hasScope(token, mcptokens.ScopeAdmin))

	return ctx
}

// callTool sends a JSON-RPC tools/call message via HandleMessage and returns the result.
func (e *testEnv) callTool(ctx context.Context, t *testing.T, toolName string, args map[string]interface{}) (map[string]interface{}, error) {
	t.Helper()

	// Build JSON-RPC request
	params := map[string]interface{}{
		"name":      toolName,
		"arguments": args,
	}
	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params":  params,
	}
	body, _ := json.Marshal(request)

	// Send via HandleMessage
	response := e.Srv.mcpServer.HandleMessage(ctx, body)

	// Validate it's a JSONRPCResponse
	if resp, ok := response.(mcp.JSONRPCResponse); ok {
		// Marshal the result back
		resultBytes, _ := json.Marshal(resp.Result)
		var result map[string]interface{}
		if err := json.Unmarshal(resultBytes, &result); err != nil {
			return nil, err
		}
		return result, nil
	}

	// Check for error response
	if errResp, ok := response.(mcp.JSONRPCError); ok {
		return nil, &mcpCallError{Code: errResp.Error.Code, Message: errResp.Error.Message}
	}

	return nil, nil
}

type mcpCallError struct {
	Code    int
	Message string
}

func (e *mcpCallError) Error() string {
	return e.Message
}

// ─── Tests ───────────────────────────────────────────────────────────────────

func TestWikiListSpaces(t *testing.T) {
	env := newTestEnv(t)
	ctx := env.createTestSession(t)

	result, err := env.callTool(ctx, t, "wiki_list_spaces", nil)
	if err != nil {
		t.Fatalf("wiki_list_spaces: %v", err)
	}

	content := parseContent(t, result)
	if !strings.Contains(content, "Test Space") {
		t.Errorf("expected 'Test Space' in response, got: %s", content)
	}
}

func TestWikiGetSpaceBySlug(t *testing.T) {
	env := newTestEnv(t)
	ctx := env.createTestSession(t)

	result, err := env.callTool(ctx, t, "wiki_get_space_by_slug", map[string]interface{}{
		"slug": "test-space",
	})
	if err != nil {
		t.Fatalf("wiki_get_space_by_slug: %v", err)
	}

	content := parseContent(t, result)
	if !strings.Contains(content, "test-space") {
		t.Errorf("expected slug 'test-space' in response, got: %s", content)
	}
}

func TestWikiGetPageByID(t *testing.T) {
	env := newTestEnv(t)
	ctx := env.createTestSession(t)

	result, err := env.callTool(ctx, t, "wiki_get_page", map[string]interface{}{
		"identifier": "page-1",
	})
	if err != nil {
		t.Fatalf("wiki_get_page: %v", err)
	}

	content := parseContent(t, result)
	if !strings.Contains(content, "Test Page") {
		t.Errorf("expected 'Test Page' in response, got: %s", content)
	}
}

func TestWikiCreatePage(t *testing.T) {
	env := newTestEnv(t)
	ctx := env.createTestSession(t)

	result, err := env.callTool(ctx, t, "wiki_create_page", map[string]interface{}{
		"space_id": "space-1",
		"title":    "Created by MCP",
		"content":  "## Hello from MCP\n\nThis page was created via the MCP API.",
	})
	if err != nil {
		t.Fatalf("wiki_create_page: %v", err)
	}

	content := parseContent(t, result)
	if !strings.Contains(content, "created-by-mcp") {
		t.Errorf("expected slug 'created-by-mcp' in response, got: %s", content)
	}

	// Verify the page was actually created in the DB
	page, err := env.Repos.Pages.GetPageBySlug(context.Background(), "space-1", "created-by-mcp")
	if err != nil {
		t.Fatalf("get created page: %v", err)
	}
	if page.Title.String != "Created by MCP" {
		t.Errorf("expected title 'Created by MCP', got: %s", page.Title.String)
	}
}

func TestWikiCreatePageNoAuth(t *testing.T) {
	env := newTestEnv(t)

	// Create a context with NO auth info (simulating missing auth)
	session := server.NewInProcessSession("test-session-noauth", nil)
	session.SetClientCapabilities(mcp.ClientCapabilities{})
	session.Initialize()
	if err := env.Srv.mcpServer.RegisterSession(context.Background(), session); err != nil {
		t.Fatalf("register session: %v", err)
	}
	ctx := env.Srv.mcpServer.WithContext(context.Background(), session)

	result, err := env.callTool(ctx, t, "wiki_create_page", map[string]interface{}{
		"space_id": "space-1",
		"title":    "No Auth Test",
	})
	if err == nil {
		// If no error, check if the result contains an error text
		content := parseContent(t, result)
		if !strings.Contains(content, "Authentication required") {
			t.Fatalf("expected authentication error, got: %s", content)
		}
	}
}

func TestWikiUpdatePage(t *testing.T) {
	env := newTestEnv(t)
	ctx := env.createTestSession(t)

	result, err := env.callTool(ctx, t, "wiki_update_page", map[string]interface{}{
		"page_id": "page-1",
		"title":   "Updated Title",
		"content": "Updated content from MCP",
	})
	if err != nil {
		t.Fatalf("wiki_update_page: %v", err)
	}

	_ = result // wiki_update_page returns text, not JSON

	// Verify the page was updated
	getResult, err := env.callTool(ctx, t, "wiki_get_page", map[string]interface{}{
		"identifier": "page-1",
	})
	if err != nil {
		t.Fatalf("wiki_get_page after update: %v", err)
	}

	content := parseContent(t, getResult)
	if !strings.Contains(content, "Updated Title") {
		t.Errorf("expected 'Updated Title' in get response, got: %s", content)
	}
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

// parseContent extracts the text content from a CallToolResult.
func parseContent(t *testing.T, result map[string]interface{}) string {
	t.Helper()

	contentRaw, ok := result["content"]
	if !ok {
		t.Fatal("response has no 'content' field")
	}

	contentList, ok := contentRaw.([]interface{})
	if !ok || len(contentList) == 0 {
		t.Fatal("response content is not a non-empty array")
	}

	firstItem, ok := contentList[0].(map[string]interface{})
	if !ok {
		t.Fatal("content[0] is not an object")
	}

	text, ok := firstItem["text"].(string)
	if !ok {
		t.Fatal("content[0].text is not a string")
	}

	return text
}

// ─── Pure unit tests ────────────────────────────────────────────────────────

func TestSlugify(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello World", "hello-world"},
		{"  My Page  ", "my-page"},
		{"Special!@#Chars", "special-chars"},
		{"Already-a-slug", "already-a-slug"},
		{"UPPERCASE", "uppercase"},
		{"one", "one"},
		{"", ""},
	}

	for _, tt := range tests {
		result := slugify(tt.input)
		if result != tt.expected {
			t.Errorf("slugify(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestSplitIdentifier(t *testing.T) {
	tests := []struct {
		input string
		parts int
	}{
		{"space-slug/page-slug", 2},
		{"just-uuid", 1},
		{"a/b/c", 3},
		{"", 0},
	}

	for _, tt := range tests {
		result := splitIdentifier(tt.input)
		if len(result) != tt.parts {
			t.Errorf("splitIdentifier(%q) = %v (len=%d), want len=%d", tt.input, result, len(result), tt.parts)
		}
	}
}

func TestTruncate(t *testing.T) {
	if truncate("short", 10) != "short" {
		t.Errorf("expected 'short', got '%s'", truncate("short", 10))
	}
	result := truncate("this is a long string", 10)
	if len(result) != 13 {
		t.Errorf("expected length 13, got %d", len(result))
	}
	if result != "this is a ..." {
		t.Errorf("expected 'this is a ...', got '%s'", result)
	}
}
