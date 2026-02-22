# gRPC 项目完善：添加 main 函数和 Makefile

## TL;DR

> **Quick Summary**: 为 gRPC stream 项目添加完整的 main 函数（server 和 client），并创建 Makefile 便于构建和测试。
> 
> **Deliverables**:
> - server/main.go - 完整的 main 函数，启动 gRPC 服务器
> - client/main.go - 完整的 main 函数，测试两种流式调用
> - Makefile - 构建、运行、测试的便捷命令
> 
> **Estimated Effort**: Quick
> **Parallel Execution**: NO - sequential (3 tasks)
> **Critical Path**: Task 1 → Task 2 → Task 3

---

## Context

### Original Request
给 client and server 添加 main 函数，新增 makefile（或 sh）便于测试

### Interview Summary
**Key Discussions**:
- 项目已有 proto 定义和 service 实现逻辑
- server 缺少 main 函数来启动 gRPC 服务
- client 缺少 main 函数来调用服务
- 需要 Makefile 简化构建和测试流程

**Research Findings**:
- 项目使用 gRPC v1.79.1
- 定义了两种流：服务端流 (GetStockUpdates) 和 双向流 (Chat)
- 包名：stream_grpc

---

## Work Objectives

### Core Objective
完善 gRPC 项目的可运行性，添加入口函数和构建脚本。

### Concrete Deliverables
- server/main.go - 可运行的 gRPC 服务器（监听端口 50051）
- client/main.go - 可运行的客户端，测试两种流式调用
- Makefile - 包含 build, run-server, run-client, test, proto 命令

### Definition of Done
- [ ] `make build` 成功编译 server 和 client
- [ ] `make run-server` 启动服务器
- [ ] `make run-client` 客户端能连接并测试两种流
- [ ] `make proto` 能重新生成 proto 文件（如果需要）

### Must Have
- 服务器监听 localhost:50051
- 客户端先测试 GetStockUpdates（服务端流），再测试 Chat（双向流）
- Makefile 命令简洁易用

### Must NOT Have (Guardrails)
- 不修改现有的 proto 定义
- 不修改 pb/proto/ 下自动生成的文件
- 不修改 server/main.go 中已有的 service 实现逻辑

---

## Verification Strategy

> **ZERO HUMAN INTERVENTION** — ALL verification is agent-executed.

### Test Decision
- **Infrastructure exists**: NO
- **Automated tests**: None (手动验证通过 Makefile 运行)
- **Framework**: N/A
- **Agent-Executed QA**: ALWAYS (mandatory for all tasks)

### QA Policy
每个任务必须包含 agent 执行的 QA 场景：
- **构建验证**: 运行 `make build` 检查编译错误
- **运行验证**: 通过 tmux 启动 server，然后运行 client
- **功能验证**: 检查日志输出确认流式通信正常

---

## Execution Strategy

### Sequential Execution (dependencies require order)

```
Task 1: 完善 server/main.go [quick]
  └── Task 2: 完善 client/main.go [quick]
        └── Task 3: 创建 Makefile [quick]
```

### Dependency Matrix
- **1**: — — 2
- **2**: 1 — 3
- **3**: 2 — Final

### Agent Dispatch Summary
- **1**: **3** — T1 → `quick`, T2 → `quick`, T3 → `quick`

---

## TODOs

- [x] 1. 完善 server/main.go - 添加 main 函数启动 gRPC 服务器

  **What to do**:
  - 导入必要的包：net, log/slog, google.golang.org/grpc
  - 创建 main 函数：
    - 创建 TCP 监听器 (localhost:50051)
    - 创建 gRPC 服务器 (grpc.NewServer())
    - 注册服务 (pb.RegisterTikerServiceServer)
    - 启动服务器 (Serve)
  - 添加启动日志和优雅关闭处理

  **Must NOT do**:
  - 不修改已有的 server 结构体和 GetStockUpdates/Chat 方法
  - 不使用硬编码的配置（考虑未来扩展）

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: 标准的 gRPC 服务器启动代码，模式固定
  - **Skills**: 无特殊技能需求
  - **Skills Evaluated but Omitted**: 无

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Sequential (第一个任务)
  - **Blocks**: Task 2
  - **Blocked By**: None

  **References**:
  - `server/main.go:12-55` - 已有的 service 实现，需要保留
  - Official docs: `https://grpc.io/docs/languages/go/basics/` - gRPC Go 服务器启动模式

  **Acceptance Criteria**:
  - [ ] server/main.go 包含完整的 main 函数
  - [ ] 服务器监听 :50051 端口
  - [ ] 有启动日志输出

  **QA Scenarios**:

  ```
  Scenario: 编译服务器代码
    Tool: Bash
    Preconditions: 在项目根目录
    Steps:
      1. 运行 cd /home/mikufan/code/weeklyLearningJournal/26winter/mini_grpc/stream
      2. 运行 go build -o bin/server ./server
    Expected Result: 编译成功，生成 bin/server 可执行文件
    Failure Indicators: 编译错误，无输出文件
    Evidence: .sisyphus/evidence/task-1-build-server.txt
  ```

  **Commit**: YES (groups with 2, 3)
  - Message: `feat: add main functions and Makefile for gRPC stream demo`
  - Files: `server/main.go, client/main.go, Makefile`

