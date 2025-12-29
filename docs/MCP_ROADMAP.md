# MCP Server Roadmap

Extracted from vibe-kanban session (2025-12-28). These represent planned improvements to the MCP server based on research into best practices.

---

## [P1] MCP Security Hardening

**Priority:** 1 (Highest)
**Effort:** Medium (2-3 days)

### Summary
Implement security best practices for the MCP server to prevent injection attacks, path traversal, and abuse.

### Tasks

#### 1.1 Input Validation with go-playground/validator
**Files:** `mcp/server.go`

Add struct-based validation for all tool inputs:

```go
import "github.com/go-playground/validator/v10"

type ExportDocInput struct {
    DocID    string `validate:"required,alphanum,min=12"`
    Format   string `validate:"required,oneof=excel grist"`
    Filename string `validate:"omitempty,max=255"`
}
```

- [ ] Add `github.com/go-playground/validator/v10` dependency
- [ ] Create input structs for all 6 tools
- [ ] Validate inputs before processing
- [ ] Return clear error messages for validation failures

#### 1.2 Path Traversal Prevention in export_doc
**Files:** `mcp/server.go` (lines 244-256)

```go
func SanitizeFilename(filename string) string {
    filename = strings.ReplaceAll(filename, "..", "_")
    filename = strings.ReplaceAll(filename, "/", "_")
    filename = strings.ReplaceAll(filename, "\\", "_")
    safePattern := regexp.MustCompile(`[^a-zA-Z0-9_\-\.]`)
    return safePattern.ReplaceAllString(filename, "_")
}
```

- [ ] Create `SanitizeFilename()` function
- [ ] Apply to export_doc handler
- [ ] Add tests for path traversal attempts (`../../../etc/passwd`)
- [ ] Reject absolute paths

#### 1.3 Rate Limiting
**Files:** `mcp/server.go` (new middleware)

```go
import "golang.org/x/time/rate"

var (
    globalLimiter = rate.NewLimiter(100, 200)  // 100 req/sec, burst 200
    toolLimits = map[string]*rate.Limiter{
        "export_doc": rate.NewLimiter(0.1, 2),  // 6/min, burst 2
        "list_orgs":  rate.NewLimiter(1, 10),   // 1/sec, burst 10
    }
)
```

- [ ] Add `golang.org/x/time/rate` dependency
- [ ] Implement global rate limiter
- [ ] Implement per-tool rate limiters
- [ ] Return proper MCP error on rate limit exceeded

#### 1.4 Secret Redaction in Logs
**Files:** `gristapi/gristapi.go`

```go
var sensitivePatterns = []*regexp.Regexp{
    regexp.MustCompile(`(?i)bearer\s+[a-zA-Z0-9_\-\.]+`),
    regexp.MustCompile(`(?i)(token|key|secret)\s*[:=]\s*['"]?[^\s'"]+`),
}

func RedactSecrets(input string) string {
    for _, pattern := range sensitivePatterns {
        input = pattern.ReplaceAllString(input, "[REDACTED]")
    }
    return input
}
```

- [ ] Create `RedactSecrets()` function
- [ ] Wrap all log/error output through redaction
- [ ] Never include GRIST_TOKEN in error messages

### Dependencies
- `github.com/go-playground/validator/v10`
- `golang.org/x/time/rate`

### References
- [OWASP MCP Top 10](https://owasp.org/www-project-mcp-top-10/)
- [MCP Security Best Practices](https://modelcontextprotocol.io/specification/2025-06-18/basic/security_best_practices)

---

## [P2] MCP Performance Optimization

**Priority:** 2
**Effort:** Small-Medium (1-2 days)

### Summary
Optimize network I/O, JSON encoding, and connection management for better MCP server performance.

### Tasks

#### 2.1 HTTP Client Reuse
**Files:** `gristapi/gristapi.go` (line 135)

Current implementation creates a new `http.Client` for every request:

```go
// RECOMMENDED
var (
    httpClient *http.Client
    clientOnce sync.Once
)

