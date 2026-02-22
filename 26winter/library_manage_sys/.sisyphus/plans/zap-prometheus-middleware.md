# Zap & Prometheus Logging Middleware for Gin

## TL;DR

> **Quick Summary**: Create a Gin middleware that combines structured logging with Zap and Prometheus metrics monitoring.
>
> **Deliverables**:
> - Add Zap and Prometheus dependencies to go.mod
> - Create `backend/middleware/prometheus.go` with metric definitions (CounterVec, HistogramVec)
> - Create `backend/middleware/logger.go` with InitLogger() factory function
> - Create `backend/middleware/logging_and_metrics.go` with GinLoggerAndMetrics() middleware
> - Update `backend/main.go` to initialize logger, register middleware, and add /metrics endpoint
> - Add /ping test route for verification
> - Full TDD test suite for the middleware (RED-GREEN-REFACTOR cycle)
>
> **Estimated Effort**: Medium
> **Parallel Execution**: YES - 2 waves (foundation + integration + tests)
> **Critical Path**: Dependencies → Prometheus Metrics → Logger → Middleware → Main.go Integration → Tests

---

## Context

### Original Request
Create a Gin middleware integrating structured logging (Zap) and Prometheus metrics monitoring. Must include CounterVec for request counting, HistogramVec for latency tracking, log level logic based on status codes, and metrics endpoint.

### Interview Summary
**Key Discussions**:
- **Package Structure**: Add to existing `backend/middleware` (not `internal/middleware`)
- **Path Label Strategy**: Normalize to route patterns (e.g., `/books/:id` instead of `/books/123`)
- **Testing Approach**: TDD (RED-GREEN-REFACTOR cycle)
- **Logger Configuration**: Always use production config (`zap.NewProduction()`)

**Decisions Made**:
- Keep existing project structure (`backend/middleware/` package)
- Use Gin's `c.FullPath()` for route pattern extraction (built-in normalization)
- Use Prometheus default histogram buckets
- Prefix metric names with "library_" to avoid conflicts
- Follow existing Chinese log message pattern for consistency

### Metis Review
**Identified Gaps** (addressed):
- **Duplicate middleware problem**: Lines 66-67 in main.go have duplicate gin.Logger()/gin.Recovery() calls. Decision: Document but do NOT modify unless explicitly requested.
- **TDD in testless project**: Project has NO existing tests. Added explicit test infrastructure setup tasks.
- **Language consistency**: Existing codebase uses Chinese logs. Decision: Match this pattern in new middleware.
- **Path normalization**: Gin's `c.FullPath()` already returns route patterns. Decision: Use it directly.
- **Histogram buckets**: Default vs custom. Decision: Use Prometheus default buckets.
- **Metric naming**: Generic vs app-prefixed. Decision: Use "library_" prefix to prevent conflicts.
- **Guardrails added**: Do NOT modify existing middleware, do NOT change config beyond specified, do NOT add extra metrics.

---

## Work Objectives

### Core Objective
Implement a production-ready Gin middleware that combines structured logging with Zap and Prometheus metrics, following Go best practices and matching existing codebase patterns.

### Concrete Deliverables
- `backend/middleware/prometheus.go` - Prometheus metrics definitions (CounterVec, HistogramVec)
- `backend/middleware/logger.go` - Zap logger factory function
- `backend/middleware/logging_and_metrics.go` - Main middleware implementation
- `backend/middleware/prometheus_test.go` - TDD tests for metrics (RED-GREEN-REFACTOR)
- `backend/middleware/logging_and_metrics_test.go` - TDD tests for middleware (RED-GREEN-REFACTOR)
- Updated `backend/main.go` - Logger initialization, middleware registration, /metrics endpoint, /ping route
- Updated `backend/go.mod` - Added Zap and Prometheus dependencies

### Definition of Done
- [ ] `go mod tidy` completes successfully
- [ ] `go build ./...` compiles without errors
- [ ] `bun test ./middleware/...` runs all tests with 100% pass rate
- [ ] `curl -s http://localhost:8080/metrics | grep "library_http_request_count_total"` returns Prometheus metrics
- [ ] `curl -s http://localhost:8080/ping` returns `{"message":"pong"}` and increments counter
- [ ] Application starts and serves existing API endpoints without breaking existing functionality

### Must Have
- CounterVec metric: `library_http_request_count_total` with labels: method, path, status
- HistogramVec metric: `library_http_request_duration_seconds` with labels: method, path, default buckets
- Zap logger initialized with production config (`zap.NewProduction()`)
- Middleware that logs at appropriate levels: Error (>=500), Warn (>=400), Info (else)
- Metrics endpoint at `/metrics` exposed via `gin.WrapH(promhttp.Handler())`
- Normalized path labels using `c.FullPath()` (route patterns, not exact paths)
- Full TDD test suite with RED-GREEN-REFACTOR cycle

