# Kaniko Rootless 模式构建镜像示例

本示例演示如何在 Kubernetes 集群中使用 Kaniko 在**非特权模式**下构建并推送容器镜像。

## 特性

- ✅ **无需 privileged 模式**：容器设置 `allowPrivilegeEscalation: false`
- ✅ **无需 Docker 守护进程**：使用 Kaniko 直接在用户空间构建
- ✅ **使用 ConfigMap 提供构建上下文**：通过 initContainer 准备构建文件
- ✅ **安全可靠**：适合生产环境使用

> 📖 **想了解 Rootless 与 Privileged 模式的区别？** 查看 [对比文档](./COMPARISON.md)

## 前置要求

- 已配置 `kubectl` 访问 Kubernetes 集群
- 集群内置私有镜像仓库服务：`registry`（位于 `kube-system` 命名空间）
  - 推送端点：`registry.kube-system.svc.cluster.local:5000`
  - 工作负载拉取端点：`localhost:5000`
- 基础镜像存在：`registry.kube-system.svc.cluster.local:5000/ones/plugin-host-node:v6.33.1`

## 快速开始

### 方式一：程序内构建（推荐）⭐

在程序内直接调用 Kaniko 构建镜像，无需创建 Job。

> 📖 **详细说明**：查看 [程序内构建文档](./PROGRAM_BUILD.md)

#### 1. 构建程序

```bash
cd kaniko_rootless_demo
make build
```

#### 2. 运行方式

**方式 A：在 Docker 容器内运行**

```bash
# 构建包含 Kaniko 的 Docker 镜像
make build-image

# 运行（挂载 server 目录）
make run-docker
```

**方式 B：在 K8s Pod 中运行**

```bash
# 构建并推送镜像到 registry
docker build -t registry.kube-system.svc.cluster.local:5000/kaniko-build:latest .
docker push registry.kube-system.svc.cluster.local:5000/kaniko-build:latest

# 创建 Pod 运行
kubectl apply -f kaniko-pod.yaml
kubectl -n imgbuild logs -f kaniko-build
```

**方式 D：使用自动化测试脚本**

```bash
# 完整测试流程（构建、运行、验证）
./test-program.sh

# 仅验证构建的镜像
./verify-image.sh
```

**方式 C：本地运行（需要安装 Kaniko）**

```bash
# 设置 Kaniko executor 路径
export KANIKO_EXECUTOR=/path/to/executor
make run-local
```

### 方式二：使用 Job 方式（传统方式）

#### 使用自动化测试脚本

```bash
cd kaniko_rootless_demo
./test.sh
```

脚本会自动完成：
1. 创建命名空间
2. 清理旧资源
3. 创建 ConfigMap
4. 运行 Kaniko Job
5. 验证镜像内容

#### 手动执行

#### 1. 创建命名空间

```bash
kubectl get ns imgbuild >/dev/null 2>&1 || kubectl create ns imgbuild
```

#### 2. 创建构建上下文 ConfigMap

```bash
kubectl apply -f kaniko-context.yaml
```

#### 3. 创建并运行 Kaniko Job

```bash
kubectl apply -f kaniko-job.yaml
kubectl -n imgbuild wait --for=condition=complete job/kaniko-addfile --timeout=5m
```

#### 4. 查看构建日志

```bash
kubectl -n imgbuild logs job/kaniko-addfile
```

#### 5. 验证镜像内容

```bash
kubectl -n imgbuild run verify-image \
  --image=localhost:5000/new-kaniko-image:latest \
  --restart=Never --command -- sh -c 'cat /opt/app/hello.txt && echo OK'

kubectl -n imgbuild logs verify-image

# 清理
kubectl -n imgbuild delete pod verify-image --ignore-not-found
```

## 文件说明

### Job 方式（传统方式）
- `kaniko-context.yaml`: 包含 Dockerfile 和构建文件的 ConfigMap
- `kaniko-job.yaml`: Kaniko Job 配置，包含 initContainer 和 executor
- `test.sh`: 自动化测试脚本

### 程序内构建方式（推荐）
- `main.go`: Go 程序，在程序内调用 Kaniko 构建镜像
- `go.mod`: Go 模块定义
- `Dockerfile`: 用于构建包含 Kaniko 的容器镜像
- `Makefile`: 构建和运行脚本
- `kaniko-pod.yaml`: K8s Pod 配置，用于运行构建程序
- `test-program.sh`: 自动化测试脚本（构建、运行、验证）
- `verify-image.sh`: 镜像验证脚本
- `PROGRAM_BUILD.md`: 程序内构建方式详细说明

### 文档
- `COMPARISON.md`: Rootless vs Privileged 模式详细对比

## 工作原理

### 程序内构建方式

1. **Go 程序** (`main.go`): 
   - 准备 Dockerfile（参考 crane_demo：将 main 复制到 `/usr/local/app/main`，设置工作目录和入口点）
   - 准备构建上下文（复制 main 文件）
   - 调用 Kaniko executor 构建镜像
2. **Kaniko Executor**: 在程序内通过 `exec.Command` 调用，构建镜像并推送到 registry
3. **安全配置**: 
   - `allowPrivilegeEscalation: false`
   - 不使用 `privileged: true`
   - 不需要 Docker 守护进程

### Job 方式（传统）

1. **initContainer** (`prepare-context`): 从 ConfigMap 复制文件到 `emptyDir:/workspace`
2. **Kaniko Executor**: 从 `/workspace` 读取 Dockerfile 和上下文，构建镜像并推送到 registry
3. **安全配置**: 
   - `allowPrivilegeEscalation: false`
   - 不使用 `privileged: true`
   - 不需要 Docker 守护进程

## 构建的镜像

### 程序内构建方式

- **镜像名称**: `registry.kube-system.svc.cluster.local:5000/new-kaniko-image:latest`
- **工作负载拉取**: `localhost:5000/new-kaniko-image:latest`
- **添加的文件**: `/usr/local/app/main`（参考 crane_demo）
- **工作目录**: `/usr/local/app`
- **入口点**: `/usr/local/app/main`

### Job 方式

- **镜像名称**: `registry.kube-system.svc.cluster.local:5000/new-kaniko-image:latest`
- **工作负载拉取**: `localhost:5000/new-kaniko-image:latest`
- **添加的文件**: `/opt/app/hello.txt`

## 清理资源

```bash
kubectl -n imgbuild delete job kaniko-addfile --ignore-not-found
kubectl -n imgbuild delete configmap kaniko-context --ignore-not-found
kubectl -n imgbuild delete pod verify-image --ignore-not-found
```

## 常见问题

### ImagePullBackOff

如果工作负载拉取镜像失败，尝试使用 `localhost:5000/...` 而不是 `registry.kube-system.svc.cluster.local:5000/...`

### 权限问题

如果遇到权限错误，可以在 `kaniko-job.yaml` 的 `args` 中添加：
```yaml
- --kaniko-dir=/tmp/kaniko
```

### 证书问题

已配置 `--skip-tls-verify` 和 `--skip-tls-verify-pull`，如有正式证书可移除这些参数。


