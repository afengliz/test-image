# 实际环境测试总结

## 测试日期
2024-11-10

## 测试环境
- K8s 集群：`p8207-k3s-2.k3s-dev.myones.net`
- 命名空间：`imgbuild`
- 基础镜像：`registry.kube-system.svc.cluster.local:5000/ones/plugin-host-node:v6.33.1`

## 测试结果

### ✅ Job 方式测试 - 成功

**测试命令**：
```bash
kubectl apply -f kaniko-context.yaml
kubectl apply -f kaniko-job.yaml
kubectl -n imgbuild wait --for=condition=complete job/kaniko-addfile --timeout=5m
```

**结果**：
- ✅ Job 执行成功
- ✅ 镜像构建并推送成功：`registry.kube-system.svc.cluster.local:5000/new-kaniko-image:latest`
- ✅ 镜像内容验证成功（包含 `/opt/app/hello.txt`）

**日志关键信息**：
```
INFO[0006] Pushing image to registry.kube-system.svc.cluster.local:5000/new-kaniko-image:latest 
INFO[0006] Pushed registry.kube-system.svc.cluster.local:5000/new-kaniko-image@sha256:848447c0f736089a4ae229287b868002522a7e7cd586d89762fe02d3da352240
```

### ✅ 程序内构建方式测试 - 成功（使用 kubectl cp）

**测试方法**：使用 `kubectl cp` 将文件复制到 Pod 中，然后运行构建程序

**测试命令**：
```bash
# 1. 创建 Pod
kubectl apply -f kaniko-pod-simple.yaml

# 2. 等待 Pod 就绪
kubectl -n imgbuild wait --for=condition=Ready pod/kaniko-build-simple --timeout=60s

# 3. 复制文件
kubectl -n imgbuild cp main kaniko-build-simple:/workspace/main
kubectl -n imgbuild exec kaniko-build-simple -- mkdir -p /workspace/server
kubectl -n imgbuild cp ../demo_server/main kaniko-build-simple:/workspace/server/main

# 4. 运行构建程序
kubectl -n imgbuild exec kaniko-build-simple -- chmod +x /workspace/main
kubectl -n imgbuild exec kaniko-build-simple -- /workspace/main
```

**结果**：
- ✅ Pod 创建成功
- ✅ 文件复制成功（main: 2.6MB, server/main: 7.8MB）
- ✅ 镜像构建并推送成功：`registry.kube-system.svc.cluster.local:5000/new-kaniko-image:latest`
- ✅ 镜像内容验证成功（包含 `/usr/local/app/main`，参考 crane_demo 结构）

**日志关键信息**：
```
=== 使用 Kaniko 在程序内构建镜像 ===
基础镜像: registry.kube-system.svc.cluster.local:5000/ones/plugin-host-node:v6.33.1
源文件: /workspace/server/main
目标镜像: registry.kube-system.svc.cluster.local:5000/new-kaniko-image:latest
✓ Dockerfile 创建成功
✓ 复制文件: /workspace/server/main -> /tmp/kaniko-build/build-context/main
INFO[0007] Pushing image to registry.kube-system.svc.cluster.local:5000/new-kaniko-image:latest 
INFO[0007] Pushed registry.kube-system.svc.cluster.local:5000/new-kaniko-image@sha256:1b0d63308eb5b3701892337b7c5c22a38b6b7240e650d1fdce80a673a0e1ec0b 
✓ 镜像构建并推送成功
```

**自动化测试脚本**：
```bash
./test-program-copy.sh
```

## 解决方案

### 方案 1：Job 方式（已验证可行）✅

**优点**：
- ✅ 已验证成功
- ✅ 适合小文件场景（<3MB）
- ✅ 无需推送镜像
- ✅ 配置简单

**缺点**：
- ⚠️ ConfigMap 有大小限制（~3MB）

**使用方式**：
```bash
kubectl apply -f kaniko-context.yaml
kubectl apply -f kaniko-job.yaml
kubectl -n imgbuild logs job/kaniko-addfile
```

### 方案 2：程序内构建 + kubectl cp（已验证可行）✅⭐

**步骤**：

1. **构建 Go 程序**：
```bash
make build
```

2. **创建 Pod**：
```bash
kubectl apply -f kaniko-pod-simple.yaml
kubectl -n imgbuild wait --for=condition=Ready pod/kaniko-build-simple --timeout=60s
```

3. **复制文件并运行**：
```bash
# 复制构建程序
kubectl -n imgbuild cp main kaniko-build-simple:/workspace/main

# 复制 server/main
kubectl -n imgbuild exec kaniko-build-simple -- mkdir -p /workspace/server
kubectl -n imgbuild cp ../demo_server/main kaniko-build-simple:/workspace/server/main

# 运行构建程序
kubectl -n imgbuild exec kaniko-build-simple -- chmod +x /workspace/main
kubectl -n imgbuild exec kaniko-build-simple -- /workspace/main
```

**或使用自动化脚本**：
```bash
./test-program-copy.sh
```

**优点**：
- ✅ 已验证成功
- ✅ 支持大文件（无大小限制）
- ✅ 无需预先构建镜像
- ✅ 灵活，适合动态场景

**缺点**：
- ⚠️ 需要手动复制文件（可用脚本自动化）

### 方案 3：程序内构建 + 镜像包含文件

**步骤**：

1. **构建镜像**（包含所有文件）：
```bash
# 复制文件到构建上下文
cp ../demo_server/main server-main

# 构建镜像
make build-image
```

2. **推送到 registry**（需要配置）：
```bash
# 推送到集群 registry
docker tag kaniko-build:latest <registry>/kaniko-build:latest
docker push <registry>/kaniko-build:latest
```

3. **运行 Pod**：
```bash
kubectl apply -f kaniko-pod.yaml
```

**优点**：
- ✅ 支持大文件
- ✅ 无需运行时挂载
- ✅ 更灵活

**缺点**：
- ⚠️ 需要配置镜像推送或加载

## 最终建议

### 对于小文件（<3MB）
使用 **Job 方式**（方案 1），已验证可行。

### 对于大文件（>3MB）
使用 **程序内构建 + kubectl cp**（方案 2），已验证可行：
1. 构建 Go 程序（`make build`）
2. 创建 Pod（`kubectl apply -f kaniko-pod-simple.yaml`）
3. 复制文件并运行（使用 `./test-program-copy.sh` 自动化）

## 测试验证状态

| 测试项 | 状态 | 说明 |
|--------|------|------|
| Go 程序编译 | ✅ 成功 | 生成 2.6MB 二进制 |
| Docker 镜像构建 | ✅ 成功 | 包含所有文件 |
| Job 方式测试 | ✅ 成功 | 已验证可行 |
| 程序内构建（kubectl cp） | ✅ 成功 | 已验证可行 |
| 程序内构建（镜像方式） | ⚠️ 需配置 | 镜像已构建，需推送/加载 |
| ConfigMap 方式 | ❌ 不可行 | 文件太大（>3MB） |
| hostPath 方式 | ❌ 不可行 | 路径在节点上不存在 |

## 结论

1. **Job 方式已验证成功** ✅
   - 适合小文件场景（<3MB）
   - 配置简单，无需额外配置

2. **程序内构建方式已验证成功** ✅
   - 代码编译成功
   - 使用 `kubectl cp` 方式已验证可行
   - 支持大文件（无大小限制）
   - 镜像构建并推送成功

3. **推荐使用方式**：
   - **小文件（<3MB）**：Job 方式（已验证）
   - **大文件（>3MB）**：程序内构建 + kubectl cp（已验证）

**所有代码和配置都已就绪，两种方式均已验证可行！** ✅