### Must NOT Have (Guardrails)
- **DO NOT modify existing middleware** (`backend/middleware/middleware.go` - session, auth)
- **DO NOT modify CORS configuration** in main.go
- **DO NOT change duplicate gin.Logger()/gin.Recovery() calls** at lines 66-67 (document but don't touch)
- **DO NOT modify route definitions** in `routes/*.go`
- **DO NOT add configuration management** beyond specified (no env vars for logger level)
- **DO NOT add extra metrics** beyond the two specified
- **DO NOT add log aggregation** or external integrations (no ELK, Loki, CloudWatch)
- **DO NOT add Prometheus push gateway** support (only /metrics scraping endpoint)
- **DO NOT modify graceful shutdown logic** (lines 90-111 in main.go)
- **DO NOT change existing log language** to English if existing logs are Chinese

---

## Verification Strategy

> **ZERO HUMAN INTERVENTION** — ALL verification is agent-executed. No exceptions.
> Acceptance criteria requiring "user manually tests/confirms" are FORBIDDEN.

### Test Decision
- **Infrastructure exists**: YES (Go has built-in testing, will use `go test`/`bun test`)
- **Automated tests**: YES (TDD - RED-GREEN-REFACTOR cycle)
- **Framework**: Go standard testing + `github.com/stretchr/testify` (if not present, will use standard assertions)
- **If TDD**: Each task follows RED (failing test) → GREEN (minimal impl) → REFACTOR

### QA Policy
Every task MUST include agent-executed QA scenarios (see TODO template below).
Evidence saved to `.sisyphus/evidence/task-{N}-{scenario-slug}.{ext}`.

- **Frontend/UI**: N/A (this is backend middleware)
- **API/Backend**: Use Bash (curl) — Send requests, assert status + response fields, verify /metrics endpoint
- **Library/Module**: Use Bash (bun test / go test) — Run tests, assert pass rate
- **Metrics**: Use Bash (curl + grep) — Verify Prometheus format output

---

## Execution Strategy

### Parallel Execution Waves

> Maximize throughput by grouping independent tasks into parallel waves.
> Each wave completes before the next begins.
> Target: 5-8 tasks per wave. Fewer than 3 per wave (except final) = under-splitting.

```
Wave 1 (Start Immediately — dependencies + metrics foundation + test setup):
├── Task 1: Add Zap and Prometheus dependencies to go.mod [quick]
├── Task 2: Create prometheus.go with metric definitions [quick]
├── Task 3: Create logger.go with InitLogger() factory [quick]
├── Task 4: Create test helper utilities (test context, mock requests) [quick]
├── Task 5: Test: Prometheus metrics RED (failing tests) [quick]
├── Task 6: Test: Middleware RED (failing tests) [quick]
└── Task 7: TDD GREEN: Implement basic Prometheus metrics registration [quick]

Wave 2 (After Wave 1 — core implementation + TDD GREEN):
├── Task 8: TDD GREEN: Implement middleware logic (start/timing/next/end) [deep]
├── Task 9: TDD GREEN: Implement metrics recording (counter + histogram) [deep]
├── Task 10: TDD GREEN: Implement Zap logging with level logic [deep]
├── Task 11: Test: Path normalization (c.FullPath()) edge cases [deep]
├── Task 12: Test: Error/warn/info level logic verification [deep]
└── Task 13: Test: Prometheus metrics format validation [quick]

Wave 3 (After Wave 2 — integration + main.go + final tests):
├── Task 14: Update main.go - Initialize logger with InitLogger() [quick]
├── Task 15: Update main.go - Register middleware in correct order [quick]
├── Task 16: Update main.go - Add /metrics endpoint [quick]
├── Task 17: Update main.go - Add /ping test route [quick]
├── Task 18: TDD REFACTOR: Code review and cleanup [quick]
├── Task 19: Integration test - Verify metrics endpoint works end-to-end [deep]
└── Task 20: Integration test - Verify logging for different status codes [deep]

Wave FINAL (After ALL tasks — independent review, 4 parallel):
├── Task F1: Plan compliance audit (oracle)
├── Task F2: Code quality review (unspecified-high)
├── Task F3: Real manual QA (unspecified-high)
└── Task F4: Scope fidelity check (deep)

Critical Path: Task 1 → Task 2 → Task 5 → Task 7 → Task 8 → Task 14 → Task 19 → F1-F4
Parallel Speedup: ~65% faster than sequential
Max Concurrent: 7 (Waves 1 & 2)
```

### Dependency Matrix

- **1-7**: — — 8-13, 1
- **8**: 3, 5, 7 — 11-12, 2
- **9**: 2, 5, 7 — 11-13, 2
- **10**: 3, 6 — 12, 2
- **11**: 8, 9 — 13, 3
- **12**: 8, 10 — 13, 3
- **13**: 2, 9 — 19, 3
- **14**: 3 — 15-17, 3
- **15**: 8-10, 14 — 17, 3
- **16**: 2, 14, 15 — 17, 3
- **17**: 14, 15 — 19, 3
- **18**: 8-13 — 19-20, 3
- **19**: 14-18 — 20, 4
- **20**: 14-18 — F1-F4, 4

> This is abbreviated for reference. YOUR generated plan must include the FULL matrix for ALL tasks.

### Agent Dispatch Summary

- **1**: **7** — T1 → `quick`, T2 → `quick`, T3 → `quick`, T4 → `quick`, T5 → `quick`, T6 → `quick`, T7 → `quick`
- **2**: **6** — T8 → `deep`, T9 → `deep`, T10 → `deep`, T11 → `deep`, T12 → `deep`, T13 → `quick`
- **3**: **7** — T14 → `quick`, T15 → `quick`, T16 → `quick`, T17 → `quick`, T18 → `quick`, T19 → `deep`, T20 → `deep`
- **FINAL**: **4** — F1 → `oracle`, F2 → `unspecified-high`, F3 → `unspecified-high`, F4 → `deep`

---

## TODOs

> Implementation + Test = ONE Task. Never separate.
> EVERY task MUST have: Recommended Agent Profile + Parallelization info + QA Scenarios.
> **A task WITHOUT QA Scenarios is INCOMPLETE. No exceptions.**

- [ ] 1. **Add Zap and Prometheus Dependencies to go.mod**

  **What to do**:
  - Add `go.uber.org/zap` dependency to `backend/go.mod`
  - Add `github.com/prometheus/client_golang/prometheus` dependency
  - Add `github.com/prometheus/client_golang/prometheus/promhttp` dependency
  - Run `go mod tidy` to resolve and download dependencies
  - Verify dependencies are installed correctly by checking `go list -m`

  **Must NOT do**:
  - Do NOT add any other dependencies beyond the three specified
  - Do NOT modify existing dependencies or versions
  - Do NOT remove any existing dependencies

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Simple dependency addition with standard go commands
  - **Skills**: [] (no special skills needed)

  **Parallelization**:
  - **Can Run In Parallel**: YES (but should run first as it unblocks all other tasks)
  - **Parallel Group**: Wave 1 (with Tasks 2-7)
  - **Blocks**: None (other tasks depend on these dependencies)
  - **Blocked By**: None (can start immediately)

  **References**:
  - `backend/go.mod` - Current dependencies file to edit
  - `https://pkg.go.dev/go.uber.org/zap` - Zap official documentation
  - `https://pkg.go.dev/github.com/prometheus/client_golang/prometheus` - Prometheus Go client docs

  **Acceptance Criteria**:
  - [ ] `backend/go.mod` includes `go.uber.org/zap`
  - [ ] `backend/go.mod` includes `github.com/prometheus/client_golang/prometheus`
  - [ ] `backend/go.mod` includes `github.com/prometheus/client_golang/prometheus/promhttp`
  - [ ] `go mod tidy` completes without errors
  - [ ] `go list -m go.uber.org/zap` returns version info
  - [ ] `go list -m github.com/prometheus/client_golang/prometheus` returns version info

  **QA Scenarios**:
  ```
  Scenario: Verify dependencies are installed
    Tool: Bash
    Preconditions: Dependencies added to go.mod
    Steps:
      1. cd backend
      2. go mod tidy
      3. go list -m go.uber.org/zap
      4. go list -m github.com/prometheus/client_golang/prometheus
      5. go list -m github.com/prometheus/client_golang/prometheus/promhttp
    Expected Result: All three commands return valid version information (no "not found" errors)
    Failure Indicators: Any command returns "not found", go mod tidy fails with errors
    Evidence: .sisyphus/evidence/task-1-dependencies-installed.txt

  Scenario: Verify no extra dependencies added
    Tool: Bash
    Preconditions: Dependencies added
    Steps:
      1. cd backend
      2. grep -c "require" go.mod | grep -E "^[0-9]+$"
    Expected Result: Dependency count should increase by exactly 3 (Zap + 2 Prometheus packages)
    Failure Indicators: Dependency count increases by more than 3
    Evidence: .sisyphus/evidence/task-1-dependency-count.txt
  ```

  **Commit**: YES
  - Message: `chore(deps): add zap and prometheus dependencies`
  - Files: `backend/go.mod`, `backend/go.sum`
  - Pre-commit: `go mod tidy`

- [ ] 2. **Create prometheus.go with Metric Definitions**

  **What to do**:
  - Create `backend/middleware/prometheus.go` file
  - Define `init()` function that registers metrics with Prometheus default registry
  - Create CounterVec: `library_http_request_count_total`
    - Help: "Total number of HTTP requests"
    - Labels: `method`, `path`, `status`
  - Create HistogramVec: `library_http_request_duration_seconds`
    - Help: "HTTP request latency in seconds"
    - Labels: `method`, `path`
    - Buckets: `prometheus.DefBuckets` (default buckets)
  - Export metric variables for use in middleware

  **Must NOT do**:
  - Do NOT create any other metrics beyond the two specified
  - Do NOT add custom bucket configuration (use prometheus.DefBuckets)
  - Do NOT use generic metric names (use "library_" prefix)

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Straightforward metric definition with standard Prometheus patterns
  - **Skills**: [] (no special skills needed)

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 3-7)
  - **Blocks**: Task 9 (metrics recording implementation)
  - **Blocked By**: Task 1 (dependencies must be installed first)

  **References**:
  - `https://pkg.go.dev/github.com/prometheus/client_golang/prometheus#CounterOpts` - CounterVec documentation
  - `https://pkg.go.dev/github.com/prometheus/client_golang/prometheus#HistogramOpts` - HistogramVec documentation
  - `https://pkg.go.dev/github.com/prometheus/client_golang/prometheus#DefBuckets` - Default buckets definition

  **Acceptance Criteria**:
  - [ ] File `backend/middleware/prometheus.go` created
  - [ ] CounterVec `library_http_request_count_total` defined with correct labels
  - [ ] HistogramVec `library_http_request_duration_seconds` defined with correct labels
  - [ ] Metrics registered in `init()` with `prometheus.MustRegister()`
  - [ ] `go build ./middleware/...` compiles without errors

  **QA Scenarios**:
  ```
  Scenario: Verify metrics are registered correctly
    Tool: Bash (go test)
    Preconditions: prometheus.go created
    Steps:
      1. cd backend
      2. cat << 'EOF' > /tmp/test_metrics.go
      package main
      import (
        _ "github.com/Dailiduzhou/library_manage_sys/middleware"
        "github.com/prometheus/client_golang/prometheus"
        "fmt"
      )
      func main() {
        metrics, _ := prometheus.DefaultGatherer.Gather()
        foundCount := false
        foundDuration := false
        for _, m := range metrics {
          if m.GetName() == "library_http_request_count_total" {
            foundCount = true
          }
          if m.GetName() == "library_http_request_duration_seconds" {
            foundDuration = true
          }
        }
        if foundCount && foundDuration {
          fmt.Println("OK: Both metrics registered")
        } else {
          fmt.Printf("FAIL: Count=%v Duration=%v\n", foundCount, foundDuration)
        }
      }
      EOF
      3. go run /tmp/test_metrics.go
    Expected Result: Output "OK: Both metrics registered"
    Failure Indicators: Metric not found, compilation errors, runtime errors
    Evidence: .sisyphus/evidence/task-2-metrics-registered.txt

  Scenario: Verify metric metadata is correct
    Tool: Bash
    Preconditions: Metrics registered
    Steps:
      1. cd backend
      2. cat << 'EOF' > /tmp/test_metadata.go
      package main
      import (
        _ "github.com/Dailiduzhou/library_manage_sys/middleware"
        "github.com/prometheus/client_golang/prometheus"
        "fmt"
      )
      func main() {
        metrics, _ := prometheus.DefaultGatherer.Gather()
        for _, m := range metrics {
          if m.GetName() == "library_http_request_count_total" {
            if m.GetHelp() != "Total number of HTTP requests" {
              fmt.Printf("FAIL: Wrong help text: %s\n", m.GetHelp())
              return
            }
            if len(m.Metric) != 0 { // Should be 0 initially
              fmt.Println("FAIL: Counter should start at 0")
              return
            }
          }
        }
        fmt.Println("OK: Metadata correct")
      }
      EOF
      3. go run /tmp/test_metadata.go
    Expected Result: Output "OK: Metadata correct"
    Failure Indicators: Wrong help text, wrong labels, non-zero initial values
    Evidence: .sisyphus/evidence/task-2-metrics-metadata.txt
  ```

  **Commit**: YES
  - Message: `feat(middleware): add prometheus metrics definitions`
  - Files: `backend/middleware/prometheus.go`
  - Pre-commit: `go build ./middleware/...`

- [ ] 3. **Create logger.go with InitLogger() Factory Function**

  **What to do**:
  - Create `backend/middleware/logger.go` file
  - Create `InitLogger()` function that returns `*zap.Logger`
  - Use `zap.NewProduction()` to configure logger with production settings
  - Return logger instance for use in middleware
  - Handle errors appropriately (production config should not fail, but defer/flush on shutdown)

  **Must NOT do**:
  - Do NOT add configuration via environment variables (always use production config)
  - Do NOT create dev/staging configs
  - Do NOT use `zap.NewDevelopment()` or other config modes

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Simple factory function with standard Zap production config
  - **Skills**: [] (no special skills needed)

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1-2, 4-7)
  - **Blocks**: Task 10 (middleware logging implementation)
  - **Blocked By**: Task 1 (Zap dependency must be installed)

  **References**:
  - `https://pkg.go.dev/go.uber.org/zap#NewProduction` - Zap production config documentation
  - `backend/middleware/middleware.go:1-15` - Import patterns and package structure

  **Acceptance Criteria**:
  - [ ] File `backend/middleware/logger.go` created
  - [ ] `InitLogger()` function defined returning `*zap.Logger`
  - [ ] Uses `zap.NewProduction()` configuration
  - [ ] `go build ./middleware/...` compiles without errors
  - [ ] Logger is usable (can call methods like Info(), Error(), etc.)

  **QA Scenarios**:
  ```
  Scenario: Verify logger initialization
    Tool: Bash (go test)
    Preconditions: logger.go created
    Steps:
      1. cd backend
      2. cat << 'EOF' > /tmp/test_logger.go
      package main
      import (
        "github.com/Dailiduzhou/library_manage_sys/middleware"
        "go.uber.org/zap"
      )
      func main() {
        logger := middleware.InitLogger()
        if logger == nil {
          println("FAIL: Logger is nil")
          return
        }
        logger.Info("Test log message")
        logger.Sync()
        println("OK: Logger initialized successfully")
      }
      EOF
      3. go run /tmp/test_logger.go
    Expected Result: Output "OK: Logger initialized successfully"
    Failure Indicators: Logger is nil, compilation errors, runtime errors
    Evidence: .sisyphus/evidence/task-3-logger-init.txt

  Scenario: Verify logger uses production config
    Tool: Bash
    Preconditions: Logger initialized
    Steps:
      1. cd backend
      2. cat << 'EOF' > /tmp/test_prod_config.go
      package main
      import (
        "github.com/Dailiduzhou/library_manage_sys/middleware"
        "go.uber.org/zap"
        "os"
      )
      func main() {
        logger := middleware.InitLogger()
        // Capture stdout to verify JSON format (production logs are JSON)
        old := os.Stdout
        r, w, _ := os.Pipe()
        os.Stdout = w
        logger.Info("test", zap.String("key", "value"))
        w.Close()
        os.Stdout = old
        buf := make([]byte, 1024)
        n, _ := r.Read(buf)
        output := string(buf[:n])
        if output == "" {
          println("FAIL: No output")
          return
        }
        // Production logs are JSON, should contain quotes
        if !contains(output, `"level"`) && !contains(output, `"msg"`) {
          println("FAIL: Not JSON format")
          return
        }
        println("OK: Production config verified")
      }
      func contains(s, substr string) bool { return len(s) >= len(substr) && s[:len(substr)] == substr || (len(s) > len(substr) && contains(s[1:], substr)) }
      EOF
      3. go run /tmp/test_prod_config.go
    Expected Result: Output "OK: Production config verified"
    Failure Indicators: Logs not in JSON format (plain text indicates dev config)
    Evidence: .sisyphus/evidence/task-3-logger-prod-config.txt
  ```

  **Commit**: YES
  - Message: `feat(middleware): add zap logger factory`
  - Files: `backend/middleware/logger.go`
  - Pre-commit: `go build ./middleware/...`

