#!/bin/bash

set -eux

# 颜色输出
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== Kaniko 程序内构建测试 ===${NC}"

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# 命名空间
NAMESPACE="imgbuild"

# 1. 检查 server/main 文件是否存在
echo -e "\n${YELLOW}[1/5] 检查源文件...${NC}"
SERVER_MAIN="../demo_server/main"
if [ ! -f "$SERVER_MAIN" ]; then
    echo -e "${YELLOW}警告: $SERVER_MAIN 不存在，尝试构建...${NC}"
    if [ -f "../demo_server/main.go" ]; then
        cd ../demo_server
        go build -o main main.go
        cd "$SCRIPT_DIR"
        echo -e "${GREEN}✓ demo_server/main 构建成功${NC}"
    else
        echo -e "${RED}✗ 找不到 demo_server/main.go，请先创建${NC}"
        exit 1
    fi
fi

if [ ! -f "$SERVER_MAIN" ]; then
    echo -e "${RED}✗ server/main 文件不存在: $SERVER_MAIN${NC}"
    exit 1
fi
echo -e "${GREEN}✓ 源文件检查通过${NC}"

# 2. 构建 Go 程序
echo -e "\n${YELLOW}[2/5] 构建 Go 程序...${NC}"
make build
echo -e "${GREEN}✓ Go 程序构建成功${NC}"

# 3. 构建包含 Kaniko 的 Docker 镜像
echo -e "\n${YELLOW}[3/5] 构建 Docker 镜像...${NC}"
make build-image
echo -e "${GREEN}✓ Docker 镜像构建成功${NC}"

# 4. 创建命名空间
echo -e "\n${YELLOW}[4/5] 准备 K8s 环境...${NC}"
kubectl get ns "$NAMESPACE" >/dev/null 2>&1 || kubectl create ns "$NAMESPACE"
echo -e "${GREEN}✓ 命名空间已就绪${NC}"

# 5. 在 K8s 中运行（需要先推送镜像）
echo -e "\n${YELLOW}[5/5] 在 K8s 中运行...${NC}"
echo -e "${YELLOW}提示: 需要先将镜像推送到 registry${NC}"
echo -e "${YELLOW}或者使用 Docker 方式运行: make run-docker${NC}"

# 检查是否在 K8s 环境中
if kubectl cluster-info >/dev/null 2>&1; then
    # 创建临时 Pod 配置（使用 hostPath 挂载 server 目录）
    TEMP_POD_YAML=$(mktemp)
    cat > "$TEMP_POD_YAML" <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: kaniko-build-test
  namespace: $NAMESPACE
spec:
  restartPolicy: Never
  containers:
    - name: kaniko-build
      image: kaniko-build:latest
      imagePullPolicy: Never
      volumeMounts:
        - name: server
          mountPath: /workspace/server
          readOnly: true
      securityContext:
        allowPrivilegeEscalation: false
  volumes:
    - name: server
      hostPath:
        path: $(realpath ../demo_server)
        type: Directory
EOF

    echo -e "${YELLOW}创建测试 Pod...${NC}"
    kubectl apply -f "$TEMP_POD_YAML" || {
        echo -e "${YELLOW}如果 Pod 创建失败，可能需要：${NC}"
        echo -e "${YELLOW}1. 推送镜像到 registry${NC}"
        echo -e "${YELLOW}2. 修改 kaniko-pod.yaml 中的镜像地址${NC}"
        echo -e "${YELLOW}3. 或使用 Docker 方式: make run-docker${NC}"
        rm -f "$TEMP_POD_YAML"
        exit 0
    }

    echo -e "${YELLOW}等待 Pod 运行...${NC}"
    sleep 5

    echo -e "${GREEN}查看构建日志:${NC}"
    kubectl -n "$NAMESPACE" logs -f kaniko-build-test || true

    echo -e "\n${YELLOW}清理测试 Pod...${NC}"
    kubectl -n "$NAMESPACE" delete pod kaniko-build-test --ignore-not-found=true
    rm -f "$TEMP_POD_YAML"
else
    echo -e "${YELLOW}未检测到 K8s 集群，跳过 K8s 测试${NC}"
    echo -e "${YELLOW}可以使用 Docker 方式运行: make run-docker${NC}"
fi

echo -e "\n${GREEN}=== 测试完成 ===${NC}"
echo -e "构建的镜像: registry.kube-system.svc.cluster.local:5000/new-kaniko-image:latest"
echo -e "验证镜像: kubectl -n $NAMESPACE run verify --image=localhost:5000/new-kaniko-image:latest --restart=Never --command -- /usr/local/app/main"

