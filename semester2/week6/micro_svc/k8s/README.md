# Kubernetes 部署指南

## 目录结构

```
k8s/
├── namespace.yaml      # 命名空间
├── etcd.yaml           # etcd 服务发现和配置中心
├── deploy.sh           # 一键部署脚本
svc-b/
├── Dockerfile          # 服务 B 容器镜像
├── svc-b-deploy.yaml   # 服务 B K8s 部署配置
├── etc/pb.yaml         # 本地环境配置
└── etc/pb-k8s.yaml     # K8s 环境配置
svc-a/
├── Dockerfile          # 服务 A 容器镜像
├── svc-a-deploy.yaml   # 服务 A K8s 部署配置
├── etc/pb.yaml         # 本地环境配置
└── etc/pb-k8s.yaml     # K8s 环境配置
gateway/
├── Dockerfile          # 网关容器镜像
├── gateway-deploy.yaml # 网关 K8s 部署配置
├── etc/gateway.yaml    # 本地环境配置
└── etc/gateway-k8s.yaml # K8s 环境配置
```

## 构建镜像

```bash
# 构建服务 B 镜像
cd svc-b
docker build -t svc-b:latest .

# 构建服务 A 镜像
cd svc-a
docker build -t svc-a:latest .

# 构建网关镜像
cd gateway
docker build -t gateway:latest .
```

## 部署到 Kubernetes

### 方式一：使用部署脚本

```bash
cd k8s
chmod +x deploy.sh
./deploy.sh
```

### 方式二：手动部署

```bash
# 1. 创建命名空间
kubectl apply -f k8s/namespace.yaml

# 2. 部署 etcd
kubectl apply -f k8s/etcd.yaml

# 3. 等待 etcd 就绪
kubectl wait --for=condition=ready pod -l app=etcd -n micro-svc --timeout=120s

# 4. 部署服务 B
kubectl apply -f svc-b/svc-b-deploy.yaml

# 5. 部署服务 A
kubectl apply -f svc-a/svc-a-deploy.yaml

# 6. 部署网关
kubectl apply -f gateway/gateway-deploy.yaml
```

## 配置说明

### 本地开发 vs K8s 部署

各服务提供了两套配置文件：

- `etc/*.yaml` - 本地开发环境配置，连接本地 etcd (127.0.0.1:2379)
- `etc/*-k8s.yaml` - K8s 环境配置，连接 K8s 内部 etcd (etcd-client:2379)

**注意**：Dockerfile 中复制的是默认配置文件。要在 K8s 中使用 K8s 配置，需要通过 ConfigMap 挂载。

### 创建 ConfigMap

```bash
# 为服务 B 创建 ConfigMap
kubectl create configmap svc-b-config --from-file=svc-b/etc/pb-k8s.yaml -n micro-svc

# 为服务 A 创建 ConfigMap
kubectl create configmap svc-a-config --from-file=svc-a/etc/pb-k8s.yaml -n micro-svc

# 为网关创建 ConfigMap
kubectl create configmap gateway-config --from-file=gateway/etc/gateway-k8s.yaml -n micro-svc
```

### 更新 Deployment 使用 ConfigMap

需要修改各服务的 Deployment YAML，添加 volume 挂载：

```yaml
spec:
  template:
    spec:
      containers:
      - name: svc-b
        volumeMounts:
        - name: config
          mountPath: /app/etc
      volumes:
      - name: config
        configMap:
          name: svc-b-config
```

## 查看服务状态

```bash
# 查看所有资源
kubectl get all -n micro-svc

# 查看 Pod 日志
kubectl logs -f deployment/svc-b -n micro-svc
kubectl logs -f deployment/svc-a -n micro-svc
kubectl logs -f deployment/gateway -n micro-svc

# 查看 etcd 状态
kubectl logs -f statefulset/etcd -n micro-svc
```

## 访问服务

```bash
# 端口转发网关到本地
kubectl port-forward svc/gateway-svc 8888:8888 -n micro-svc

# 测试 API
curl http://localhost:8888/api/a/call
```

## 扩容

```bash
# 手动扩容
kubectl scale deployment svc-b --replicas=5 -n micro-svc

# HPA 自动扩容（已配置）
kubectl get hpa -n micro-svc
```

## 清理

```bash
kubectl delete namespace micro-svc
```