- [ ] 4. **Create Test Helper Utilities**

  **What to do**:
  - Create `backend/middleware/testutil.go` file (package `middleware_test` or use build tags)
  - Create helper function to create test Gin context with mock request
  - Create helper function to create test response recorder
  - Create helper function to execute middleware in test environment
  - Document usage in comments

  **Must NOT do**:
  - Do NOT create production code in test files
  - Do NOT add test utilities that are not needed for TDD
  - Do NOT create complex mocking frameworks

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Simple test helpers for middleware testing
  - **Skills**: [] (no special skills needed)

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1-3, 5-7)
  - **Blocks**: Tasks 5-6 (TDD RED tests need these helpers)
  - **Blocked By**: None (can start immediately)

  **References**:
  - `https://pkg.go.dev/github.com/gin-gonic/gin#Context` - Gin context documentation
  - `https://pkg.go.dev/github.com/gin-gonic/gin#TestResponseRecorder` - Gin test response recorder

  **Acceptance Criteria**:
  - [ ] File `backend/middleware/testutil.go` created
  - [ ] Helper function to create test Gin context exists
  - [ ] Helper function to create test response recorder exists
  - [ ] `go build ./middleware/...` compiles without errors
  - [ ] Helpers are usable in test files (can import and use)

  **QA Scenarios**:
  ```
  Scenario: Verify test helpers compile
    Tool: Bash (go build)
    Preconditions: testutil.go created
    Steps:
      1. cd backend/middleware
      2. go build ./...
    Expected Result: No compilation errors
    Failure Indicators: Compilation errors, missing imports
    Evidence: .sisyphus/evidence/task-4-test-helpers-compile.txt

  Scenario: Verify test helpers are importable
    Tool: Bash (go test)
    Preconditions: testutil.go created
    Steps:
      1. cd backend
      2. cat << 'EOF' > /tmp/test_helpers.go
      package main
      import (
        _ "github.com/Dailiduzhou/library_manage_sys/middleware"
      )
      func main() {
        println("OK: Helpers importable")
      }
      EOF
      3. go run /tmp/test_helpers.go
    Expected Result: Output "OK: Helpers importable"
    Failure Indicators: Import errors, build errors
    Evidence: .sisyphus/evidence/task-4-test-helpers-import.txt
  ```

  **Commit**: NO (group with Task 5-6)

- [ ] 5. **Test: Prometheus Metrics RED (Failing Tests)**

  **What to do**:
  - Create `backend/middleware/prometheus_test.go` file
  - Write test for CounterVec: Verify metric is registered and has correct labels
  - Write test for HistogramVec: Verify metric is registered with correct labels and buckets
  - Write test for counter increment: Verify Increment() method works
  - Write test for histogram observation: Verify Observe() method records duration
  - Ensure tests FAIL (RED) before implementation (Tasks 7 will make them GREEN)
  - Follow Go testing best practices (table-driven tests, subtests)

  **Must NOT do**:
  - Do NOT implement the metrics registration in this task (that's Task 7)
  - Do NOT make tests pass artificially
  - Do NOT skip tests or mark them as t.Skip()

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Standard Go test writing, follows TDD RED phase
  - **Skills**: [] (no special skills needed)

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1-4, 6-7)
  - **Blocks**: Task 7 (GREEN phase - implementation)
  - **Blocked By**: Task 1 (dependencies), Task 4 (test helpers)

  **References**:
  - `https://pkg.go.dev/github.com/prometheus/client_golang/prometheus` - Prometheus client for tests
  - `https://github.com/Dailiduzhou/library_manage_sys/backend/middleware/prometheus.go` - Implementation to test (once Task 7 is done)

  **Acceptance Criteria**:
  - [ ] File `backend/middleware/prometheus_test.go` created
  - [ ] Test for CounterVec registration exists
  - [ ] Test for HistogramVec registration exists
  - [ ] Test for counter increment exists
  - [ ] Test for histogram observation exists
  - [ ] Tests compile: `go test -c ./middleware/...`
  - [ ] Tests FAIL when run: `bun test ./middleware/...` (expected RED state)

  **QA Scenarios**:
  ```
  Scenario: Verify tests compile
    Tool: Bash (go test)
    Preconditions: prometheus_test.go created
    Steps:
      1. cd backend
      2. go test -c ./middleware/...
    Expected Result: No compilation errors
    Failure Indicators: Compilation errors, syntax errors, missing imports
    Evidence: .sisyphus/evidence/task-5-tests-compile.txt

  Scenario: Verify tests are in RED state (failing)
    Tool: Bash (go test)
    Preconditions: Tests compile
    Steps:
      1. cd backend
      2. bun test ./middleware/... -v 2>&1 | tee /tmp/test_output.txt
      3. grep -E "(FAIL|PASS)" /tmp/test_output.txt | tail -10
    Expected Result: Tests should FAIL (RED state before implementation)
    Failure Indicators: Tests pass unexpectedly ( GREEN state too early)
    Evidence: .sisyphus/evidence/task-5-tests-red-state.txt
  ```

  **Commit**: YES
  - Message: `test(middleware): add RED tests for prometheus metrics`
  - Files: `backend/middleware/prometheus_test.go`, `backend/middleware/testutil.go` (Task 4)
  - Pre-commit: `go test -c ./middleware/...`

- [ ] 6. **Test: Middleware RED (Failing Tests)**

  **What to do**:
  - Create `backend/middleware/logging_and_metrics_test.go` file
  - Write test for middleware timing: Verifies start time and end time are recorded
  - Write test for path normalization: Verifies `c.FullPath()` returns route pattern
  - Write test for metrics recording: Verifies counter and histogram are called
  - Write test for log levels: Verifies error (>=500), warn (>=400), info (else) logic
  - Write test for metadata extraction: Verifies method, path, IP, status are captured
  - Ensure tests FAIL (RED) before implementation (Tasks 8-10 will make them GREEN)
  - Follow Go testing best practices

  **Must NOT do**:
  - Do NOT implement the middleware logic in this task (that's Tasks 8-10)
  - Do NOT make tests pass artificially
  - Do NOT skip tests or mark them as t.Skip()

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Standard Go test writing, follows TDD RED phase
  - **Skills**: [] (no special skills needed)

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1-5, 7)
  - **Blocks**: Tasks 8-10 (GREEN phase - implementation)
  - **Blocked By**: Task 1 (dependencies), Task 3 (logger), Task 4 (test helpers)

  **References**:
  - `https://pkg.go.dev/github.com/gin-gonic/gin#Test` - Gin testing utilities
  - `https://pkg.go.dev/github.com/Dailiduzhou/library_manage_sys/backend/middleware/logging_and_metrics.go` - Implementation to test (once Tasks 8-10 are done)

  **Acceptance Criteria**:
  - [ ] File `backend/middleware/logging_and_metrics_test.go` created
  - [ ] Test for timing exists
  - [ ] Test for path normalization exists
  - [ ] Test for metrics recording exists
  - [ ] Test for log levels exists
  - [ ] Test for metadata extraction exists
  - [ ] Tests compile: `go test -c ./middleware/...`
  - [ ] Tests FAIL when run: `bun test ./middleware/...` (expected RED state)

  **QA Scenarios**:
  ```
  Scenario: Verify tests compile
    Tool: Bash (go test)
    Preconditions: logging_and_metrics_test.go created
    Steps:
      1. cd backend
      2. go test -c ./middleware/...
    Expected Result: No compilation errors
    Failure Indicators: Compilation errors, syntax errors, missing imports
    Evidence: .sisyphus/evidence/task-6-middleware-tests-compile.txt

  Scenario: Verify tests are in RED state (failing)
    Tool: Bash (go test)
    Preconditions: Tests compile
    Steps:
      1. cd backend
      2. bun test ./middleware/... -v 2>&1 | tee /tmp/test_output.txt
      3. grep -E "(FAIL|PASS)" /tmp/test_output.txt | tail -10
    Expected Result: Tests should FAIL (RED state before implementation)
    Failure Indicators: Tests pass unexpectedly (GREEN state too early)
    Evidence: .sisyphus/evidence/task-6-middleware-tests-red.txt
  ```

  **Commit**: YES
  - Message: `test(middleware): add RED tests for logging and metrics middleware`
  - Files: `backend/middleware/logging_and_metrics_test.go`
  - Pre-commit: `go test -c ./middleware/...`