func getHTTPClient() *http.Client {
    clientOnce.Do(func() {
        httpClient = &http.Client{
            Transport: &http.Transport{
                MaxIdleConnsPerHost: 10,
                MaxIdleConns:        100,
                IdleConnTimeout:     90 * time.Second,
                TLSHandshakeTimeout: 10 * time.Second,
            },
            Timeout: 30 * time.Second,
        }
    })
    return httpClient
}
```

- [ ] Create singleton HTTP client with proper transport config
- [ ] Replace all `&http.Client{}` with `getHTTPClient()`
- [ ] Configure connection pooling (10 per host, 100 total)
- [ ] Benchmark before/after

#### 2.2 JSON Encoding Optimization
**Files:** `mcp/server.go`

Replace `json.MarshalIndent` with `json.Marshal` (or jsoniter for 2-3x speedup):

```go
import jsoniter "github.com/json-iterator/go"
var json = jsoniter.ConfigCompatibleWithStandardLibrary
```

- [ ] Replace `MarshalIndent` with `Marshal`
- [ ] Evaluate jsoniter vs standard library
- [ ] Benchmark encoding performance

#### 2.3 sync.Pool for Buffer Reuse
**Files:** `mcp/server.go`, `gristapi/gristapi.go`

```go
var bufferPool = sync.Pool{
    New: func() interface{} {
        return new(bytes.Buffer)
    },
}
```

- [ ] Implement buffer pool for JSON encoding
- [ ] Benchmark memory allocations before/after
- [ ] Verify no data races with `-race` flag

### Expected Impact
- HTTP client reuse: 10-50ms saved per request
- JSON optimization: 2-3x encoding speedup
- Buffer pooling: Reduced GC pressure

---

## [P3] MCP Reliability & Observability

**Priority:** 3
**Effort:** Medium (2-3 days)

### Summary
Improve server stability with panic recovery, context cancellation, progress reporting, and lifecycle hooks.

### Tasks

#### 3.1 Panic Recovery Middleware
**Files:** `mcp/server.go`

```go
func NewServer() *server.MCPServer {
    s := server.NewMCPServer("gristle", "1.0.0",
        server.WithToolCapabilities(true),
        server.WithRecovery(),  // ADD THIS
    )
}
```

- [ ] Add `server.WithRecovery()` option
- [ ] Verify panics are caught and logged
- [ ] Add test that triggers panic and verifies recovery

#### 3.2 Context Cancellation in Tool Handlers
**Files:** `mcp/server.go` (all handlers)

```go
func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    select {
    case <-ctx.Done():
        return mcp.NewToolResultError("operation cancelled"), nil
    default:
    }
    // ... handler logic
}
```

- [ ] Add context cancellation checks to all handlers
- [ ] Check before network calls
- [ ] Check in loops (get_doc_tables iterates tables)

#### 3.3 Progress Reporting for Long Operations
**Files:** `mcp/server.go` (export_doc, get_doc_tables)

```go
progressToken := req.GetProgressToken()
if progressToken != nil {
    notifyProgress(ctx, progressToken, 0, 100, "Starting export...")
}
```

- [ ] Add progress reporting to export_doc
- [ ] Add progress reporting to get_doc_tables
- [ ] Rate-limit progress notifications

#### 3.4 Lifecycle Hooks for Observability
**Files:** `mcp/server.go`

```go
hooks := &server.Hooks{
    OnRegisterSession: func(ctx context.Context, session *server.Session) {
        log.Printf("Session connected: %s", session.ID())
    },
    BeforeAny: func(ctx context.Context, id any, method mcp.MCPMethod, msg any) {
        log.Printf("Request: [%s] %v", method, id)
    },
}
```

- [ ] Add session lifecycle hooks
- [ ] Add request/response hooks
- [ ] Log with structured format

#### 3.5 Graceful Shutdown
**Files:** `mcp/server.go`, `main.go`

- [ ] Handle SIGINT/SIGTERM gracefully
- [ ] Allow in-flight requests to complete
- [ ] Set shutdown timeout (30s)

---

## [P4] MCP Testing Infrastructure

**Priority:** 4
**Effort:** Medium-Large (3-5 days)

### Summary
Establish comprehensive testing including unit tests, integration tests, fuzz tests, and benchmarks.

### Current State
- `mcp/server.go`: 0 tests
- No integration tests for MCP protocol
- No fuzz tests for input validation
- No benchmarks

### Tasks

#### 4.1 Unit Tests for Tool Handlers
**Files:** `mcp/server_test.go` (new)

```go
type MockGristAPI struct {
    OrgsFunc func() []gristapi.Org
}

