# 本地测试指南 (Docker Compose)

使用 Docker Compose 一键启动所有服务进行本地测试，无需安装 etcd 或处理端口冲突。

## 📁 配置文件说明

```
.
├── docker-compose.yml          # 编排文件
├── Makefile                    # 快捷命令
├── svc-b/
│   ├── Dockerfile.dev          # 开发环境镜像
│   ├── etc/
│   │   ├── pb.yaml            # 本地配置 (127.0.0.1:2379)
│   │   └── pb-docker.yaml     # Docker 配置 (etcd:2379)
├── svc-a/
│   ├── Dockerfile.dev
│   ├── etc/
│   │   ├── pb.yaml
│   │   └── pb-docker.yaml
└── gateway/
    ├── Dockerfile.dev
    └── etc/
        ├── gateway.yaml
        └── gateway-docker.yaml
```

**三套配置的区别：**
- `*.yaml` - 本地直接运行（`go run`），连接本地 etcd (127.0.0.1:2379)
- `*-docker.yaml` - Docker Compose 环境，连接容器内 etcd (etcd:2379)
- `*-k8s.yaml` - Kubernetes 环境，连接 K8s 内 etcd (etcd-client:2379)

## 🚀 快速开始

### 1. 一键启动所有服务

```bash
cd /home/mikufan/code/weeklyLearningJournal/semester2/week6/micro_svc
make up
```

或者使用 docker-compose 命令：

```bash
docker-compose up -d
```

### 2. 测试 API

```bash
make test
```

或直接 curl：

```bash
curl http://localhost:8888/api/a/call
```

预期输出：
```json
{"msg":"Service A executed. -> Hello from svc B"}
```

### 3. 查看日志

```bash
# 所有服务日志
make logs

# 单个服务日志
make logs-gateway
make logs-svc-a
make logs-svc-b
make logs-etcd
```

### 4. 停止服务

```bash
make down
```

彻底清理（包括数据卷）：

```bash
make clean
```

## 🛠️ 常用命令

### 分步启动（便于调试）

```bash
# 1. 只启动 etcd
make etcd

# 2. 启动服务 B
make svc-b

# 3. 启动服务 A
make svc-a

# 4. 启动网关
make gateway
```

### 进入容器调试

```bash
# 进入服务 B 容器
make exec-svc-b

# 进入服务 A 容器
make exec-svc-a

# 进入网关容器
make exec-gateway

# 检查 etcd 健康状态
make exec-etcd
```

### 多次测试

```bash
# 连续调用 5 次 API
make test-loop
```

## 🔧 手动操作

### 只构建镜像

```bash
docker-compose build
```

### 重启单个服务

```bash
# 重启服务 A
docker-compose restart svc-a

# 强制重新创建并启动
docker-compose up -d --force-recreate svc-a
```

### 查看服务状态

```bash
docker-compose ps
```

## 📝 注意事项

1. **首次启动较慢**：需要下载基础镜像和构建应用镜像
2. **服务启动顺序**：etcd → svc-b → svc-a → gateway（由 depends_on 控制）
3. **etcd 数据持久化**：使用 Docker volume，重启后数据不会丢失
4. **端口占用**：确保本地 8888 和 2379 端口未被占用

## 🐛 故障排查

### 服务无法启动

```bash
# 查看详细日志
docker-compose logs <service-name>

# 检查 etcd 是否健康
docker-compose exec etcd etcdctl endpoint health
```

### 端口冲突

如果 8888 或 2379 端口被占用，修改 `docker-compose.yml`：

```yaml
gateway:
  ports:
    - "8889:8888"  # 改为 8889

etcd:
  ports:
    - "2380:2379"  # 改为 2380
```

### 配置未生效

Docker 会缓存配置文件，修改配置后需要：

```bash
docker-compose down
docker-compose up -d
```

## 🧹 清理

```bash
# 停止并删除容器
make down

# 彻底清理（包括镜像和数据卷）
make clean

# 删除所有相关镜像
docker rmi micro_svc-svc-b micro_svc-svc-a micro_svc-gateway
```