- [ ] 7. **TDD GREEN: Implement Basic Prometheus Metrics Registration**

  **What to do**:
  - Ensure `backend/middleware/prometheus.go` is properly implemented (from Task 2)
  - Verify metrics are registered with correct names, labels, and help text
  - Verify CounterVec and HistogramVec are exported variables
  - Verify metrics use "library_" prefix to avoid conflicts
  - Verify HistogramVec uses `prometheus.DefBuckets` (default buckets)
  - Run tests from Task 5 to ensure they now PASS (GREEN state)
  - If tests fail, fix implementation until all tests pass

  **Must NOT do**:
  - Do NOT add any additional metrics beyond the two specified
  - Do NOT use custom buckets (use prometheus.DefBuckets)
  - Do NOT remove or modify tests

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Making Task 5 tests pass by implementing metrics registration
  - **Skills**: [] (no special skills needed)

  **Parallelization**:
  - **Can Run In Parallel**: NO (must wait for Task 5 RED tests)
  - **Parallel Group**: Wave 1 (sequential after Task 5)
  - **Blocks**: Tasks 8-10 (middleware implementation needs metrics)
  - **Blocked By**: Task 5 (RED tests), Task 1 (dependencies), Task 2 (prometheus.go skeleton)

  **References**:
  - `backend/middleware/prometheus_test.go` - Tests that must now pass
  - `backend/middleware/prometheus.go` - Implementation file

  **Acceptance Criteria**:
  - [ ] All tests from Task 5 PASS: `bun test ./middleware/... -run "TestPrometheus"`
  - [ ] CounterVec registered with correct labels: method, path, status
  - [ ] HistogramVec registered with correct labels: method, path
  - [ ] HistogramVec uses prometheus.DefBuckets
  - [ ] Metrics use "library_" prefix
  - [ ] No compilation errors

  **QA Scenarios**:
  ```
  Scenario: Verify all Prometheus tests pass (GREEN state)
    Tool: Bash (go test)
    Preconditions: RED tests from Task 5
    Steps:
      1. cd backend
      2. bun test ./middleware/... -run "TestPrometheus" -v 2>&1 | tee /tmp/test_output.txt
      3. grep -E "PASS:.*TestPrometheus" /tmp/test_output.txt | wc -l
    Expected Result: All Prometheus tests PASS (count > 0)
    Failure Indicators: Any test FAIL, tests still in RED state
    Evidence: .sisyphus/evidence/task-7-prometheus-tests-green.txt

  Scenario: Verify metrics are correctly named and labeled
    Tool: Bash (go run)
    Preconditions: Metrics registered
    Steps:
      1. cd backend
      2. cat << 'EOF' > /tmp/verify_metrics.go
      package main
      import (
        _ "github.com/Dailiduzhou/library_manage_sys/middleware"
        "github.com/prometheus/client_golang/prometheus"
        "fmt"
      )
      func main() {
        metrics, _ := prometheus.DefaultGatherer.Gather()
        for _, m := range metrics {
          if m.GetName() == "library_http_request_count_total" {
            fmt.Printf("OK: Counter name correct, labels=%v\n", m.GetLabel())
          } else if m.GetName() == "library_http_request_duration_seconds" {
            fmt.Printf("OK: Histogram name correct, labels=%v\n", m.GetLabel())
          }
        }
      }
      EOF
      3. go run /tmp/verify_metrics.go
    Expected Result: Both metrics found with correct names
    Failure Indicators: Metrics not found, wrong names, wrong labels
    Evidence: .sisyphus/evidence/task-7-metrics-verified.txt
  ```

  **Commit**: YES (already done in Task 2, this is validation)

- [ ] 8. **TDD GREEN: Implement Core Middleware Logic**

  **What to do**:
  - Create `backend/middleware/logging_and_metrics.go` file
  - Implement `GinLoggerAndMetrics(logger *zap.Logger) gin.HandlerFunc` function
  - Record request start time using `time.Now()`
  - Call `c.Next()` to process the request chain
  - Calculate latency: `time.Since(start).Seconds()`
  - Extract metadata: `c.Request.Method`, `c.FullPath()`, `c.ClientIP()`, `c.Writer.Status()`
  - Make tests from Task 6 PASS (timing, metadata extraction, path normalization)
  - Follow TDD GREEN phase: implement only what's needed to make tests pass

  **Must NOT do**:
  - Do NOT implement metrics recording in this task (that's Task 9)
  - Do NOT implement logging in this task (that's Task 10)
  - Do NOT add extra functionality beyond what tests require
  - Do NOT modify tests to make them pass

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Core middleware logic with Gin context handling, timing, and metadata extraction
  - **Skills**: [] (no special skills needed)

  **Parallelization**:
  - **Can Run In Parallel**: NO (must wait for Task 6 RED tests)
  - **Parallel Group**: Wave 2 (with Tasks 9-13)
  - **Blocks**: Tasks 9-10 (depend on timing and metadata from this task)
  - **Blocked By**: Task 3 (logger), Task 6 (RED tests), Task 7 (metrics available)

  **References**:
  - `backend/middleware/logging_and_metrics_test.go` - Tests that must now pass (timing, metadata, path)
  - `backend/middleware/logger.go` - InitLogger() function
  - `backend/middleware/middleware.go:90-129` - Example Gin middleware (AuthRequired) for reference
  - `https://pkg.go.dev/github.com/gin-gonic/gin#Context.FullPath` - FullPath() documentation

  **Acceptance Criteria**:
  - [ ] File `backend/middleware/logging_and_metrics.go` created
  - [ ] `GinLoggerAndMetrics(logger *zap.Logger)` function exists
  - [ ] Timing tests PASS: `bun test ./middleware/... -run "TestTiming"`
  - [ ] Path normalization tests PASS: `bun test ./middleware/... -run "TestPath"`
  - [ ] Metadata extraction tests PASS: `bun test ./middleware/... -run "TestMetadata"`
  - [ ] Middleware calls `c.Next()` correctly
  - [ ] No compilation errors

  **QA Scenarios**:
  ```
  Scenario: Verify timing tests pass (GREEN state)
    Tool: Bash (go test)
    Preconditions: RED tests from Task 6
    Steps:
      1. cd backend
      2. bun test ./middleware/... -run "TestTiming" -v 2>&1 | tee /tmp/test_output.txt
      3. grep -E "PASS:.*TestTiming" /tmp/test_output.txt | wc -l
    Expected Result: All timing tests PASS (count > 0)
    Failure Indicators: Any test FAIL, tests still in RED state
    Evidence: .sisyphus/evidence/task-8-timing-tests-green.txt

  Scenario: Verify middleware records start time and calculates latency
    Tool: Bash (curl)
    Preconditions: Middleware implemented, main.go updated (later tasks)
    Steps:
      1. cd backend
      2. go run main.go > /tmp/server.log 2>&1 &
      3. sleep 2
      4. curl -s http://localhost:8080/ping
      5. sleep 1
      6. pkill -f "go run main.go"
      7. grep -i "latency" /tmp/server.log | tail -1
    Expected Result: Log shows latency in seconds (e.g., "latency":0.001234)
    Failure Indicators: No latency logged, latency in wrong units (milliseconds instead of seconds)
    Evidence: .sisyphus/evidence/task-8-latency-logged.txt
  ```

  **Commit**: YES
  - Message: `feat(middleware): implement core middleware logic (timing, metadata, path)`
  - Files: `backend/middleware/logging_and_metrics.go`
  - Pre-commit: `go test -run "TestTiming|TestPath|TestMetadata" ./middleware/...`

- [ ] 9. **TDD GREEN: Implement Metrics Recording**

  **What to do**:
  - Extend `GinLoggerAndMetrics()` function to record Prometheus metrics
  - Increment CounterVec: `httpRequestCountTotal.WithLabelValues(method, path, strconv.Itoa(status)).Inc()`
  - Observe HistogramVec: `httpRequestDurationSeconds.WithLabelValues(method, path).Observe(latency)`
  - Use `c.FullPath()` for path label (normalized route pattern)
  - Convert status code to string using `strconv.Itoa()` for label
  - Make tests from Task 6 PASS (metrics recording tests)
  - Follow TDD GREEN phase

  **Must NOT do**:
  - Do NOT add any additional metrics beyond the two specified
  - Do NOT use exact path for label (must use normalized route pattern)
  - Do NOT forget to convert status to string for Counter label

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Metrics recording with Prometheus client, label handling
  - **Skills**: [] (no special skills needed)

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 8, 10-13)
  - **Blocks**: Tasks 11-12 (edge case tests)
  - **Blocked By**: Task 7 (metrics available), Task 8 (timing, metadata, path), Task 6 (RED tests)

  **References**:
  - `backend/middleware/logging_and_metrics_test.go` - Tests that must now pass (metrics recording)
  - `backend/middleware/prometheus.go` - Metric variables to use
  - `https://pkg.go.dev/github.com/prometheus/client_golang/prometheus#Vec.WithLabelValues` - WithLabelValues() documentation

  **Acceptance Criteria**:
  - [ ] Counter incremented with correct labels: method, path, status (as string)
  - [ ] Histogram observed with correct labels: method, path, latency (seconds)
  - [ ] Metrics recording tests PASS: `bun test ./middleware/... -run "TestMetrics"`
  - [ ] No compilation errors
  - [ ] Counter shows increment after requests

  **QA Scenarios**:
  ```
  Scenario: Verify metrics recording tests pass (GREEN state)
    Tool: Bash (go test)
    Preconditions: RED tests from Task 6
    Steps:
      1. cd backend
      2. bun test ./middleware/... -run "TestMetrics" -v 2>&1 | tee /tmp/test_output.txt
      3. grep -E "PASS:.*TestMetrics" /tmp/test_output.txt | wc -l
    Expected Result: All metrics recording tests PASS (count > 0)
    Failure Indicators: Any test FAIL, tests still in RED state
    Evidence: .sisyphus/evidence/task-9-metrics-tests-green.txt

  Scenario: Verify counter increments on requests
    Tool: Bash (curl)
    Preconditions: Middleware integrated in main.go (later tasks)
    Steps:
      1. cd backend
      2. go run main.go > /tmp/server.log 2>&1 &
      3. sleep 2
      4. INITIAL=$(curl -s http://localhost:8080/metrics | grep "library_http_request_count_total{method=\"GET\",path=\"/ping\",status=\"200\"" | awk '{print $2}')
      5. curl -s http://localhost:8080/ping > /dev/null
      6. curl -s http://localhost:8080/ping > /dev/null
      7. curl -s http://localhost:8080/ping > /dev/null
      8. FINAL=$(curl -s http://localhost:8080/metrics | grep "library_http_request_count_total{method=\"GET\",path=\"/ping\",status=\"200\"" | awk '{print $2}')
      9. DIFF=$((FINAL - INITIAL))
      10. pkill -f "go run main.go"
      11. echo "Counter incremented by: $DIFF"
    Expected Result: Counter incremented by 3 (one per /ping request)
    Failure Indicators: Counter unchanged, wrong increment value
    Evidence: .sisyphus/evidence/task-9-counter-incremented.txt

  Scenario: Verify histogram records duration
    Tool: Bash (curl)
    Preconditions: Middleware integrated in main.go (later tasks)
    Steps:
      1. cd backend
      2. go run main.go > /tmp/server.log 2>&1 &
      3. sleep 2
      4. curl -s http://localhost:8080/ping > /dev/null
      5. curl -s http://localhost:8080/metrics | grep "library_http_request_duration_seconds_sum{method=\"GET\",path=\"/ping\"}"
      6. pkill -f "go run main.go"
    Expected Result: Output shows sum > 0 (e.g., `library_http_request_duration_seconds_sum{...} 0.001234`)
    Failure Indicators: Sum is 0, histogram not recorded
    Evidence: .sisyphus/evidence/task-9-histogram-recorded.txt
  ```

  **Commit**: YES
  - Message: `feat(middleware): add metrics recording (counter + histogram)`
  - Files: `backend/middleware/logging_and_metrics.go`
  - Pre-commit: `go test -run "TestMetrics" ./middleware/...`

- [ ] 10. **TDD GREEN: Implement Zap Logging with Level Logic**

  **What to do**:
  - Extend `GinLoggerAndMetrics()` function to add Zap logging
  - Create zap.Field list: method, path, status, latency, ip
  - Implement log level logic:
    - If `status >= 500`: `logger.Error(msg, fields...)`
    - Else if `status >= 400`: `logger.Warn(msg, fields...)`
    - Else: `logger.Info(msg, fields...)`
  - Use Chinese log messages to match existing codebase pattern
  - Make tests from Task 6 PASS (log level tests)
  - Follow TDD GREEN phase

  **Must NOT do**:
  - Do NOT use English log messages (use Chinese to match existing codebase)
  - Do NOT log response bodies (security risk)
  - Do NOT change log level logic (>=500 error, >=400 warn, else info)

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Logging with Zap, conditional log levels, Chinese messages
  - **Skills**: [] (no special skills needed)

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 8-9, 11-13)
  - **Blocks**: Task 12 (log level verification tests)
  - **Blocked By**: Task 3 (logger), Task 8 (metadata), Task 6 (RED tests)

  **References**:
  - `backend/middleware/logging_and_metrics_test.go` - Tests that must now pass (log levels)
  - `backend/middleware/logger.go` - InitLogger() function
  - `backend/middleware/middleware.go` - Chinese log message patterns for reference
  - `https://pkg.go.dev/go.uber.org/zap#Logger.Info` - Zap logging methods

  **Acceptance Criteria**:
  - [ ] Log level logic implemented correctly (>=500 error, >=400 warn, else info)
  - [ ] Log messages in Chinese (matching existing codebase)
  - [ ] Log fields include: method, path, status, latency, ip
  - [ ] Log level tests PASS: `bun test ./middleware/... -run "TestLogLevel"`
  - [ ] No compilation errors

  **QA Scenarios**:
  ```
  Scenario: Verify log level tests pass (GREEN state)
    Tool: Bash (go test)
    Preconditions: RED tests from Task 6
    Steps:
      1. cd backend
      2. bun test ./middleware/... -run "TestLogLevel" -v 2>&1 | tee /tmp/test_output.txt
      3. grep -E "PASS:.*TestLogLevel" /tmp/test_output.txt | wc -l
    Expected Result: All log level tests PASS (count > 0)
    Failure Indicators: Any test FAIL, tests still in RED state
    Evidence: .sisyphus/evidence/task-10-loglevel-tests-green.txt

  Scenario: Verify 500 status logs at error level
    Tool: Bash (curl)
    Preconditions: Middleware integrated in main.go, endpoint returns 500
    Steps:
      1. cd backend
      2. # Temporarily add a route that returns 500 (or use existing if available)
      3. go run main.go > /tmp/server.log 2>&1 &
      4. sleep 2
      5. curl -s http://localhost:8080/test-500 > /dev/null 2>&1
      6. sleep 1
      7. pkill -f "go run main.go"
      8. grep -E '"level":"error"' /tmp/server.log | tail -1
    Expected Result: Log entry with level:"error" for 500 status
    Failure Indicators: No error-level log, wrong log level used
    Evidence: .sisyphus/evidence/task-10-error-level-log.txt

  Scenario: Verify 400 status logs at warn level
    Tool: Bash (curl)
    Preconditions: Middleware integrated in main.go, endpoint returns 400
    Steps:
      1. cd backend
      2. go run main.go > /tmp/server.log 2>&1 &
      3. sleep 2
      4. # Send invalid request that returns 400
      5. curl -s -X POST http://localhost:8080/api/books -d '{}' -H "Content-Type: application/json" > /dev/null 2>&1
      6. sleep 1
      7. pkill -f "go run main.go"
      8. grep -E '"level":"warn"' /tmp/server.log | tail -1
    Expected Result: Log entry with level:"warn" for 400 status
    Failure Indicators: No warn-level log, wrong log level used
    Evidence: .sisyphus/evidence/task-10-warn-level-log.txt

  Scenario: Verify 200 status logs at info level
    Tool: Bash (curl)
    Preconditions: Middleware integrated in main.go, /ping endpoint available
    Steps:
      1. cd backend
      2. go run main.go > /tmp/server.log 2>&1 &
      3. sleep 2
      4. curl -s http://localhost:8080/ping > /dev/null
      5. sleep 1
      6. pkill -f "go run main.go"
      7. grep -E '"level":"info"' /tmp/server.log | tail -1
    Expected Result: Log entry with level:"info" for 200 status
    Failure Indicators: No info-level log, wrong log level used
    Evidence: .sisyphus/evidence/task-10-info-level-log.txt
  ```

  **Commit**: YES
  - Message: `feat(middleware): add zap logging with level logic`
  - Files: `backend/middleware/logging_and_metrics.go`
  - Pre-commit: `go test -run "TestLogLevel" ./middleware/...`

