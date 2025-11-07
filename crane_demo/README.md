# 使用 Crane 在现有镜像上叠加文件

## 功能说明

这个程序演示了如何使用 **Crane** 在现有镜像上叠加文件，无需 Dockerfile 或 Docker 守护进程。

### Crane 的优势

- ✅ **无需 Docker 守护进程**：纯 Go 实现，不依赖 Docker
- ✅ **无需特权模式**：可以在非特权容器中运行
- ✅ **轻量级**：只操作镜像层，不涉及容器运行时
- ✅ **支持镜像操作**：拉取、推送、追加层、修改配置

## 工作原理

### 1. 拉取基础镜像

```go
baseImg, err := crane.Pull(baseImage)
```

### 2. 创建文件层（tarball）

将需要叠加的文件打包成 tarball：

```
/usr/local/app/main  (要叠加的文件)
```

### 3. 追加文件层

```go
newImg, err := crane.Append(baseImg, tarballPath)
```

### 4. 修改镜像配置

```go
newImg, err = crane.Mutate(newImg, func(cfg map[string]interface{}) error {
    config := cfg["config"].(map[string]interface{})
    config["WorkingDir"] = "/usr/local/app"
    config["Entrypoint"] = []string{"/usr/local/app/main"}
    return nil
})
```

### 5. 推送新镜像

```go
err := crane.Push(newImg, newImageName)
```

## 使用方法

### 1. 安装依赖

```bash
cd crane_demo
go mod tidy
```

### 2. 编译程序

```bash
GOOS=linux GOARCH=amd64 go build -o crane-demo main.go
```

### 3. 在 K8s Pod 中运行

```bash
# 复制程序到 Pod
kubectl cp crane-demo ones/<pod-name>:/workspace/crane-demo
kubectl cp server/main ones/<pod-name>:/workspace/server/main

# 运行程序
kubectl exec -n ones <pod-name> -- /workspace/crane-demo
```

## 与 Buildah/Kaniko 对比

| 特性 | Crane | Buildah | Kaniko |
|------|-------|---------|--------|
| **构建镜像** | ❌ 不支持（只能叠加文件） | ✅ 支持 | ✅ 支持 |
| **叠加文件** | ✅ 支持 | ✅ 支持 | ✅ 支持 |
| **推送镜像** | ✅ 支持 | ✅ 支持 | ✅ 支持 |
| **权限要求** | 🟢 低（无需特权） | 🔴 高（需要特权） | 🔴 高（需要特权） |
| **Go SDK** | ✅ 有 | ✅ 有 | ❌ 无 |
| **轻量级** | 🟢 是 | 🟡 中等 | 🟡 中等 |

## 适用场景

### ✅ 适合使用 Crane

1. **在现有镜像上添加文件**
   - 添加配置文件
   - 添加二进制文件
   - 添加静态资源

2. **镜像复制和迁移**
   - 在不同仓库间复制镜像
   - 镜像格式转换

3. **镜像管理操作**
   - 修改镜像配置（环境变量、入口点等）
   - 镜像标签管理

### ❌ 不适合使用 Crane

1. **从 Dockerfile 构建镜像**
   - Crane 不支持执行 Dockerfile 命令
   - 需要配合其他工具（如 Buildah/Kaniko）

2. **复杂的构建流程**
   - 需要编译代码
   - 需要安装依赖
   - 需要执行构建脚本

## 示例：命令行方式

除了 Go SDK，也可以直接使用 `crane` 命令行工具：

```bash
# 1. 创建包含文件的 tarball
mkdir -p usr/local/app
cp main usr/local/app/
tar -czf layer.tar usr/local/app/

# 2. 追加文件层到镜像
crane append \
  --image=registry.kube-system.svc.cluster.local:5000/ones/plugin-host-node:v6.33.1 \
  --tarball=layer.tar \
  --tag=registry.kube-system.svc.cluster.local:5000/new-image:latest

# 3. 修改镜像配置（可选）
crane mutate \
  --entrypoint='["/usr/local/app/main"]' \
  --workdir=/usr/local/app \
  registry.kube-system.svc.cluster.local:5000/new-image:latest
```

## 注意事项

1. **文件路径**：tarball 中的文件路径应该是镜像内的绝对路径
2. **权限**：确保文件有执行权限（如果需要）
3. **镜像格式**：Crane 支持 OCI 和 Docker 镜像格式
4. **认证**：推送镜像需要配置仓库认证信息

## 参考

- [Crane GitHub](https://github.com/google/go-containerregistry/tree/main/cmd/crane)
- [go-containerregistry 文档](https://github.com/google/go-containerregistry)

