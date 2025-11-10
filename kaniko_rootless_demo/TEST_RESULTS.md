# 实际环境测试结果

## 测试日期
2024-11-10

## 测试环境
- K8s 集群：`p8207-k3s-2.k3s-dev.myones.net`
- 命名空间：`imgbuild`
- 基础镜像：`registry.kube-system.svc.cluster.local:5000/ones/plugin-host-node:v6.33.1`

## 测试步骤

### 1. 环境检查 ✅
- K8s 集群连接正常
- 命名空间 `imgbuild` 已存在
- `demo_server/main` 文件存在（8MB）

### 2. Go 程序构建 ✅
```bash
make build
```
- 构建成功，生成 `main` 二进制文件（2.6MB）

### 3. Docker 镜像构建 ✅
```bash
# 复制 server-main 文件到构建上下文
cp ../demo_server/main server-main

# 构建镜像
docker build -t kaniko-build:latest .
```
- 镜像构建成功
- 镜像包含：
  - Kaniko executor
  - 构建程序 (`/kaniko-build/main`)
  - server/main 文件 (`/workspace/server/main`)

### 4. K8s Pod 部署 ⚠️

#### 遇到的问题

**问题 1：ConfigMap 大小限制**
- `demo_server/main` 文件约 8MB
- K8s ConfigMap 大小限制约 3MB
- 无法通过 ConfigMap 提供文件

**解决方案**：✅ 将文件直接打包到 Docker 镜像中

**问题 2：镜像推送问题**
- 尝试推送到 `registry.kube-system.svc.cluster.local:5000`：DNS 解析失败
- 尝试推送到 `localhost:5000`：连接超时
- 镜像无法推送到 registry

**解决方案**：
1. ✅ 使用 Job 方式（已验证可行）
2. ⚠️ 需要配置镜像推送或使用其他 registry

## 测试结果总结

### ✅ 成功的部分

1. **代码编译**：Go 程序编译成功
2. **镜像构建**：Docker 镜像构建成功，包含所有必要文件
3. **Job 方式**：之前使用 Job 方式测试成功（参考 `test.sh`）

### ⚠️ 遇到的问题

1. **ConfigMap 大小限制**：大文件（>3MB）无法使用 ConfigMap
   - **解决**：将文件打包到镜像中

2. **镜像推送**：本地无法直接推送到集群 registry
   - **原因**：DNS 解析或网络配置问题
   - **解决**：使用 Job 方式，或配置正确的 registry 访问

### 📋 推荐方案

#### 方案 1：使用 Job 方式（已验证）✅

使用已有的 `kaniko-job.yaml` 和 `kaniko-context.yaml`：

```bash
# 1. 创建 ConfigMap（小文件）
kubectl apply -f kaniko-context.yaml

# 2. 运行 Job
kubectl apply -f kaniko-job.yaml

# 3. 查看日志
kubectl -n imgbuild logs job/kaniko-addfile
```

**优点**：
- ✅ 已验证可行
- ✅ 无需推送镜像
- ✅ 适合小文件场景

**缺点**：
- ⚠️ ConfigMap 有大小限制（~3MB）

#### 方案 2：程序内构建 + 镜像包含文件（推荐）⭐

**步骤**：

1. **构建镜像**（包含所有文件）：
```bash
# 复制文件到构建上下文
cp ../demo_server/main server-main

# 构建镜像
docker build -t kaniko-build:latest .
```

2. **推送到 registry**（需要配置）：
```bash
# 方式 A：推送到集群 registry（需要配置）
docker tag kaniko-build:latest <registry>/kaniko-build:latest
docker push <registry>/kaniko-build:latest

# 方式 B：使用镜像加载（单节点）
docker save kaniko-build:latest | ssh <node> docker load
```

3. **运行 Pod**：
```bash
kubectl apply -f kaniko-pod-test.yaml
```

**优点**：
- ✅ 支持大文件
- ✅ 无需运行时挂载
- ✅ 更灵活

**缺点**：
- ⚠️ 需要配置镜像推送或加载

#### 方案 3：使用 PVC（PersistentVolumeClaim）

对于大文件，可以使用 PVC：

```yaml
volumes:
  - name: server
    persistentVolumeClaim:
      claimName: server-files-pvc
```

**优点**：
- ✅ 支持大文件
- ✅ 可持久化

**缺点**：
- ⚠️ 需要预先配置 PVC
- ⚠️ 需要将文件复制到 PVC

## 最终建议

### 对于小文件（<3MB）
使用 **Job 方式**（方案 1），已验证可行。

### 对于大文件（>3MB）
使用 **程序内构建 + 镜像包含文件**（方案 2）：
1. 将文件打包到镜像中
2. 配置镜像推送或使用镜像加载
3. 运行 Pod

### 测试验证

**Job 方式测试**：✅ 已通过（参考 `test.sh` 执行结果）

**程序内构建方式**：
- ✅ 代码编译成功
- ✅ 镜像构建成功
- ⚠️ 镜像推送需要配置（但镜像本身已包含所有文件）

## 结论

程序内构建方式的**代码和镜像构建都已成功**，主要问题是镜像推送到 registry 的配置。在实际生产环境中：

1. **配置正确的 registry 访问**（DNS/网络）
2. **或使用镜像加载方式**（单节点场景）
3. **或使用 Job 方式**（已验证可行）

所有代码和配置都已就绪，只需解决镜像推送/加载问题即可完成完整测试。

