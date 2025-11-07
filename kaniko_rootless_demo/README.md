# Kaniko 非特权模式构建镜像示例

## 概述

此示例演示如何在 Kubernetes Pod 中使用 Kaniko 的**非特权模式**构建和推送容器镜像。非特权模式提高了安全性，避免了容器逃逸对主机的影响。

## 核心特性

### 1. 非特权模式构建

- **无需 privileged: true**：理论上 Kaniko 可以在非特权模式下运行
- **安全性高**：即使容器被攻破，影响范围有限
- **符合最小权限原则**：只授予必要的 capabilities

### 2. 与 Privileged 模式的区别

| 特性 | 非特权模式 | Privileged 模式 |
|------|-----------|----------------|
| **权限要求** | 受限的 capabilities | 主机 root 权限 |
| **安全性** | 🟢 高（容器逃逸影响范围小） | 🔴 低（容器逃逸可能影响整个节点） |
| **K8s 配置** | `securityContext` 不设置 `privileged: true` | `securityContext.privileged: true` |
| **Capabilities** | 只添加必要的 capabilities | 所有 capabilities |
| **适用环境** | 生产环境 | 开发/测试环境 |

## 部署到 K8s

### 1. 创建 Deployment

使用 `k8s/kaniko-rootless-demo-deployment.yaml` 创建 Pod。此 Deployment **不设置 `privileged: true`**，以测试非特权模式。

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kaniko-rootless-demo-deployment
  namespace: ones
spec:
  template:
    spec:
      containers:
      - name: kaniko-rootless-demo
        image: registry.cn-hangzhou.aliyuncs.com/kube-image-repo/kaniko:v1.9.1-debug
        securityContext:
          runAsNonRoot: false
          allowPrivilegeEscalation: false
          capabilities:
            drop:
            - ALL
            add:
            - CHOWN
            - SETUID
            - SETGID
            - FOWNER
```

**执行命令**:
```bash
kubectl apply -f k8s/kaniko-rootless-demo-deployment.yaml
```

### 2. 编译程序

在本地编译 Go 程序，目标架构为 `linux/amd64`：

```bash
cd kaniko_rootless_demo
GOOS=linux GOARCH=amd64 go build -o kaniko-rootless-demo main.go
```

### 3. 复制文件到 Pod

将编译好的程序和 `server/main` 复制到 Pod 中：

```bash
POD_NAME=$(kubectl get pods -n ones -l app=kaniko-rootless-demo -o jsonpath='{.items[0].metadata.name}')
kubectl cp kaniko-rootless-demo ones/$POD_NAME:/workspace/kaniko-rootless-demo
kubectl cp ../server/main ones/$POD_NAME:/workspace/server/main
```

### 4. 运行程序

在 Pod 中执行构建程序：

```bash
kubectl exec -n ones $POD_NAME -- /workspace/kaniko-rootless-demo
```

## 验证新镜像

### 1. 创建测试 Deployment

使用 `k8s/test-kaniko-rootless-deployment.yaml` 创建 Pod，运行新构建的镜像：

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-kaniko-rootless-deployment
  namespace: ones
spec:
  template:
    spec:
      containers:
      - name: test-kaniko-rootless
        image: localhost:5000/new-kaniko-rootless-image:latest
        imagePullPolicy: IfNotPresent
```

### 2. 查看 Pod 日志

检查 `test-kaniko-rootless-pod` 的日志，确认是否打印 "Hello, World!"：

```bash
POD_NAME=$(kubectl get pods -n ones -l app=test-kaniko-rootless -o jsonpath='{.items[0].metadata.name}')
kubectl logs -n ones $POD_NAME
```

**预期输出**:
```
Hello World
Server started on port 8081
```

## 可能遇到的问题

### 问题 1: 权限不足

**描述**: Kaniko 可能需要某些系统权限来执行构建操作。

**解决方案**:
- 尝试添加更多 capabilities（如 `SYS_ADMIN`、`DAC_OVERRIDE`）
- 如果仍然失败，可能需要使用 `privileged: true`

### 问题 2: 文件系统权限

**描述**: Kaniko 可能需要访问某些文件系统功能。

**解决方案**:
- 确保工作目录有写权限
- 检查 `/kaniko` 目录的权限

### 问题 3: 网络问题

**描述**: 无法连接到 registry。

**解决方案**:
- 使用 `--insecure` 和 `--skip-tls-verify` 参数
- 检查 registry 的网络连接

## 可行性结论

**待测试**：Kaniko 在非特权模式下的可行性需要实际测试验证。

**理论支持**：
- ✅ Kaniko 官方文档声称支持非特权模式
- ✅ 不依赖 Docker 守护进程
- ✅ 在用户空间执行构建

**实际限制**：
- ⚠️ 某些操作可能需要特殊权限
- ⚠️ 文件系统操作可能受限
- ⚠️ 网络配置可能需要额外设置

## 建议

1. **如果非特权模式失败**：
   - 考虑使用 `privileged: true`（牺牲安全性）
   - 或使用 **Crane**（无需特权模式，但只支持叠加文件）

2. **如果非特权模式成功**：
   - 这是最佳方案，兼顾安全性和功能性
   - 适合生产环境使用

## 附录

### 相关文件

- `kaniko_rootless_demo/main.go`: Kaniko 非特权模式构建程序
- `kaniko_rootless_demo/go.mod`: Go 模块定义
- `kaniko_rootless_demo/README.md`: 本文档
- `k8s/kaniko-rootless-demo-deployment.yaml`: Kubernetes Deployment 配置

### 参考链接

- [Kaniko 官方文档](https://github.com/GoogleContainerTools/kaniko)
- [Kubernetes Security Context](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/)

---

**文档版本**: v1.0  
**创建日期**: 2025-11-07  
**作者**: 技术团队

