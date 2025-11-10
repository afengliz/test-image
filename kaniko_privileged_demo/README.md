# Kaniko Privileged 模式构建镜像示例

本示例演示如何在 Kubernetes 集群中使用 Kaniko 在**特权模式**下构建并推送容器镜像。

## 特性

- ⚠️ **使用 privileged 模式**：容器设置 `privileged: true`
- ✅ **无需 Docker 守护进程**：使用 Kaniko 直接在用户空间构建
- ✅ **程序内构建**：在 Go 程序内调用 Kaniko executor
- ⚠️ **安全性较低**：适合开发/测试环境，不推荐生产环境

> 📖 **想了解 Rootless 与 Privileged 模式的区别？** 查看 [对比文档](../kaniko_rootless_demo/COMPARISON.md)

## 前置要求

- 已配置 `kubectl` 访问 Kubernetes 集群
- 集群内置私有镜像仓库服务：`registry`（位于 `kube-system` 命名空间）
  - 推送端点：`registry.kube-system.svc.cluster.local:5000`
  - 工作负载拉取端点：`localhost:5000`
- 基础镜像存在：`registry.kube-system.svc.cluster.local:5000/ones/plugin-host-node:v6.33.1`

## 快速开始

### 1. 构建 Go 程序

```bash
cd kaniko_privileged_demo
go mod download
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build-image main.go
```

### 2. 准备源文件

确保 `../demo_server/main` 文件存在：
```bash
ls -lh ../demo_server/main
```

### 3. 运行方式

**方式 A：在 Docker 容器内运行**

```bash
# 使用 Kaniko 官方镜像
docker run --rm \
  --privileged \
  -v $(pwd)/../demo_server:/workspace/server:ro \
  -v $(pwd)/build-image:/workspace/build-image:ro \
  registry.cn-hangzhou.aliyuncs.com/kube-image-repo/kaniko:v1.9.1-debug \
  /workspace/build-image
```

**方式 B：在 K8s Pod 中运行**

```bash
# 1. 创建 Pod（使用 privileged 模式）
kubectl apply -f kaniko-pod.yaml

# 2. 等待 Pod 就绪
kubectl -n imgbuild wait --for=condition=Ready pod/kaniko-privileged --timeout=60s

# 3. 复制文件
kubectl -n imgbuild cp build-image kaniko-privileged:/workspace/build-image
kubectl -n imgbuild exec kaniko-privileged -- mkdir -p /workspace/server
kubectl -n imgbuild cp ../demo_server/main kaniko-privileged:/workspace/server/main

# 4. 运行构建程序
kubectl -n imgbuild exec kaniko-privileged -- chmod +x /workspace/build-image
kubectl -n imgbuild exec kaniko-privileged -- /workspace/build-image
```

**方式 C：使用自动化测试脚本**

```bash
./test.sh
```

## 工作原理

1. **创建构建上下文**：在 `/workspace/build-context` 目录准备 Dockerfile 和源文件
2. **调用 Kaniko executor**：使用 `exec.Command` 调用 `/kaniko/executor`
3. **构建镜像**：Kaniko 在用户空间构建镜像（即使使用 privileged 模式，Kaniko 仍使用用户空间操作）
4. **推送镜像**：直接推送到 registry

## 与 Rootless 模式对比

| 特性 | Privileged 模式 | Rootless 模式 |
|------|----------------|---------------|
| **安全配置** | `privileged: true` | `allowPrivilegeEscalation: false` |
| **安全性** | 🔴 低 | 🟢 高 |
| **构建功能** | ✅ 完全相同 | ✅ 完全相同 |
| **性能** | 🟢 相同 | 🟢 相同 |
| **容器逃逸风险** | 🔴 高 | 🟢 低 |
| **适用环境** | ⚠️ 开发/测试 | ✅ 生产环境 |

**关键结论**：
- 功能相同：两种模式在构建功能上完全相同
- 安全性不同：Rootless 模式安全性更高
- 推荐使用 Rootless 模式：生产环境应优先使用 Rootless 模式

详见：[对比文档](../kaniko_rootless_demo/COMPARISON.md)

## 文件说明

- `main.go` - 主程序（调用 Kaniko executor）
- `go.mod` - Go 模块定义
- `kaniko-pod.yaml` - K8s Pod 配置（privileged 模式）
- `test.sh` - 自动化测试脚本
- `Makefile` - 构建和运行脚本
- `README.md` - 本文档

## 注意事项

1. **安全性警告**：
   - ⚠️ Privileged 模式安全性较低，存在容器逃逸风险
   - ⚠️ 不推荐在生产环境使用
   - ✅ 建议使用 Rootless 模式（参考 `../kaniko_rootless_demo`）

2. **权限要求**：
   - 需要集群允许创建 privileged Pod
   - 某些集群（如 OpenShift）可能限制 privileged Pod

3. **功能说明**：
   - 即使使用 privileged 模式，Kaniko 仍使用用户空间操作
   - 功能与 Rootless 模式完全相同
   - 使用 privileged 模式主要是为了兼容性，而非功能需求

## 参考文档

- [Kaniko 官方文档](https://github.com/GoogleContainerTools/kaniko)
- [Rootless 模式示例](../kaniko_rootless_demo/README.md)
- [Rootless vs Privileged 对比](../kaniko_rootless_demo/COMPARISON.md)
