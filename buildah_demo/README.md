# Buildah Go SDK 示例

本示例演示如何使用 Buildah Go SDK 在程序内构建容器镜像，支持 rootless 模式。

## 特性

- ✅ 使用 Buildah Go SDK（非 CLI 调用）
- ✅ 支持 rootless 模式
- ✅ 在程序内直接构建镜像
- ✅ 参考 crane_demo 的镜像结构

## 前置要求

### 1. 系统要求

- Linux 内核 3.18+（推荐 RHEL 8+/Fedora 31+/Ubuntu 20.04+）
- 启用用户命名空间（User Namespaces）

### 2. 安装依赖

**Fedora/RHEL:**
```bash
sudo dnf install -y buildah fuse-overlayfs
```

**Ubuntu:**
```bash
sudo apt-get update
sudo apt-get install -y buildah fuse-overlayfs
```

### 3. 配置 rootless 模式

```bash
# 配置用户命名空间
echo "user.max_user_namespaces=28633" | sudo tee /proc/sys/user/max_user_namespaces

# 配置存储驱动（推荐 fuse-overlayfs）
mkdir -p ~/.config/containers
cat > ~/.config/containers/storage.conf << EOF
[storage]
driver = "overlay"
[storage.options]
mount_program = "/usr/bin/fuse-overlayfs"
EOF
```

### 4. 验证配置

```bash
buildah info
```

输出中应包含 `"rootless"`：
```json
"SecurityOptions": [
  "seccomp=unconfined",
  "apparmor=unconfined",
  "rootless"
]
```

## 快速开始

### ⚠️ 编译说明

Buildah Go SDK 需要 CGO 和 C 库支持，编译较复杂。建议：
- **开发测试**：使用 `../buildah_rootless_demo`（CLI 方式，已验证可行）
- **生产环境**：使用多阶段 Docker 构建（见 `Dockerfile.build`）

### 1. 构建 Go 程序

**方式 A：多阶段 Docker 构建（推荐）**

```bash
cd buildah_demo
docker build -f Dockerfile.build -t buildah-demo:latest .
```

**方式 B：本地编译（需要 CGO 和 C 库）**

```bash
cd buildah_demo
# 需要安装 gpgme-dev 等 C 库
go mod download
CGO_ENABLED=1 go build -o main main.go
```

**方式 C：使用 CLI 方式（更简单）**

参考 `../buildah_rootless_demo`，使用 `exec.Command` 调用 Buildah CLI。

### 2. 准备源文件

确保 `../demo_server/main` 文件存在：
```bash
ls -lh ../demo_server/main
```

### 3. 运行程序

**方式 A：在本地运行（需要安装 Buildah）**

```bash
# 设置源文件路径（如果不在容器内）
export MAIN_FILE_PATH=/path/to/demo_server/main

# 运行
./main
```

**方式 B：在容器内运行**

```bash
# 构建包含 Buildah 的镜像
make build-image

# 运行（挂载 server 目录）
make run-docker
```

**方式 C：在 K8s Pod 中运行（推荐）**

```bash
# 使用自动化测试脚本
./test.sh
```

或手动执行：

```bash
# 1. 创建 Pod
kubectl apply -f buildah-pod.yaml

# 2. 等待 Pod 就绪
kubectl -n imgbuild wait --for=condition=Ready pod/buildah-demo --timeout=60s

# 3. 复制文件
kubectl -n imgbuild cp main buildah-demo:/workspace/main
kubectl -n imgbuild exec buildah-demo -- mkdir -p /workspace/server
kubectl -n imgbuild cp ../demo_server/main buildah-demo:/workspace/server/main

# 4. 运行
kubectl -n imgbuild exec buildah-demo -- chmod +x /workspace/main
kubectl -n imgbuild exec buildah-demo -- /workspace/main
```

## 工作原理

1. **创建构建器**：使用 `buildah.NewBuilder` 创建构建器实例
2. **配置镜像**：设置工作目录、添加文件、设置入口点
3. **提交镜像**：使用 `builder.Commit` 提交镜像
4. **推送镜像**：使用 containers/image 库推送到 registry

## 与 Kaniko 对比

| 特性 | Buildah Go SDK | Kaniko CLI |
|------|----------------|------------|
| Go SDK | ✅ 有 | ❌ 无 |
| Rootless | ✅ 原生支持 | ✅ 支持 |
| 使用方式 | Go API | CLI 调用 |
| 灵活性 | 🟢 高 | 🟡 中 |
| 代码集成 | 🟢 原生支持 | 🟡 通过 exec |

## 注意事项

1. **认证配置**：推送到私有 registry 需要配置认证信息
2. **存储驱动**：rootless 模式推荐使用 `fuse-overlayfs`
3. **权限要求**：某些操作可能需要 `SYS_ADMIN` capability
4. **依赖管理**：Buildah Go SDK 依赖较多，需要正确配置 `go.mod`

## 文件说明

- `main.go` - 主程序（使用 Buildah Go SDK）
- `go.mod` - Go 模块定义
- `Dockerfile` - 容器镜像构建文件
- `buildah-pod.yaml` - K8s Pod 配置
- `Makefile` - 构建脚本
- `test.sh` - 自动化测试脚本
- `README.md` - 本文档
- `USAGE.md` - 详细使用说明和 API 文档

## 参考文档

- [Buildah 官方文档](https://github.com/containers/buildah)
- [Buildah Go API 文档](https://pkg.go.dev/github.com/containers/buildah)
- [containers/image 文档](https://github.com/containers/image)

## 相关示例

- `../kaniko_rootless_demo` - Kaniko CLI 方式示例
- `../crane_demo` - Crane 方式示例
- `../buildah_rootless_demo` - Buildah CLI 方式示例