- [ ] 11. **Test: Path Normalization Edge Cases**

  **What to do**:
  - Add tests to `backend/middleware/logging_and_metrics_test.go` for edge cases
  - Test: `c.FullPath()` returns route pattern (e.g., `/books/:id` not `/books/123`)
  - Test: 404 routes (no route match) - verify handling of nil or empty path
  - Test: OPTIONS requests (CORS preflight) - verify they're logged/metricized
  - Test: Static file requests (`/uploads/*`) - verify they're included
  - Test: Routes with multiple parameters (e.g., `/users/:id/books/:bookid`)
  - Ensure tests PASS after implementation

  **Must NOT do**:
  - Do NOT modify middleware implementation in this task (write tests first, verify they pass)
  - Do NOT skip testing edge cases
  - Do NOT assume edge cases will never happen

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Edge case testing for path normalization, 404s, CORS, static files
  - **Skills**: [] (no special skills needed)

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 8-10, 12-13)
  - **Blocks**: None (independent testing task)
  - **Blocked By**: Task 8 (middleware core logic), Task 9 (metrics)

  **References**:
  - `backend/middleware/logging_and_metrics.go` - Middleware implementation
  - `https://pkg.go.dev/github.com/gin-gonic/gin#Context.FullPath` - FullPath() documentation

  **Acceptance Criteria**:
  - [ ] Test for route pattern vs exact path exists
  - [ ] Test for 404 handling exists
  - [ ] Test for OPTIONS requests exists
  - [ ] Test for static file requests exists
  - [ ] Test for multi-parameter routes exists
  - [ ] All edge case tests PASS: `bun test ./middleware/... -run "TestPathEdgeCase"`
  - [ ] No test failures

  **QA Scenarios**:
  ```
  Scenario: Verify edge case tests pass
    Tool: Bash (go test)
    Preconditions: Edge case tests written
    Steps:
      1. cd backend
      2. bun test ./middleware/... -run "TestPathEdgeCase" -v
    Expected Result: All edge case tests PASS
    Failure Indicators: Any test FAIL
    Evidence: .sisyphus/evidence/task-11-edgecase-tests-pass.txt

  Scenario: Verify 404 routes are handled correctly
    Tool: Bash (curl)
    Preconditions: Middleware integrated in main.go
    Steps:
      1. cd backend
      2. go run main.go > /tmp/server.log 2>&1 &
      3. sleep 2
      4. curl -s http://localhost:8080/nonexistent-route > /dev/null
      5. sleep 1
      6. pkill -f "go run main.go"
      7. grep -E '"status":404' /tmp/server.log | tail -1
    Expected Result: Log entry for 404 status (even with no route match)
    Failure Indicators: No log for 404, crash on nil path
    Evidence: .sisyphus/evidence/task-11-404-handled.txt

  Scenario: Verify OPTIONS requests are logged
    Tool: Bash (curl)
    Preconditions: Middleware integrated in main.go
    Steps:
      1. cd backend
      2. go run main.go > /tmp/server.log 2>&1 &
      3. sleep 2
      4. curl -s -X OPTIONS http://localhost:8080/api/books -H "Origin: http://localhost:3000" -H "Access-Control-Request-Method: GET" > /dev/null
      5. sleep 1
      6. pkill -f "go run main.go"
      7. grep -E '"method":"OPTIONS"' /tmp/server.log | tail -1
    Expected Result: Log entry for OPTIONS request (CORS preflight)
    Failure Indicators: No log for OPTIONS request
    Evidence: .sisyphus/evidence/task-11-options-logged.txt
  ```

  **Commit**: YES
  - Message: `test(middleware): add path normalization edge case tests`
  - Files: `backend/middleware/logging_and_metrics_test.go`
  - Pre-commit: `go test -run "TestPathEdgeCase" ./middleware/...`