func TestListOrgsHandler(t *testing.T) {
    tests := []struct {
        name     string
        mockOrgs []gristapi.Org
        wantLen  int
    }{
        {"empty", []gristapi.Org{}, 0},
        {"multiple", []gristapi.Org{{Id: 1}, {Id: 2}}, 2},
    }
    // ...
}
```

- [ ] Create GristAPI interface for mocking
- [ ] Write table-driven tests for all 6 tools
- [ ] Achieve >80% coverage on handler logic

#### 4.2 Integration Tests with mcptest
**Files:** `mcp/integration_test.go` (new)

```go
func TestMCPServerIntegration(t *testing.T) {
    s := NewServer()
    testServer, _ := mcptest.NewServerFromMCPServer(t, s)
    client := testServer.Client()

    tools, _ := client.ListTools(ctx)
    // verify tools...
}
```

- [ ] Set up mcptest infrastructure
- [ ] Test tool listing and each tool call
- [ ] Test error responses for invalid inputs

#### 4.3 Fuzz Tests for Input Validation
**Files:** `mcp/fuzz_test.go` (new)

```go
func FuzzExportDocParams(f *testing.F) {
    f.Add([]byte(`{"doc_id": "../../../etc/passwd"}`))
    f.Fuzz(func(t *testing.T, data []byte) {
        // Handler should never panic
    })
}
```

- [ ] Add fuzz tests for all tool inputs
- [ ] Add fuzz test for filename sanitization
- [ ] Run fuzzing for significant duration

#### 4.4 Benchmark Tests
**Files:** `mcp/bench_test.go` (new)

- [ ] Add benchmarks for all handlers
- [ ] Track allocations with `b.ReportAllocs()`
- [ ] Add `benchstat` comparison to CI

---

## [P5] MCP Code Quality & Architecture

**Priority:** 5
**Effort:** Medium (2-3 days)

### Summary
Improve code organization, extract reusable patterns, and enhance maintainability.

### Tasks

#### 5.1 Extract Handler Logic from Inline Functions
**Files:** `mcp/server.go` → `mcp/handlers.go` (new)

```go
type Handlers struct {
    api GristAPI
}

func (h *Handlers) ListOrgs(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    orgs := h.api.GetOrgs()
    return encodeResult(orgs)
}
```

- [ ] Create `handlers.go` with extracted handler methods
- [ ] Create `Handlers` struct with API dependency

#### 5.2 Create GristAPI Interface
**Files:** `gristapi/interface.go` (new)

```go
type API interface {
    GetOrgs() []Org
    GetOrgWorkspaces(orgID int) []Workspace
    GetDoc(docID string) Doc
    // ...
}
```

- [ ] Define `API` interface with all methods
- [ ] Enable mock implementations for testing

#### 5.3 Structured Logging
**Files:** `mcp/logging.go` (new)

```go
import "log/slog"

logger.Info("listing organizations",
    slog.String("tool", "list_orgs"),
    slog.String("request_id", getRequestID(ctx)),
)
```

- [ ] Add slog-based structured logging
- [ ] Support JSON and text formats via env var

#### 5.4 Read-Only Mode
**Files:** `mcp/server.go`

```go
if !cfg.ReadOnly {
    s.AddTool(exportDocTool, h.ExportDoc)
}
```

- [ ] Add `MCP_READ_ONLY` environment variable
- [ ] Add `--read-only` CLI flag

#### 5.5 Tool Definitions as Constants
**Files:** `mcp/tools.go` (new)

- [ ] Extract tool definitions to separate file
- [ ] Use `mcp.Enum()` for format validation

#### 5.6 Configuration Management
**Files:** `mcp/config.go` (new)

- [ ] Create centralized config struct
- [ ] Load from environment variables

### Target File Structure

```
mcp/
├── server.go           # Server creation and registration
├── handlers.go         # Handler implementations
├── tools.go            # Tool definitions
├── config.go           # Configuration management
├── logging.go          # Structured logging
├── middleware.go       # Rate limiting, validation
├── server_test.go      # Unit tests
├── integration_test.go
├── fuzz_test.go
└── bench_test.go
```

---

## References

- [OWASP MCP Top 10](https://owasp.org/www-project-mcp-top-10/)
- [MCP Security Best Practices](https://modelcontextprotocol.io/specification/2025-06-18/basic/security_best_practices)
- [mcp-go Documentation](https://pkg.go.dev/github.com/mark3labs/mcp-go)
- [Go Fuzz Testing](https://go.dev/doc/security/fuzz/)
