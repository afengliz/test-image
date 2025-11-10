# Buildah Go SDK 编译说明

## 编译问题

Buildah Go SDK 依赖 C 库（gpgme），需要：
1. **CGO 支持**：必须启用 `CGO_ENABLED=1`
2. **C 库依赖**：需要安装 `gpgme-dev`、`libassuan-dev` 等
3. **交叉编译困难**：在 macOS 上交叉编译到 Linux 较复杂

## 解决方案

### 方案 1：多阶段 Docker 构建（推荐）⭐

使用 `Dockerfile.build` 进行多阶段构建：

```bash
# 构建包含编译好的程序的镜像
docker build -f Dockerfile.build -t buildah-demo:latest .
```

**优点**：
- ✅ 自动处理所有依赖
- ✅ 在 Linux 环境中编译，避免交叉编译问题
- ✅ 最终镜像只包含运行时依赖

### 方案 2：在容器内编译

使用包含 Go 的容器来编译：

```bash
# 使用 golang 镜像编译
docker run --rm -v $(pwd):/workspace -w /workspace \
  golang:1.20-alpine sh -c "apk add gpgme-dev && go build -o main main.go"
```

### 方案 3：使用 Buildah CLI（替代方案）

如果 Go SDK 编译太复杂，可以使用 CLI 方式（参考 `../buildah_rootless_demo`）：

```go
exec.Command("buildah", "bud", ...)
```

**优点**：
- ✅ 无需编译 Go SDK 代码
- ✅ 依赖更少
- ✅ 已验证可行

## 当前状态

由于编译复杂性，建议：
1. **开发阶段**：使用 `buildah_rootless_demo`（CLI 方式）
2. **生产环境**：使用多阶段 Docker 构建（`Dockerfile.build`）

## 快速开始

```bash
# 1. 构建镜像（包含编译好的程序）
docker build -f Dockerfile.build -t buildah-demo:latest .

# 2. 推送到 registry
docker tag buildah-demo:latest <registry>/buildah-demo:latest
docker push <registry>/buildah-demo:latest

# 3. 在 K8s 中使用
kubectl apply -f buildah-pod.yaml
```