- [ ] 12. **Test: Error/Warn/Info Level Logic Verification**

  **What to do**:
  - Add tests to `backend/middleware/logging_and_metrics_test.go` for log levels
  - Test: Verify exact status code thresholds (500+ error, 400+ warn, 200-399 info)
  - Test: Verify 499 status logs at warn level
  - Test: Verify 501 status logs at error level
  - Test: Verify 401 status logs at warn level (auth failure)
  - Test: Verify 403 status logs at warn level (forbidden)
  - Test: Verify 404 status logs at warn level
  - Ensure tests PASS after implementation

  **Must NOT do**:
  - Do NOT modify middleware implementation in this task
  - Do NOT test wrong thresholds (use exact spec: >=500 error, >=400 warn, else info)

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Detailed log level testing at status code boundaries
  - **Skills**: [] (no special skills needed)

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 8-11, 13)
  - **Blocks**: None (independent testing task)
  - **Blocked By**: Task 10 (logging implementation)

  **References**:
  - `backend/middleware/logging_and_metrics.go` - Middleware log level logic

  **Acceptance Criteria**:
  - [ ] Test for 500+ status at error level exists
  - [ ] Test for 400-499 status at warn level exists
  - [ ] Test for 200-399 status at info level exists
  - [ ] Test for exact boundary codes (499, 501, 401, 403, 404) exists
  - [ ] All log level tests PASS: `bun test ./middleware/... -run "TestLogLevel"`
  - [ ] No test failures

  **QA Scenarios**:
  ```
  Scenario: Verify log level boundary tests pass
    Tool: Bash (go test)
    Preconditions: Log level tests written
    Steps:
      1. cd backend
      2. bun test ./middleware/... -run "TestLogLevel" -v
    Expected Result: All log level tests PASS (including boundary tests)
    Failure Indicators: Any test FAIL, wrong threshold detected
    Evidence: .sisyphus/evidence/task-12-loglevel-boundary-tests-pass.txt

  Scenario: Verify 499 status logs at warn (not error)
    Tool: Bash (go test)
    Preconditions: Log level tests written
    Steps:
      1. cd backend
      2. bun test ./middleware/... -run "TestLogLevel.*499" -v
    Expected Result: Test verifies 499 logs at warn level
    Failure Indicators: Test FAIL (499 logged at error level)
    Evidence: .sisyphus/evidence/task-12-499-warn-level.txt

  Scenario: Verify 501 status logs at error (not warn)
    Tool: Bash (go test)
    Preconditions: Log level tests written
    Steps:
      1. cd backend
      2. bun test ./middleware/... -run "TestLogLevel.*501" -v
    Expected Result: Test verifies 501 logs at error level
    Failure Indicators: Test FAIL (501 logged at warn level)
    Evidence: .sisyphus/evidence/task-12-501-error-level.txt
  ```

  **Commit**: YES
  - Message: `test(middleware): add log level boundary verification tests`
  - Files: `backend/middleware/logging_and_metrics_test.go`
  - Pre-commit: `go test -run "TestLogLevel" ./middleware/...`

- [ ] 13. **Test: Prometheus Metrics Format Validation**

  **What to do**:
  - Add tests to `backend/middleware/prometheus_test.go` for metrics format
  - Test: Verify CounterVec labels are correct (method, path, status as string)
  - Test: Verify HistogramVec labels are correct (method, path)
  - Test: Verify metric names match spec (library_http_request_count_total, library_http_request_duration_seconds)
  - Test: Verify HistogramVec buckets are Prometheus default buckets
  - Test: Verify metrics output in Prometheus text format when scraped
  - Ensure tests PASS

  **Must NOT do**:
  - Do NOT modify metrics implementation in this task
  - Do NOT accept non-standard bucket configurations

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Metrics format validation and Prometheus compatibility checks
  - **Skills**: [] (no special skills needed)

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 8-12)
  - **Blocks**: None (independent testing task)
  - **Blocked By**: Task 9 (metrics implementation)

  **References**:
  - `backend/middleware/prometheus.go` - Metrics definitions
  - `https://prometheus.io/docs/instrumenting/exposition_formats/` - Prometheus text format spec

  **Acceptance Criteria**:
  - [ ] Test for CounterVec labels exists
  - [ ] Test for HistogramVec labels exists
  - [ ] Test for metric names exists
  - [ ] Test for default buckets exists
  - [ ] Test for Prometheus text format exists
  - [ ] All format validation tests PASS: `bun test ./middleware/... -run "TestMetricsFormat"`
  - [ ] No test failures

  **QA Scenarios**:
  ```
  Scenario: Verify metrics format tests pass
    Tool: Bash (go test)
    Preconditions: Metrics format tests written
    Steps:
      1. cd backend
      2. bun test ./middleware/... -run "TestMetricsFormat" -v
    Expected Result: All metrics format tests PASS
    Failure Indicators: Any test FAIL
    Evidence: .sisyphus/evidence/task-13-format-tests-pass.txt

  Scenario: Verify Prometheus text format output
    Tool: Bash (curl)
    Preconditions: Middleware integrated in main.go, /metrics endpoint available
    Steps:
      1. cd backend
      2. go run main.go > /tmp/server.log 2>&1 &
      3. sleep 2
      4. curl -s http://localhost:8080/metrics | head -20
      5. pkill -f "go run main.go"
    Expected Result: Output in Prometheus text format (HELP, TYPE, metric lines)
    Failure Indicators: Non-Prometheus format, malformed output
    Evidence: .sisyphus/evidence/task-13-prometheus-format.txt

  Scenario: Verify Histogram uses default buckets
    Tool: Bash (curl)
    Preconditions: Middleware integrated, /metrics endpoint available
    Steps:
      1. cd backend
      2. go run main.go > /tmp/server.log 2>&1 &
      3. sleep 2
      4. curl -s http://localhost:8080/metrics | grep "library_http_request_duration_seconds_bucket"
      5. pkill -f "go run main.go"
    Expected Result: Output shows default bucket values (0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, +Inf)
    Failure Indicators: Wrong buckets, custom buckets detected
    Evidence: .sisyphus/evidence/task-13-default-buckets.txt
  ```

  **Commit**: YES
  - Message: `test(middleware): add Prometheus metrics format validation tests`
  - Files: `backend/middleware/prometheus_test.go`
  - Pre-commit: `go test -run "TestMetricsFormat" ./middleware/...`

- [ ] 14. **Update main.go - Initialize Logger with InitLogger()**

  **What to do**:
  - Import `middleware` package in `backend/main.go`
  - Initialize logger using `logger := middleware.InitLogger()` near top of main()
  - Ensure logger is available for middleware registration
  - Add `defer logger.Sync()` before main() exits to flush any pending logs
  - Do NOT modify any other parts of main.go in this task

  **Must NOT do**:
  - Do NOT change existing import statements unless necessary
  - Do NOT modify existing middleware registration (session, auth)
  - Do NOT remove or change CORS configuration
  - Do NOT modify graceful shutdown logic (lines 90-111)

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Simple logger initialization in main.go
  - **Skills**: [] (no special skills needed)

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 15-17, 18-20)
  - **Blocks**: Tasks 15-17 (middleware registration needs logger)
  - **Blocked By**: Task 3 (InitLogger() implementation)

  **References**:
  - `backend/middleware/logger.go` - InitLogger() function
  - `backend/main.go:37-38` - Current initialization code (DB connect, InitAdmin)
  - `backend/main.go:110` - Graceful shutdown code (add logger.Sync() before this)

  **Acceptance Criteria**:
  - [ ] Logger initialized with `middleware.InitLogger()` in main()
  - [ ] `defer logger.Sync()` added before main() exits
  - [ ] No compilation errors: `go build ./...`
  - [ ] Logger variable available for middleware registration
  - [ ] Existing code not broken (session, auth, CORS still work)

  **QA Scenarios**:
  ```
  Scenario: Verify logger initialization in main.go
    Tool: Bash (grep)
    Preconditions: main.go updated
    Steps:
      1. cd backend
      2. grep "InitLogger()" main.go
      3. grep "logger.Sync()" main.go
    Expected Result: Both InitLogger() and logger.Sync() calls present
    Failure Indicators: InitLogger() missing, logger.Sync() missing
    Evidence: .sisyphus/evidence/task-14-logger-init-in-main.txt

  Scenario: Verify application still compiles
    Tool: Bash (go build)
    Preconditions: main.go updated
    Steps:
      1. cd backend
      2. go build ./...
    Expected Result: No compilation errors
    Failure Indicators: Compilation errors, missing imports
    Evidence: .sisyphus/evidence/task-14-main-compiles.txt

  Scenario: Verify application still starts
    Tool: Bash (go run)
    Preconditions: main.go updated
    Steps:
      1. cd backend
      2. timeout 5 go run main.go > /tmp/startup.log 2>&1 || true
      3. grep -i "服务器启动" /tmp/startup.log || grep -i "server.*start" /tmp/startup.log
    Expected Result: Application starts without errors
    Failure Indicators: Application fails to start, crashes on logger init
    Evidence: .sisyphus/evidence/task-14-app-starts.txt
  ```

  **Commit**: YES
  - Message: `chore(main): initialize zap logger`
  - Files: `backend/main.go`
  - Pre-commit: `go build ./...`

- [ ] 15. **Update main.go - Register Middleware in Correct Order**

  **What to do**:
  - Register `GinLoggerAndMetrics(logger)` middleware in main.go
  - Place middleware registration AFTER CORS and BEFORE session middleware
  - Use `r.Use(middleware.GinLoggerAndMetrics(logger))`
  - Ensure middleware runs in correct order: CORS → Metrics → Session → Auth
  - Do NOT remove duplicate gin.Logger()/gin.Recovery() calls (document but don't touch)

  **Must NOT do**:
  - Do NOT modify existing middleware (session, auth, CORS)
  - Do NOT remove duplicate gin.Logger()/gin.Recovery() calls at lines 66-67
  - Do NOT change middleware order beyond what's specified
  - Do NOT remove gin.Default() and replace with gin.New()

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Middleware registration following existing patterns
  - **Skills**: [] (no special skills needed)

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 14, 16-17, 18-20)
  - **Blocks**: Tasks 16-17 (registration must complete before adding endpoints)
  - **Blocked By**: Task 8-10 (middleware implementation), Task 14 (logger initialized)

  **References**:
  - `backend/middleware/logging_and_metrics.go` - GinLoggerAndMetrics() function
  - `backend/main.go:63-69` - Current middleware registration order (CORS, gin.Logger, gin.Recovery, session)

  **Acceptance Criteria**:
  - [ ] `r.Use(middleware.GinLoggerAndMetrics(logger))` added to main.go
  - [ ] Middleware registered AFTER CORS, BEFORE session
  - [ ] No compilation errors
  - [ ] Application starts successfully
  - [ ] Middleware is active (logs are written on requests)

  **QA Scenarios**:
  ```
  Scenario: Verify middleware registration in main.go
    Tool: Bash (grep)
    Preconditions: main.go updated
    Steps:
      1. cd backend
      2. grep -A 2 -B 2 "GinLoggerAndMetrics" main.go
    Expected Result: Middleware registration call present
    Failure Indicators: GinLoggerAndMetrics not found, registration missing
    Evidence: .sisyphus/evidence/task-15-middleware-registration.txt

  Scenario: Verify middleware order is correct
    Tool: Bash (grep -n)
    Preconditions: main.go updated
    Steps:
      1. cd backend
      2. grep -n "cors.New\|GinLoggerAndMetrics\|InitSession" main.go | sort -t: -k1 -n
    Expected Result: Order: cors.New → GinLoggerAndMetrics → InitSession
    Failure Indicators: Wrong order, middleware not registered
    Evidence: .sisyphus/evidence/task-15-middleware-order.txt

  Scenario: Verify middleware is active (logs appear)
    Tool: Bash (curl)
    Preconditions: Application started with middleware registered
    Steps:
      1. cd backend
      2. go run main.go > /tmp/middleware-test.log 2>&1 &
      3. sleep 2
      4. curl -s http://localhost:8080/ping > /dev/null 2>&1
      5. sleep 1
      6. pkill -f "go run main.go"
      7. grep -E '"latency":|"level":"info"' /tmp/middleware-test.log | tail -1
    Expected Result: Log entry with latency and info level
    Failure Indicators: No log from middleware, middleware not active
    Evidence: .sisyphus/evidence/task-15-middleware-active.txt
  ```

  **Commit**: YES
  - Message: `chore(main): register logging and metrics middleware`
  - Files: `backend/main.go`
  - Pre-commit: `go build ./...`

