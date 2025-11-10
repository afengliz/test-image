# 程序内构建方式说明

本文档说明如何使用 Go 程序在程序内调用 Kaniko 构建镜像，无需创建 K8s Job。

## 与 crane_demo 的对比

本程序实现的功能与 `crane_demo` 相同：
- ✅ 从基础镜像 `registry.kube-system.svc.cluster.local:5000/ones/plugin-host-node:v6.33.1` 开始
- ✅ 将 `/workspace/server/main` 文件复制到镜像的 `/usr/local/app/main`
- ✅ 设置工作目录为 `/usr/local/app`
- ✅ 设置入口点为 `/usr/local/app/main`
- ✅ 构建并推送到 `registry.kube-system.svc.cluster.local:5000/new-kaniko-image:latest`

## 与 Job 方式的区别

| 特性 | 程序内构建 | Job 方式 |
|------|-----------|---------|
| **部署方式** | 单个 Pod 运行程序 | 需要创建 Job |
| **灵活性** | 🟢 高 - 可在程序内动态调整 | 🟡 中 - 需要修改 YAML |
| **代码控制** | 🟢 完全由代码控制 | 🟡 需要 YAML 配置 |
| **适用场景** | 集成到其他服务中 | 独立的构建任务 |

## 快速开始

### 1. 构建 Go 程序

```bash
cd kaniko_rootless_demo
make build
```

这会生成 `main` 二进制文件。

### 2. 构建包含 Kaniko 的 Docker 镜像

```bash
make build-image
```

这会创建一个包含 Kaniko executor 和构建程序的 Docker 镜像。

### 3. 运行方式

#### 方式 A：在 Docker 容器内运行（本地测试）

```bash
# 确保 server 目录存在
ls ../server/main

# 运行
make run-docker
```

#### 方式 B：在 K8s Pod 中运行

```bash
# 1. 构建并推送镜像到 registry
docker build -t registry.kube-system.svc.cluster.local:5000/kaniko-build:latest .
docker push registry.kube-system.svc.cluster.local:5000/kaniko-build:latest

# 2. 修改 kaniko-pod.yaml 中的 hostPath（如果需要）
# 或使用其他方式提供 server 目录

# 3. 创建 Pod 运行
kubectl apply -f kaniko-pod.yaml
kubectl -n imgbuild logs -f kaniko-build

# 4. 查看结果
kubectl -n imgbuild get pod kaniko-build
```

#### 方式 C：本地运行（需要安装 Kaniko）

```bash
# 设置 Kaniko executor 路径
export KANIKO_EXECUTOR=/path/to/kaniko/executor

# 运行
make run-local
# 或
go run main.go
```

## 程序工作流程

1. **检查文件**：验证 `/workspace/server/main` 文件是否存在
2. **创建临时目录**：在 `/tmp/kaniko-build` 创建构建上下文
3. **生成 Dockerfile**：
   ```dockerfile
   FROM registry.kube-system.svc.cluster.local:5000/ones/plugin-host-node:v6.33.1
   WORKDIR /usr/local/app
   COPY main /usr/local/app/main
   ENTRYPOINT ["/usr/local/app/main"]
   ```
4. **复制文件**：将 `main` 文件复制到构建上下文
5. **调用 Kaniko**：执行 Kaniko executor 构建并推送镜像

## 配置参数

可以在 `main.go` 中修改以下参数：

```go
baseImage := "registry.kube-system.svc.cluster.local:5000/ones/plugin-host-node:v6.33.1"
mainFilePath := "/workspace/server/main"
newImageName := "registry.kube-system.svc.cluster.local:5000/new-kaniko-image:latest"
```

或通过环境变量：

```bash
export KANIKO_EXECUTOR="/kaniko/executor"  # Kaniko executor 路径
```

## 验证构建结果

构建成功后，可以验证镜像：

```bash
# 在 K8s 中运行验证 Pod
kubectl -n imgbuild run verify-image \
  --image=localhost:5000/new-kaniko-image:latest \
  --restart=Never --command -- /usr/local/app/main

# 查看日志
kubectl -n imgbuild logs verify-image

# 清理
kubectl -n imgbuild delete pod verify-image --ignore-not-found
```

## 常见问题

### 1. Kaniko executor 不存在

**错误**：
```
Kaniko executor 不存在: /kaniko/executor
```

**解决**：
- 确保在 Kaniko 容器内运行，或
- 设置 `KANIKO_EXECUTOR` 环境变量指向正确的路径

### 2. main 文件不存在

**错误**：
```
main 文件不存在: /workspace/server/main
```

**解决**：
- 确保 `/workspace/server/main` 文件存在
- 在 Pod 中挂载包含 `main` 文件的目录

### 3. 镜像推送失败

**错误**：
```
推送镜像失败: ...
```

**解决**：
- 检查 registry 地址是否正确
- 确认网络连接正常
- 检查是否有推送权限

## 与 crane_demo 的对比

| 特性 | 本程序（Kaniko） | crane_demo |
|------|-----------------|------------|
| **构建方式** | Kaniko executor | Crane 库 |
| **依赖** | 需要 Kaniko 容器/二进制 | Go 库（无需外部工具） |
| **构建速度** | 🟢 快（支持缓存） | 🟡 中等 |
| **功能** | ✅ 完整 Dockerfile 支持 | ✅ 文件叠加 |
| **适用场景** | 需要完整构建功能 | 简单文件叠加 |

## 总结

程序内构建方式提供了更高的灵活性，可以在代码中动态控制构建过程，适合集成到其他服务中。与 crane_demo 相比，使用 Kaniko 可以获得完整的 Dockerfile 支持，包括多阶段构建、层缓存等功能。