- [x] 2. 完善 client/main.go - 添加 main 函数测试两种流式调用

  **What to do**:
  - 导入必要的包：context, time, google.golang.org/grpc
  - 创建 main 函数：
    - 建立 gRPC 连接 (grpc.DialContext 到 localhost:50051)
    - 创建客户端 (pb.NewTikerServiceClient)
    - 先调用 callServerStream 测试服务端流
    - 添加短暂延迟
    - 调用 callChat 测试双向流（需要实现）
  - 实现 callChat 函数：
    - 创建双向流 (client.Chat)
    - 启动 goroutine 发送消息
    - 主 goroutine 接收消息
    - 处理结束和错误

  **Must NOT do**:
  - 不修改已有的 callServerStream 函数逻辑
  - 不使用复杂的并发控制（保持示例简洁）

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: 标准的 gRPC 客户端调用模式
  - **Skills**: 无特殊技能需求

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Sequential (依赖 Task 1)
  - **Blocks**: Task 3
  - **Blocked By**: Task 1

  **References**:
  - `client/main.go:11-32` - 已有的 callServerStream 函数
  - `server/main.go:36-54` - Chat 服务端实现，了解消息格式
  - Official docs: `https://grpc.io/docs/languages/go/basics/` - gRPC 客户端调用模式

  **Acceptance Criteria**:
  - [ ] client/main.go 包含完整的 main 函数
  - [ ] 能连接 localhost:50051
  - [ ] 实现 callChat 函数测试双向流
  - [ ] 有清晰的日志输出区分两个测试

  **QA Scenarios**:

  ```
  Scenario: 编译客户端代码
    Tool: Bash
    Preconditions: 在项目根目录
    Steps:
      1. 运行 go build -o bin/client ./client
    Expected Result: 编译成功，生成 bin/client 可执行文件
    Failure Indicators: 编译错误
    Evidence: .sisyphus/evidence/task-2-build-client.txt
  ```

  **Commit**: YES (groups with 1, 3)

- [x] 3. 创建 Makefile - 添加构建和运行命令

  **What to do**:
  - 创建 Makefile 包含以下目标：
    - `all`: 默认目标，执行 build
    - `build`: 编译 server 和 client 到 bin/ 目录
    - `run-server`: 启动服务器
    - `run-client`: 运行客户端测试
    - `test`: 先启动服务器（后台），再运行客户端，最后清理
    - `proto`: 使用 protoc 重新生成 pb 文件
    - `clean`: 清理 bin/ 目录
  - 使用 .PHONY 声明所有目标
  - 添加简洁的注释说明每个命令的用途

  **Must NOT do**:
  - 不使用复杂的 Make 语法（保持易读性）
  - 不添加与项目无关的目标

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: 标准的 Go 项目 Makefile 模板
  - **Skills**: 无

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Sequential (最后执行)
  - **Blocks**: None
  - **Blocked By**: Task 2

  **References**:
  - `go.mod` - 了解模块名和依赖
  - 项目结构 - bin/ 目录用于存放编译产物

  **Acceptance Criteria**:
  - [ ] Makefile 包含所有 6 个目标 (all, build, run-server, run-client, test, proto, clean)
  - [ ] `make build` 能成功编译
  - [ ] `make proto` 命令正确调用 protoc

  **QA Scenarios**:

  ```
  Scenario: 验证 Makefile 构建功能
    Tool: Bash
    Preconditions: 在项目根目录
    Steps:
      1. 运行 make build
      2. 检查 bin/server 和 bin/client 是否存在
    Expected Result: 两个可执行文件都生成成功
    Failure Indicators: 编译失败或文件不存在
    Evidence: .sisyphus/evidence/task-3-make-build.txt

  Scenario: 验证 proto 生成命令
    Tool: Bash
    Preconditions: 在项目根目录
    Steps:
      1. 运行 make proto
    Expected Result: protoc 命令执行（可能成功或失败取决于环境）
    Failure Indicators: Makefile 语法错误
    Evidence: .sisyphus/evidence/task-3-make-proto.txt
  ```

  **Commit**: YES (groups with 1, 2)

---

## Final Verification Wave

> 4 review agents run in PARALLEL. ALL must APPROVE.

- [ ] F1. **Plan Compliance Audit** — `oracle`
  验证所有 Must Have 已实现，Must NOT Have 未违反。

- [ ] F2. **Code Quality Review** — `unspecified-high`
  运行 `make build` 检查编译，检查代码风格。

- [ ] F3. **Real Manual QA** — `unspecified-high`
  运行 `make test` 验证端到端功能。

- [ ] F4. **Scope Fidelity Check** — `deep`
  确认没有修改 proto 定义和生成的 pb 文件。

---

## Commit Strategy

- **1**: `feat: add main functions and Makefile for gRPC stream demo` — server/main.go, client/main.go, Makefile
  - Pre-commit: `make build`

---

## Success Criteria

### Verification Commands
```bash
make build      # Expected: 编译成功，生成 bin/server 和 bin/client
make test       # Expected: 服务器启动，客户端连接，流式通信正常
```

### Final Checklist
- [ ] server 能正常启动并监听 50051 端口
- [ ] client 能连接并测试两种流
- [ ] Makefile 所有命令可用
- [ ] 没有修改 proto 定义和生成的 pb 文件