- [ ] 16. **Update main.go - Add /metrics Endpoint**

  **What to do**:
  - Add `/metrics` endpoint in main.go after middleware registration
  - Use `r.GET("/metrics", gin.WrapH(promhttp.Handler()))`
  - Import `promhttp` from `github.com/prometheus/client_golang/prometheus/promhttp`
  - Ensure endpoint is accessible and returns Prometheus metrics
  - Place endpoint before or after existing routes (doesn't matter, but keep code organized)

  **Must NOT do**:
  - Do NOT add any other endpoints
  - Do NOT modify existing route definitions
  - Do NOT add authentication to /metrics endpoint (keep public)
  - Do NOT use custom handler (use gin.WrapH(promhttp.Handler()))

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Standard Prometheus metrics endpoint registration
  - **Skills**: [] (no special skills needed)

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 14-15, 17, 18-20)
  - **Blocks**: None (independent endpoint)
  - **Blocked By**: Task 15 (middleware registered first, but endpoint can exist even without middleware)

  **References**:
  - `backend/main.go:73` - Similar endpoint registration pattern (Swagger)
  - `https://pkg.go.dev/github.com/prometheus/client_golang/prometheus/promhttp#Handler` - promhttp.Handler() documentation

  **Acceptance Criteria**:
  - [ ] `r.GET("/metrics", gin.WrapH(promhttp.Handler()))` added to main.go
  - [ ] `promhttp` imported
  - [ ] No compilation errors
  - [ ] /metrics endpoint returns Prometheus metrics text format
  - [ ] Metrics include our two metrics (library_http_request_count_total, library_http_request_duration_seconds)

  **QA Scenarios**:
  ```
  Scenario: Verify /metrics endpoint exists
    Tool: Bash (grep)
    Preconditions: main.go updated
    Steps:
      1. cd backend
      2. grep "/metrics" main.go
    Expected Result: /metrics endpoint registration present
    Failure Indicators: /metrics not found, registration missing
    Evidence: .sisyphus/evidence/task-16-metrics-endpoint.txt

  Scenario: Verify /metrics endpoint is accessible
    Tool: Bash (curl)
    Preconditions: Application started
    Steps:
      1. cd backend
      2. go run main.go > /tmp/metrics-test.log 2>&1 &
      3. sleep 2
      4. curl -s http://localhost:8080/metrics | head -20
      5. pkill -f "go run main.go"
    Expected Result: Prometheus text format output with metrics
    Failure Indicators: 404 not found, error response, no metrics output
    Evidence: .sisyphus/evidence/task-16-metrics-accessible.txt

  Scenario: Verify our metrics are included in /metrics output
    Tool: Bash (curl)
    Preconditions: Application started, metrics endpoint available
    Steps:
      1. cd backend
      2. go run main.go > /tmp/metrics-output.log 2>&1 &
      3. sleep 2
      4. curl -s http://localhost:8080/metrics | grep "library_http_request_"
      5. pkill -f "go run main.go"
    Expected Result: Output contains library_http_request_count_total and library_http_request_duration_seconds
    Failure Indicators: Our metrics not found, only default Prometheus metrics
    Evidence: .sisyphus/evidence/task-16-metrics-included.txt
  ```

  **Commit**: YES
  - Message: `feat(main): add /metrics endpoint`
  - Files: `backend/main.go`
  - Pre-commit: `go build ./...`

- [ ] 17. **Update main.go - Add /ping Test Route**

  **What to do**:
  - Add `/ping` endpoint in main.go for testing and verification
  - Return JSON response: `{"message": "pong"}`
  - Use Gin's `c.JSON()` method
  - Place endpoint near other route definitions
  - No authentication required (public endpoint)

  **Must NOT do**:
  - Do NOT add any other test routes
  - Do NOT modify existing routes
  - Do NOT add authentication to /ping
  - Do NOT return different response format

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Simple test endpoint
  - **Skills**: [] (no special skills needed)

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 14-16, 18-20)
  - **Blocks**: Task 19-20 (integration tests need /ping endpoint)
  - **Blocked By**: None (can be added anytime)

  **References**:
  - `backend/main.go:75-76` - Similar route registration pattern (RegisterUserRoutes, RegisterBookRouters)
  - `https://pkg.go.dev/github.com/gin-gonic/gin#Context.JSON` - JSON() method documentation

  **Acceptance Criteria**:
  - [ ] `r.GET("/ping", ...)` added to main.go
  - [ ] Returns `{"message": "pong"}`
  - [ ] No authentication required
  - [ ] No compilation errors
  - [ ] Endpoint is accessible and returns correct JSON

  **QA Scenarios**:
  ```
  Scenario: Verify /ping endpoint exists
    Tool: Bash (grep)
    Preconditions: main.go updated
    Steps:
      1. cd backend
      2. grep "/ping" main.go
    Expected Result: /ping endpoint registration present
    Failure Indicators: /ping not found, registration missing
    Evidence: .sisyphus/evidence/task-17-ping-endpoint.txt

  Scenario: Verify /ping returns correct JSON
    Tool: Bash (curl + jq)
    Preconditions: Application started
    Steps:
      1. cd backend
      2. go run main.go > /tmp/ping-test.log 2>&1 &
      3. sleep 2
      4. curl -s http://localhost:8080/ping | jq .
      5. pkill -f "go run main.go"
    Expected Result: Output: `{"message": "pong"}`
    Failure Indicators: Wrong response format, error response
    Evidence: .sisyphus/evidence/task-17-ping-json.txt

  Scenario: Verify /ping is public (no auth required)
    Tool: Bash (curl)
    Preconditions: Application started
    Steps:
      1. cd backend
      2. go run main.go > /tmp/ping-public.log 2>&1 &
      3. sleep 2
      4. STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/ping)
      5. pkill -f "go run main.go"
      6. echo "Status code: $STATUS"
    Expected Result: Status code 200 (not 401 or 403)
    Failure Indicators: 401 Unauthorized, 403 Forbidden, auth required
    Evidence: .sisyphus/evidence/task-17-ping-public.txt
  ```

  **Commit**: YES
  - Message: `feat(main): add /ping test endpoint`
  - Files: `backend/main.go`
  - Pre-commit: `go build ./...`

- [ ] 18. **TDD REFACTOR: Code Review and Cleanup**

  **What to do**:
  - Review all implementation files for code quality issues
  - Remove any temporary/debugging code (e.g., TODO comments, console.log, test prints)
  - Ensure all comments are clear and helpful (not verbose, not redundant)
  - Check for code duplication and refactor if needed
  - Ensure all exports are intentional (no unnecessary capitalization)
  - Verify all imports are used and organized
  - Run all tests to ensure nothing broke during refactor

  **Must NOT do**:
  - Do NOT change API or behavior
  - Do NOT remove or modify tests
  - Do NOT add new functionality
  - Do NOT break existing tests

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Code cleanup and review after implementation
  - **Skills**: [] (no special skills needed)

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 14-17, 19-20)
  - **Blocks**: None (independent cleanup task)
  - **Blocked By**: Tasks 8-17 (all implementation complete)

  **References**:
  - `backend/middleware/*.go` - All implementation files
  - `backend/main.go` - Main file with integration

  **Acceptance Criteria**:
  - [ ] No temporary/debugging code remains
  - [ ] All comments are clear and necessary
  - [ ] No code duplication
  - [ ] All imports are used and organized
  - [ ] All tests still pass: `bun test ./middleware/...`
  - [ ] No compilation errors or warnings

  **QA Scenarios**:
  ```
  Scenario: Verify no temporary code remains
    Tool: Bash (grep)
    Preconditions: Refactor complete
    Steps:
      1. cd backend/middleware
      2. grep -r "TODO\|FIXME\|XXX\|HACK\|debug\|Debug" *.go
      3. grep -r "fmt\.Print\|log\.Print" *.go | grep -v "test"
    Expected Result: No temporary/debugging code found
    Failure Indicators: TODO/FIXME comments, debug print statements
    Evidence: .sisyphus/evidence/task-18-no-temp-code.txt

  Scenario: Verify all tests still pass after refactor
    Tool: Bash (go test)
    Preconditions: Refactor complete
    Steps:
      1. cd backend
      2. bun test ./middleware/... -v
    Expected Result: All tests PASS (100% pass rate)
    Failure Indicators: Any test FAIL, tests broken by refactor
    Evidence: .sisyphus/evidence/task-18-tests-pass.txt

  Scenario: Verify no unused imports
    Tool: Bash (go vet)
    Preconditions: Refactor complete
    Steps:
      1. cd backend
      2. go vet ./middleware/...
    Expected Result: No unused import warnings
    Failure Indicators: Unused imports, go vet warnings
    Evidence: .sisyphus/evidence/task-18-no-unused-imports.txt
  ```

  **Commit**: YES
  - Message: `refactor(middleware): code cleanup and review`
  - Files: `backend/middleware/*.go`, `backend/main.go`
  - Pre-commit: `go test ./middleware/... && go vet ./middleware/...`

