#!/bin/bash

set -eux

# 颜色输出
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== Kaniko Rootless 模式构建镜像测试 ===${NC}"

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# 命名空间
NAMESPACE="imgbuild"

# 1. 创建命名空间
echo -e "\n${YELLOW}[1/6] 创建命名空间...${NC}"
kubectl get ns "$NAMESPACE" >/dev/null 2>&1 || kubectl create ns "$NAMESPACE"
echo -e "${GREEN}✓ 命名空间已就绪${NC}"

# 2. 清理之前的资源（如果存在）
echo -e "\n${YELLOW}[2/6] 清理之前的资源...${NC}"
kubectl -n "$NAMESPACE" delete job kaniko-addfile --ignore-not-found=true
kubectl -n "$NAMESPACE" delete configmap kaniko-context --ignore-not-found=true
kubectl -n "$NAMESPACE" delete pod verify-image --ignore-not-found=true
sleep 2
echo -e "${GREEN}✓ 清理完成${NC}"

# 3. 创建 ConfigMap
echo -e "\n${YELLOW}[3/6] 创建构建上下文 ConfigMap...${NC}"
kubectl apply -f kaniko-context.yaml
echo -e "${GREEN}✓ ConfigMap 创建成功${NC}"

# 4. 创建并运行 Kaniko Job
echo -e "\n${YELLOW}[4/6] 创建并运行 Kaniko Job...${NC}"
kubectl apply -f kaniko-job.yaml

# 等待 Job 完成
echo -e "${YELLOW}等待 Job 完成（最多 5 分钟）...${NC}"
if kubectl -n "$NAMESPACE" wait --for=condition=complete job/kaniko-addfile --timeout=5m 2>/dev/null; then
    echo -e "${GREEN}✓ Job 执行成功${NC}"
else
    echo -e "${RED}✗ Job 执行失败或超时${NC}"
    echo -e "\n${YELLOW}查看 Job 日志:${NC}"
    kubectl -n "$NAMESPACE" logs job/kaniko-addfile || true
    echo -e "\n${YELLOW}查看 Job 状态:${NC}"
    kubectl -n "$NAMESPACE" get job kaniko-addfile
    exit 1
fi

# 5. 查看构建日志
echo -e "\n${YELLOW}[5/6] 查看构建日志...${NC}"
kubectl -n "$NAMESPACE" logs job/kaniko-addfile | tail -20

# 6. 验证镜像内容
echo -e "\n${YELLOW}[6/6] 验证镜像内容...${NC}"
echo -e "${YELLOW}创建验证 Pod...${NC}"
kubectl -n "$NAMESPACE" run verify-image \
  --image=localhost:5000/new-kaniko-image:latest \
  --restart=Never \
  --command -- sh -c 'cat /opt/app/hello.txt && echo "" && echo "OK"' || {
    echo -e "${RED}✗ 创建验证 Pod 失败${NC}"
    exit 1
}

# 等待 Pod 运行
echo -e "${YELLOW}等待 Pod 运行...${NC}"
sleep 5
kubectl -n "$NAMESPACE" wait --for=condition=Ready pod/verify-image --timeout=30s 2>/dev/null || {
    echo -e "${YELLOW}Pod 可能已完成，查看日志...${NC}"
}

# 查看验证结果
echo -e "\n${GREEN}验证结果:${NC}"
kubectl -n "$NAMESPACE" logs verify-image

# 检查是否包含预期内容
if kubectl -n "$NAMESPACE" logs verify-image 2>/dev/null | grep -q "built-by: kaniko"; then
    echo -e "\n${GREEN}✓✓✓ 验证成功！镜像内容正确！${NC}"
    VERIFY_SUCCESS=true
else
    echo -e "\n${RED}✗ 验证失败：未找到预期内容${NC}"
    VERIFY_SUCCESS=false
fi

# 清理验证 Pod
kubectl -n "$NAMESPACE" delete pod verify-image --ignore-not-found=true

# 总结
echo -e "\n${GREEN}=== 测试总结 ===${NC}"
echo -e "命名空间: ${NAMESPACE}"
echo -e "构建镜像: registry.kube-system.svc.cluster.local:5000/new-kaniko-image:latest"
echo -e "验证镜像: localhost:5000/new-kaniko-image:latest"
if [ "$VERIFY_SUCCESS" = true ]; then
    echo -e "${GREEN}状态: ✓ 测试通过${NC}"
    exit 0
else
    echo -e "${RED}状态: ✗ 测试失败${NC}"
    exit 1
fi

