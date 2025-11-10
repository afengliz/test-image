# Buildah Go SDK 使用说明

## 概述

本示例演示如何使用 Buildah Go SDK 在程序内构建容器镜像。与 Kaniko 不同，Buildah 提供了完整的 Go SDK，可以直接在代码中调用，无需通过 CLI。

## 核心优势

1. **Go SDK 支持**：提供完整的 Go API，可以直接在代码中使用
2. **Rootless 支持**：原生支持 rootless 模式，无需 root 权限
3. **灵活性高**：可以精确控制构建过程的每一步

## API 使用示例

### 1. 创建构建器

```go
import "github.com/containers/buildah"

ctx := context.Background()
storeOptions, err := buildah.GetDefaultStoreOptions()
if err != nil {
    return err
}

builder, err := buildah.NewBuilder(ctx, storeOptions, "container-name", buildah.BuilderOptions{
    FromImage: "base-image:tag",
})
defer builder.Delete()
```

### 2. 配置镜像

```go
// 设置工作目录
builder.SetWorkDir("/usr/local/app")

// 添加文件
builder.Add("/path/to/file", false, buildah.AddAndCopyOptions{}, "/dest/path")

// 设置入口点
builder.SetCmd([]string{"/usr/local/app/main"})
```

### 3. 提交镜像

```go
imageID, err := builder.Commit(ctx, "image-name:tag", buildah.CommitOptions{})
```

### 4. 推送镜像

```go
import (
    "github.com/containers/image/v5/copy"
    "github.com/containers/image/v5/storage"
    "github.com/containers/image/v5/transports/alltransports"
)

destRef, _ := alltransports.ParseImageName("docker://image-name:tag")
srcRef, _ := storage.Transport.ParseStoreReference(store, imageID)

cp.Image(ctx, policyContext, destRef, srcRef, &cp.Options{
    SourceCtx:      systemContext,
    DestinationCtx: systemContext,
})
```

## 与 Kaniko 对比

| 特性 | Buildah Go SDK | Kaniko CLI |
|------|----------------|------------|
| **API 方式** | ✅ Go API | ❌ CLI 调用 |
| **代码集成** | 🟢 原生支持 | 🟡 通过 exec |
| **错误处理** | 🟢 结构化错误 | 🟡 字符串解析 |
| **性能** | 🟢 直接调用 | 🟡 进程开销 |
| **Rootless** | ✅ 支持 | ✅ 支持 |

## 注意事项

1. **依赖管理**：Buildah Go SDK 依赖较多，需要正确配置 `go.mod`
2. **存储配置**：Rootless 模式需要配置存储驱动（推荐 `fuse-overlayfs`）
3. **认证配置**：推送到私有 registry 需要配置认证信息
4. **权限要求**：某些操作可能需要 `SYS_ADMIN` capability

## 常见问题

### Q: 如何配置 rootless 模式？

A: 在容器中配置存储驱动：
```bash
mkdir -p ~/.config/containers
cat > ~/.config/containers/storage.conf << EOF
[storage]
driver = "overlay"
[storage.options]
mount_program = "/usr/bin/fuse-overlayfs"
EOF
```

### Q: 如何推送到私有 registry？

A: 配置认证信息：
```go
systemContext := &types.SystemContext{
    DockerAuthConfig: &types.DockerAuthConfig{
        Username: "user",
        Password: "pass",
    },
}
```

### Q: 构建失败，提示权限不足？

A: 确保容器有 `SYS_ADMIN` capability（用于挂载文件系统）：
```yaml
securityContext:
  capabilities:
    add:
      - SYS_ADMIN
```

## 参考资源

- [Buildah 官方文档](https://github.com/containers/buildah)
- [Buildah Go API 文档](https://pkg.go.dev/github.com/containers/buildah)
- [containers/image 文档](https://github.com/containers/image)

