# Buildah Rootless 模式构建镜像示例

## 功能说明

这个程序演示了如何在 **Rootless 模式**（非 root 权限）下使用 Buildah 构建和推送容器镜像。

### Rootless 模式的特点

- ✅ **无需 root 权限**：以普通用户身份运行
- ✅ **使用用户命名空间**：通过 `buildah unshare` 创建隔离的用户命名空间
- ✅ **安全性高**：即使容器被攻破，影响范围也有限
- ✅ **适合生产环境**：符合最小权限原则

### 与 Privileged 模式的区别

| 特性 | Rootless 模式 | Privileged 模式 |
|------|--------------|----------------|
| **权限要求** | 普通用户 | root 权限 |
| **安全性** | 🟢 高 | 🔴 低 |
| **存储驱动** | vfs（不需要 remount） | overlay/vfs |
| **使用场景** | 生产环境 | 开发/测试环境 |
| **配置复杂度** | 🟡 中等 | 🟢 简单 |

## 前置要求

### 1. 安装 Buildah

```bash
# 在 Pod 中安装 Buildah
apk add buildah  # Alpine Linux
# 或
apt-get install buildah  # Debian/Ubuntu
```

### 2. 配置 subuid/subgid（通常容器镜像已配置）

Rootless 模式需要用户命名空间支持，通常需要配置 `/etc/subuid` 和 `/etc/subgid`：

```bash
# 查看当前用户的 subuid 配置
cat /etc/subuid | grep $(whoami)
# 或
cat /etc/subuid | grep 1000
```

在 Kubernetes Pod 中，如果镜像已经配置好，通常不需要手动配置。

### 3. 安装 fuse-overlayfs（可选，用于 overlay 驱动）

如果使用 overlay 驱动而不是 vfs，需要安装 `fuse-overlayfs`：

```bash
apk add fuse-overlayfs  # Alpine Linux
```

**注意**：本示例使用 `vfs` 驱动，不需要 `fuse-overlayfs`。

## 使用方法

### 1. 编译程序

```bash
cd buildah_rootless_demo
go build -o buildah-rootless-demo main.go
```

### 2. 在 Kubernetes Pod 中运行

#### 方式 1：在非特权 Pod 中运行（推荐）

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: buildah-rootless-demo
  namespace: ones
spec:
  replicas: 1
  selector:
    matchLabels:
      app: buildah-rootless
  template:
    metadata:
      labels:
        app: buildah-rootless
    spec:
      containers:
      - name: buildah-rootless
        image: localhost:5000/ones/ones/ones-toolkit:v6.37.0-ones.1
        command: ["/bin/sh"]
        args:
          - -c
          - |
            sleep 3600
        # 注意：不设置 privileged: true
        # securityContext:
        #   privileged: false  # 默认值，Rootless 模式
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "2Gi"
            cpu: "1000m"
```

#### 方式 2：在特权 Pod 中运行（用于对比测试）

```yaml
securityContext:
  privileged: true  # 特权模式，用于对比
```

### 3. 复制文件到 Pod

```bash
# 复制编译好的程序
kubectl cp buildah-rootless-demo ones/buildah-rootless-demo-xxx:/workspace/buildah-rootless-demo

# 复制 server/main（如果需要）
kubectl cp ../server/main ones/buildah-rootless-demo-xxx:/workspace/server/main
```

### 4. 在 Pod 中运行

```bash
# 进入 Pod
kubectl exec -it -n ones buildah-rootless-demo-xxx -- /bin/sh

# 运行程序
/workspace/buildah-rootless-demo
```

## 工作原理

### 1. buildah unshare

`buildah unshare` 是 Rootless 模式的核心命令：

```bash
buildah unshare buildah bud ...
```

它会：
1. 创建新的用户命名空间
2. 映射当前用户的 UID/GID 到命名空间内的 root
3. 在命名空间中运行构建命令
4. 允许非 root 用户执行需要 root 权限的操作（在命名空间内）

### 2. 存储驱动选择

**vfs 驱动**（本示例使用）：
- ✅ 不需要 remount 权限
- ✅ 适合 Rootless 模式
- ⚠️ 性能较低（每个层都是完整副本）

**overlay 驱动**（需要 fuse-overlayfs）：
- ✅ 性能更好（copy-on-write）
- ⚠️ 需要安装 fuse-overlayfs
- ⚠️ 在某些环境下可能仍有权限问题

### 3. 配置文件位置

Rootless 模式的配置文件存储在用户目录：

- `~/.config/containers/storage.conf`：存储配置
- `~/.config/containers/containers.conf`：容器配置
- `~/.local/share/containers/storage`：镜像存储位置

## 常见问题

### 1. 错误：`permission denied` 或 `remount`

**原因**：使用了需要 remount 权限的存储驱动（如 overlay），但没有 remount 权限。

**解决**：
- 使用 `vfs` 存储驱动（本示例已配置）
- 或安装 `fuse-overlayfs` 并使用 overlay 驱动

### 2. 错误：`could not find "netavark"`

**原因**：网络后端配置问题。

**解决**：确保 `containers.conf` 中配置了正确的网络后端，或安装 `netavark`。

### 3. 错误：`write /proc/xxx/gid_map: operation not permitted`

**原因**：用户命名空间创建失败，可能是：
- 容器没有 `CAP_SYS_ADMIN` 权限
- 主机不支持用户命名空间
- `/proc/sys/user/max_user_namespaces` 限制

**解决**：
- 检查 Pod 的 `securityContext` 配置
- 检查主机的用户命名空间支持
- 在某些 Kubernetes 环境中，可能需要特殊配置

### 4. 构建速度慢

**原因**：`vfs` 驱动性能较低。

**解决**：
- 使用 `overlay` 驱动 + `fuse-overlayfs`（如果环境支持）
- 或使用 Privileged 模式（如果安全要求允许）

## 与 Privileged 模式对比

### 代码差异

**Rootless 模式**（本示例）：
```go
// 使用 buildah unshare
cmd := exec.Command("buildah", "unshare", "buildah", "bud", ...)
```

**Privileged 模式**（buildah_demo）：
```go
// 直接使用 buildah bud
cmd := exec.Command("buildah", "bud", ...)
```

### 配置差异

**Rootless 模式**：
- 存储驱动：`vfs`（不需要 remount）
- 配置目录：`~/.config/containers/`
- 存储目录：`~/.local/share/containers/storage`

**Privileged 模式**：
- 存储驱动：`overlay` 或 `vfs`
- 配置目录：`/root/.config/containers/` 或系统目录
- 存储目录：`/var/lib/containers/storage`

## 参考

- [Buildah Rootless 文档](https://github.com/containers/buildah/blob/main/docs/tutorials/01-intro.md#rootless-mode)
- [Podman/Buildah Rootless 指南](https://github.com/containers/podman/blob/main/docs/tutorials/rootless_tutorial.md)
- [用户命名空间文档](https://man7.org/linux/man-pages/man7/user_namespaces.7.html)