- [ ] 19. **Integration Test - Verify Metrics Endpoint Works End-to-End**

  **What to do**:
  - Create integration test file `backend/middleware/integration_test.go`
  - Test: Start server, send requests to /ping, verify /metrics shows increment
  - Test: Verify CounterVec increments for each request
  - Test: Verify HistogramVec records duration for each request
  - Test: Verify labels are correct (method, path, status)
  - Test: Verify multiple requests increment counter correctly
  - Run integration tests with server startup/shutdown

  **Must NOT do**:
  - Do NOT modify unit tests (this is for integration testing only)
  - Do NOT test features outside scope (e.g., database, sessions)
  - Do NOT add external dependencies (use Go's built-in http client)

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Full-stack integration test with server startup/shutdown
  - **Skills**: [] (no special skills needed)

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 14-18, 20)
  - **Blocks**: None (independent integration test)
  - **Blocked By**: Tasks 14-17 (all main.go integration complete)

  **References**:
  - `backend/main.go` - Application entry point
  - `https://pkg.go.dev/net/http` - HTTP client for integration tests

  **Acceptance Criteria**:
  - [ ] Integration test file created
  - [ ] Test for counter increment exists
  - [ ] Test for histogram duration exists
  - [ ] Test for label correctness exists
  - [ ] Test for multiple requests exists
  - [ ] All integration tests PASS: `bun test ./middleware/... -run "Integration"`

  **QA Scenarios**:
  ```
  Scenario: Verify integration tests pass
    Tool: Bash (go test)
    Preconditions: Integration tests written
    Steps:
      1. cd backend
      2. bun test ./middleware/... -run "Integration" -v
    Expected Result: All integration tests PASS
    Failure Indicators: Any test FAIL, server startup errors
    Evidence: .sisyphus/evidence/task-19-integration-tests-pass.txt

  Scenario: Verify end-to-end metrics flow
    Tool: Bash (curl)
    Preconditions: Application started
    Steps:
      1. cd backend
      2. go run main.go > /tmp/e2e-test.log 2>&1 &
      3. sleep 2
      4. # Send 5 requests to /ping
      5. for i in {1..5}; do curl -s http://localhost:8080/ping > /dev/null; done
      6. # Check counter
      7. COUNT=$(curl -s http://localhost:8080/metrics | grep 'library_http_request_count_total{method="GET",path="/ping",status="200"}' | awk '{print $2}')
      8. # Check histogram sum
      9. DURATION=$(curl -s http://localhost:8080/metrics | grep 'library_http_request_duration_seconds_sum{method="GET",path="/ping"}' | awk '{print $2}')
      10. pkill -f "go run main.go"
      11. echo "Counter: $COUNT, Duration sum: $DURATION"
    Expected Result: Counter = 5, Duration sum > 0
    Failure Indicators: Counter wrong, duration not recorded
    Evidence: .sisyphus/evidence/task-19-e2e-metrics.txt
  ```

  **Commit**: YES
  - Message: `test(middleware): add integration tests`
  - Files: `backend/middleware/integration_test.go`
  - Pre-commit: `go test -run "Integration" ./middleware/...`

- [ ] 20. **Integration Test - Verify Logging for Different Status Codes**

  **What to do**:
  - Add tests to `backend/middleware/integration_test.go` for logging verification
  - Test: 200 status logs at info level (use /ping)
  - Test: 400 status logs at warn level (send invalid request)
  - Test: 401 status logs at warn level (auth failure)
  - Test: 500 status logs at error level (need endpoint that returns 500 or mock)
  - Verify log messages are in Chinese (matching existing codebase)
  - Verify log fields are correct (method, path, status, latency, ip)

  **Must NOT do**:
  - Do NOT modify unit tests
  - Do NOT test other status codes not in spec (only 200, 400-499, 500+)
  - Do NOT verify English log messages (should be Chinese)

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Integration testing of logging with various status codes
  - **Skills**: [] (no special skills needed)

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 14-19)
  - **Blocks**: None (independent integration test)
  - **Blocked By**: Task 10 (logging implementation), Task 14-17 (main.go integration)

  **References**:
  - `backend/middleware/logging_and_metrics.go` - Middleware logging logic
  - `backend/main.go` - Application endpoints

  **Acceptance Criteria**:
  - [ ] Test for 200 status info logging exists
  - [ ] Test for 400 status warn logging exists
  - [ ] Test for 500 status error logging exists
  - [ ] Tests verify Chinese log messages
  - [ ] Tests verify correct log fields
  - [ ] All logging integration tests PASS: `bun test ./middleware/... -run "Integration"`

  **QA Scenarios**:
  ```
  Scenario: Verify logging integration tests pass
    Tool: Bash (go test)
    Preconditions: Integration tests written
    Steps:
      1. cd backend
      2. bun test ./middleware/... -run "Integration" -v
    Expected Result: All logging integration tests PASS
    Failure Indicators: Any test FAIL, wrong log levels
    Evidence: .sisyphus/evidence/task-20-logging-integration-pass.txt

  Scenario: Verify 200 status logs at info level
    Tool: Bash (curl + grep)
    Preconditions: Application started
    Steps:
      1. cd backend
      2. go run main.go > /tmp/log-200.log 2>&1 &
      3. sleep 2
      4. curl -s http://localhost:8080/ping > /dev/null
      5. sleep 1
      6. pkill -f "go run main.go"
      7. grep -E '"level":"info".*"status":200' /tmp/log-200.log | tail -1
    Expected Result: Log entry with level:"info" and status:200
    Failure Indicators: Wrong log level, no log entry
    Evidence: .sisyphus/evidence/task-20-200-info-log.txt

  Scenario: Verify 400 status logs at warn level
    Tool: Bash (curl + grep)
    Preconditions: Application started
    Steps:
      1. cd backend
      2. go run main.go > /tmp/log-400.log 2>&1 &
      3. sleep 2
      4. # Send invalid request that returns 400
      5. curl -s -X POST http://localhost:8080/api/books -d '{}' -H "Content-Type: application/json" > /dev/null 2>&1
      6. sleep 1
      7. pkill -f "go run main.go"
      8. grep -E '"level":"warn".*"status":4[0-9][0-9]' /tmp/log-400.log | tail -1
    Expected Result: Log entry with level:"warn" and status:4XX
    Failure Indicators: Wrong log level, no log entry
    Evidence: .sisyphus/evidence/task-20-400-warn-log.txt
  ```

  **Commit**: YES
  - Message: `test(middleware): add logging integration tests`
  - Files: `backend/middleware/integration_test.go`
  - Pre-commit: `go test -run "Integration" ./middleware/...`

---



---

## Final Verification Wave (MANDATORY — after ALL implementation tasks)

> 4 review agents run in PARALLEL. ALL must APPROVE. Rejection → fix → re-run.

- [ ] F1. **Plan Compliance Audit** — `oracle`
  Read the plan end-to-end. For each "Must Have": verify implementation exists (read file, curl endpoint, run command). For each "Must NOT Have": search codebase for forbidden patterns — reject with file:line if found. Check evidence files exist in .sisyphus/evidence/. Compare deliverables against plan.
  Output: `Must Have [N/N] | Must NOT Have [N/N] | Tasks [N/N] | VERDICT: APPROVE/REJECT`

- [ ] F2. **Code Quality Review** — `unspecified-high`
  Run `go build ./...` + `go test ./...` + linter. Review all changed files for: empty catches, commented-out code, unused imports, API usage errors. Check for AI slop: excessive comments, over-abstraction, generic names (data/result/item/temp).
  Output: `Build [PASS/FAIL] | Tests [N pass/N fail] | Lint [PASS/FAIL] | Files [N clean/N issues] | VERDICT`

- [ ] F3. **Real Manual QA** — `unspecified-high`
  Start from clean state. Execute EVERY QA scenario from EVERY task — follow exact steps, capture evidence. Test cross-task integration (middleware + metrics + logging working together). Test edge cases: 404 routes, canceled contexts, auth aborts, static file requests. Save to `.sisyphus/evidence/final-qa/`.
  Output: `Scenarios [N/N pass] | Integration [N/N] | Edge Cases [N tested] | VERDICT`

- [ ] F4. **Scope Fidelity Check** — `deep`
  For each task: read "What to do", read actual diff (git log/diff). Verify 1:1 — everything in spec was built (no missing), nothing beyond spec was built (no creep). Check "Must NOT do" compliance. Detect cross-task contamination: Task N touching Task M's files. Flag unaccounted changes.
  Output: `Tasks [N/N compliant] | Contamination [CLEAN/N issues] | Unaccounted [CLEAN/N files] | VERDICT`

---

## Commit Strategy

- **1**: `chore(deps): add zap and prometheus dependencies` — backend/go.mod, go mod tidy
- **7**: `feat(middleware): add prometheus metrics definitions` — backend/middleware/prometheus.go
- **8**: `feat(middleware): implement core middleware logic` — backend/middleware/logging_and_metrics.go
- **14**: `chore(main): integrate logger and metrics middleware` — backend/main.go
- **17**: `feat(main): add /metrics and /ping endpoints` — backend/main.go
- **19**: `test: add integration tests for middleware` — backend/middleware/*_test.go

---

## Success Criteria

### Verification Commands
```bash
# Dependencies installed
cd backend && go mod tidy

# Build succeeds
go build ./...

# All tests pass
bun test ./middleware/... -v

# Start server and verify metrics endpoint
curl -s http://localhost:8080/metrics | grep "library_http_request_count_total{method=\"GET\",path=\"/ping\",status=\"200\"}"
# Expected: Output shows counter > 0

# Ping endpoint works
curl -s http://localhost:8080/ping | jq .
# Expected: {"message":"pong"}

# Existing API still works
curl -s http://localhost:8080/api/books | jq .
# Expected: Books array returned (or auth error if required)
```

### Final Checklist
- [ ] All "Must Have" present
- [ ] All "Must NOT Have" absent
- [ ] All tests pass (100% pass rate)
- [ ] /metrics endpoint returns Prometheus format
- [ ] Middleware logs at correct levels (error/warn/info)
- [ ] Path labels use normalized route patterns
- [ ] Existing functionality not broken
