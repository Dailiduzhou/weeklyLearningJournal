# 微服务部署配置
# 需要先更新各服务配置文件中的 etcd 地址为 k8s 内部地址

# 1. 创建命名空间
kubectl apply -f namespace.yaml

# 2. 部署 etcd
kubectl apply -f etcd.yaml

# 3. 等待 etcd 就绪
kubectl wait --for=condition=ready pod -l app=etcd -n micro-svc --timeout=120s

# 4. 部署服务 B
kubectl apply -f ../svc-b/svc-b-deploy.yaml

# 5. 部署服务 A
kubectl apply -f ../svc-a/svc-a-deploy.yaml

# 6. 部署网关
kubectl apply -f ../gateway/gateway-deploy.yaml

echo "所有服务已部署完成！"
echo "查看服务状态: kubectl get all -n micro-svc"
