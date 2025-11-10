#!/bin/bash
set -e

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== 程序内构建方式测试（使用 kubectl cp） ===${NC}"

# 切换到脚本所在目录
SCRIPT_DIR=$(dirname "$0")
cd "$SCRIPT_DIR"

NAMESPACE="imgbuild"
POD_NAME="kaniko-build-simple"
POD_YAML="kaniko-pod-simple.yaml"
NEW_IMAGE="registry.kube-system.svc.cluster.local:5000/new-kaniko-image:latest"

# 1. 检查必要文件
echo -e "\n${YELLOW}[1/6] 检查必要文件...${NC}"
if [ ! -f "main" ]; then
    echo -e "${RED}错误: main 文件不存在，请先运行 'make build'${NC}"
    exit 1
fi
if [ ! -f "../demo_server/main" ]; then
    echo -e "${RED}错误: ../demo_server/main 文件不存在${NC}"
    exit 1
fi
echo -e "${GREEN}✓ 文件检查通过${NC}"

# 2. 创建命名空间
echo -e "\n${YELLOW}[2/6] 创建命名空间...${NC}"
if kubectl get ns "$NAMESPACE" >/dev/null 2>&1; then
    echo -e "${GREEN}✓ 命名空间已就绪${NC}"
else
    kubectl create ns "$NAMESPACE"
    echo -e "${GREEN}✓ 命名空间创建成功${NC}"
fi

# 3. 清理之前的 Pod
echo -e "\n${YELLOW}[3/6] 清理之前的 Pod...${NC}"
kubectl -n "$NAMESPACE" delete pod "$POD_NAME" --ignore-not-found=true
sleep 2
echo -e "${GREEN}✓ 清理完成${NC}"

# 4. 创建 Pod
echo -e "\n${YELLOW}[4/6] 创建 Pod...${NC}"
kubectl apply -f "$POD_YAML"
echo -e "${YELLOW}等待 Pod 就绪...${NC}"
kubectl -n "$NAMESPACE" wait --for=condition=Ready pod/"$POD_NAME" --timeout=60s
echo -e "${GREEN}✓ Pod 已就绪${NC}"

# 5. 复制文件
echo -e "\n${YELLOW}[5/6] 复制文件到 Pod...${NC}"
echo -e "${YELLOW}复制构建程序 main (2.6MB)...${NC}"
kubectl -n "$NAMESPACE" cp main "$POD_NAME:/workspace/main"
echo -e "${GREEN}✓ main 已复制${NC}"

echo -e "${YELLOW}复制 server/main (7.8MB)...${NC}"
kubectl -n "$NAMESPACE" exec "$POD_NAME" -- mkdir -p /workspace/server
kubectl -n "$NAMESPACE" cp ../demo_server/main "$POD_NAME:/workspace/server/main"
echo -e "${GREEN}✓ server/main 已复制${NC}"

# 6. 运行构建程序
echo -e "\n${YELLOW}[6/6] 运行构建程序...${NC}"
kubectl -n "$NAMESPACE" exec "$POD_NAME" -- chmod +x /workspace/main
kubectl -n "$NAMESPACE" exec "$POD_NAME" -- /workspace/main

BUILD_SUCCESS=false
if kubectl -n "$NAMESPACE" exec "$POD_NAME" -- /workspace/main 2>&1 | grep -q "镜像构建并推送成功"; then
    BUILD_SUCCESS=true
fi

# 7. 验证镜像
echo -e "\n${YELLOW}[7/6] 验证镜像...${NC}"
VERIFY_POD_NAME="verify-kaniko-image-$(date +%s)"
kubectl -n "$NAMESPACE" run "$VERIFY_POD_NAME" \
    --image="$NEW_IMAGE" \
    --restart=Never \
    --command -- sh -c 'ls -lh /usr/local/app/ && echo "OK"'

sleep 5
if kubectl -n "$NAMESPACE" logs "$VERIFY_POD_NAME" 2>&1 | grep -q "OK"; then
    echo -e "${GREEN}✓ 镜像验证成功${NC}"
    VERIFY_SUCCESS=true
else
    echo -e "${RED}xxx 镜像验证失败${NC}"
    VERIFY_SUCCESS=false
fi

kubectl -n "$NAMESPACE" delete pod "$VERIFY_POD_NAME" --ignore-not-found=true

# 清理
echo -e "\n${YELLOW}清理资源...${NC}"
kubectl -n "$NAMESPACE" delete pod "$POD_NAME" --ignore-not-found=true

echo -e "\n${GREEN}=== 测试总结 ===${NC}"
echo -e "命名空间: $NAMESPACE"
echo -e "构建镜像: $NEW_IMAGE"
if [ "$BUILD_SUCCESS" = true ] && [ "$VERIFY_SUCCESS" = true ]; then
    echo -e "${GREEN}状态: ✓✓✓ 测试通过！${NC}"
    exit 0
else
    echo -e "${RED}状态: xxx 测试失败${NC}"
    exit 1
fi

