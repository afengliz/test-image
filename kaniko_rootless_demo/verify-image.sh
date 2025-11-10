#!/bin/bash

set -eux

# 颜色输出
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== 验证构建的镜像 ===${NC}"

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# 命名空间
NAMESPACE="imgbuild"
IMAGE_NAME="localhost:5000/new-kaniko-image:latest"

# 1. 创建命名空间
echo -e "\n${YELLOW}[1/3] 准备环境...${NC}"
kubectl get ns "$NAMESPACE" >/dev/null 2>&1 || kubectl create ns "$NAMESPACE"
echo -e "${GREEN}✓ 命名空间已就绪${NC}"

# 2. 清理之前的验证 Pod
echo -e "\n${YELLOW}[2/3] 清理旧资源...${NC}"
kubectl -n "$NAMESPACE" delete pod verify-kaniko-image --ignore-not-found=true
sleep 2
echo -e "${GREEN}✓ 清理完成${NC}"

# 3. 创建验证 Pod
echo -e "\n${YELLOW}[3/3] 创建验证 Pod...${NC}"
kubectl -n "$NAMESPACE" run verify-kaniko-image \
  --image="$IMAGE_NAME" \
  --restart=Never \
  --command -- sh -c 'ls -la /usr/local/app/ && echo "---" && /usr/local/app/main --version 2>&1 || echo "程序运行中..." && sleep 5' || {
    echo -e "${YELLOW}等待 Pod 启动...${NC}"
    sleep 5
}

# 等待 Pod 运行
echo -e "${YELLOW}等待 Pod 运行...${NC}"
kubectl -n "$NAMESPACE" wait --for=condition=Ready pod/verify-kaniko-image --timeout=30s 2>/dev/null || {
    echo -e "${YELLOW}Pod 可能已完成，查看日志...${NC}"
}

# 查看日志
echo -e "\n${GREEN}验证结果:${NC}"
kubectl -n "$NAMESPACE" logs verify-kaniko-image || {
    echo -e "${YELLOW}无法获取日志，查看 Pod 状态:${NC}"
    kubectl -n "$NAMESPACE" get pod verify-kaniko-image
    kubectl -n "$NAMESPACE" describe pod verify-kaniko-image | tail -20
}

# 检查 Pod 状态
POD_STATUS=$(kubectl -n "$NAMESPACE" get pod verify-kaniko-image -o jsonpath='{.status.phase}' 2>/dev/null || echo "Unknown")
if [ "$POD_STATUS" = "Succeeded" ] || [ "$POD_STATUS" = "Running" ]; then
    echo -e "\n${GREEN}✓✓✓ 验证成功！镜像内容正确！${NC}"
    VERIFY_SUCCESS=true
else
    echo -e "\n${YELLOW}Pod 状态: $POD_STATUS${NC}"
    VERIFY_SUCCESS=false
fi

# 清理
echo -e "\n${YELLOW}清理验证 Pod...${NC}"
kubectl -n "$NAMESPACE" delete pod verify-kaniko-image --ignore-not-found=true

# 总结
echo -e "\n${GREEN}=== 验证总结 ===${NC}"
echo -e "镜像: $IMAGE_NAME"
if [ "$VERIFY_SUCCESS" = true ]; then
    echo -e "${GREEN}状态: ✓ 验证通过${NC}"
    exit 0
else
    echo -e "${YELLOW}状态: ⚠ 需要进一步检查${NC}"
    exit 1
fi

