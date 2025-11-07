# Buildah 特权模式构建镜像示例

## 功能说明

这个程序演示了如何在 **Privileged 模式**（特权模式）下使用 Buildah 构建和推送容器镜像。

### Privileged 模式的特点

- ✅ **需要 root 权限**：容器以 root 用户运行
- ✅ **配置简单**：不需要额外的用户命名空间配置
- ✅ **性能好**：可以使用 overlay 存储驱动
- ⚠️ **安全性较低**：容器逃逸后可能影响整个节点
- ⚠️ **适合开发/测试环境**：生产环境建议使用 Rootless 模式

### 与 Rootless 模式的区别

| 特性 | Privileged 模式 | Rootless 模式 |
|------|---------------|--------------|
| **权限要求** | root 权限 | 普通用户 |
| **安全性** | 🔴 低 | 🟢 高 |
| **存储驱动** | overlay/vfs | vfs（不需要 remount） |
| **使用场景** | 开发/测试环境 | 生产环境 |
| **配置复杂度** | 🟢 简单 | 🟡 中等 |

## 前置要求

### 1. 安装 Buildah

```bash
# 在 Pod 中安装 Buildah
apk add buildah  # Alpine Linux
# 或
apt-get install buildah  # Debian/Ubuntu
```

### 2. 配置存储（通常容器镜像已配置）

Privileged 模式使用系统级存储配置：
- `/etc/containers/storage.conf`：存储配置
- `/var/lib/containers/storage`：镜像存储位置

## 使用方法

### 1. 编译程序

```bash
cd buildah_demo
go build -o buildah-demo main.go
```

### 2. 在 Kubernetes Pod 中运行

#### 创建特权 Pod

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: buildah-demo-deployment
  namespace: ones
spec:
  replicas: 1
  selector:
    matchLabels:
      app: buildah-demo
  template:
    metadata:
      labels:
        app: buildah-demo
    spec:
      containers:
      - name: buildah-demo
        image: localhost:5000/ones/ones/ones-toolkit:v6.37.0-ones.1
        command: ["/bin/sh"]
        args:
          - -c
          - |
            sleep 3600
        securityContext:
          privileged: true  # 启用特权模式
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "2Gi"
            cpu: "1000m"
```

### 3. 复制文件到 Pod

```bash
# 复制编译好的程序
kubectl cp buildah-demo ones/buildah-demo-xxx:/workspace/buildah-demo

# 复制 server/main（如果需要）
kubectl cp ../server/main ones/buildah-demo-xxx:/workspace/server/main
```

### 4. 在 Pod 中运行

```bash
# 进入 Pod
kubectl exec -it -n ones buildah-demo-xxx -- /bin/sh

# 运行程序
/workspace/buildah-demo
```

## 工作原理

### 1. 直接使用 buildah bud

Privileged 模式可以直接使用 `buildah bud` 命令：

```bash
buildah bud -f Dockerfile -t image-name .
```

### 2. 存储驱动选择

**overlay 驱动**（推荐）：
- ✅ 性能好（copy-on-write）
- ✅ 适合 Privileged 模式
- ⚠️ 需要 remount 权限（Privileged 模式提供）

**vfs 驱动**：
- ✅ 不需要 remount 权限
- ⚠️ 性能较低（每个层都是完整副本）

### 3. 配置文件位置

Privileged 模式的配置文件存储在系统目录：
- `/etc/containers/storage.conf`：存储配置
- `/etc/containers/containers.conf`：容器配置
- `/var/lib/containers/storage`：镜像存储位置

## 常见问题

### 1. 错误：`permission denied`

**原因**：Pod 没有启用 `privileged: true`。

**解决**：在 Deployment 的 `securityContext` 中设置 `privileged: true`。

### 2. 错误：`remount` 失败

**原因**：使用了 overlay 驱动但没有 remount 权限。

**解决**：确保 Pod 启用了 `privileged: true`。

### 3. 构建速度慢

**原因**：使用了 vfs 驱动。

**解决**：切换到 overlay 驱动（需要 Privileged 模式）。

## 与 Rootless 模式对比

### 代码差异

**Privileged 模式**（本示例）：
```go
// 直接使用 buildah bud
cmd := exec.Command("buildah", "bud", ...)
```

**Rootless 模式**（buildah_rootless_demo）：
```go
// 使用 buildah unshare
cmd := exec.Command("buildah", "unshare", "buildah", "bud", ...)
```

### 配置差异

**Privileged 模式**：
- 存储驱动：`overlay` 或 `vfs`
- 配置目录：`/etc/containers/`
- 存储目录：`/var/lib/containers/storage`

**Rootless 模式**：
- 存储驱动：`vfs`（不需要 remount）
- 配置目录：`~/.config/containers/`
- 存储目录：`~/.local/share/containers/storage`

## 参考

- [Buildah 官方文档](https://github.com/containers/buildah)
- [Buildah Rootless 文档](https://github.com/containers/buildah/blob/main/docs/tutorials/01-intro.md#rootless-mode)
- [Kubernetes Security Context](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/)

