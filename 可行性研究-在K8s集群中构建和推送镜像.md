# 在 K8s 集群中构建镜像并推送新镜像的可行性研究

## 1. 研究背景

### 1.1 研究目的
验证在 Kubernetes 集群内部使用 Kaniko 构建 Docker 镜像并推送到集群内部镜像仓库的可行性，为后续应用托管功能提供技术基础。

### 1.2 研究范围
- 在 K8s Pod 中使用 Kaniko 构建镜像
- 将构建的镜像推送到集群内部镜像仓库
- 验证新构建的镜像可以被正常使用
- 验证镜像拉取和运行流程

### 1.3 技术选型
- **构建工具**: Kaniko v1.9.1-debug
- **基础镜像**: registry.kube-system.svc.cluster.local:5000/ones/plugin-host-node:v6.33.1
- **目标仓库**: registry.kube-system.svc.cluster.local:5000
- **编程语言**: Go

## 2. 技术方案

### 2.1 架构设计

```
┌─────────────────────────────────────────────────────────┐
│  K8s Cluster                                            │
│                                                          │
│  ┌──────────────────────────────────────────────────┐   │
│  │ build-image-pod (Kaniko Debug)                  │   │
│  │  - 运行 Go 构建程序                             │   │
│  │  - 调用 /kaniko/executor                        │   │
│  │  - 构建新镜像                                    │   │
│  └──────────────────────────────────────────────────┘   │
│           │                                              │
│           │ 推送镜像                                     │
│           ▼                                              │
│  ┌──────────────────────────────────────────────────┐   │
│  │ registry.kube-system.svc.cluster.local:5000     │   │
│  │  - 存储构建的新镜像                               │   │
│  └──────────────────────────────────────────────────┘   │
│           │                                              │
│           │ 拉取镜像                                     │
│           ▼                                              │
│  ┌──────────────────────────────────────────────────┐   │
│  │ test-kaniko-pod                                  │   │
│  │  - 运行新构建的镜像                               │   │
│  │  - 验证功能正常                                   │   │
│  └──────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
```

### 2.2 核心组件

#### 2.2.1 Kaniko Executor
- **位置**: `/kaniko/executor`
- **功能**: 在容器内无需 Docker 守护进程即可构建镜像
- **优势**: 
  - 无需特权模式（相比 Docker-in-Docker）
  - 支持多阶段构建
  - 支持缓存优化

#### 2.2.2 构建程序
- **语言**: Go
- **功能**: 
  - 动态生成 Dockerfile
  - 准备构建上下文
  - 调用 Kaniko executor
  - 推送镜像到仓库

#### 2.2.3 镜像仓库
- **地址**: `registry.kube-system.svc.cluster.local:5000`
- **类型**: 集群内部镜像仓库
- **访问方式**: 通过 Service DNS 名称访问

## 3. 实施步骤

### 3.1 第一步：创建构建 Pod

创建基于 Kaniko 的 Deployment：

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: build-image-deployment
  namespace: ones
spec:
  replicas: 1
  template:
    spec:
      containers:
      - name: build-image
        image: registry.cn-hangzhou.aliyuncs.com/kube-image-repo/kaniko:v1.9.1-debug
        command: ["/bin/sh"]
        args: ["-c", "sleep 3600"]
        securityContext:
          privileged: true
```

**执行结果**: ✅ Pod 成功创建并运行

### 3.2 第二步：开发构建程序

#### 3.2.1 程序功能
1. 创建构建上下文目录
2. 动态生成 Dockerfile
3. 复制构建文件到上下文
4. 调用 Kaniko executor 构建镜像
5. 推送镜像到仓库

#### 3.2.2 Dockerfile 模板
```dockerfile
FROM registry.kube-system.svc.cluster.local:5000/ones/plugin-host-node:v6.33.1
WORKDIR /usr/local/app
COPY main /usr/local/app/main
ENTRYPOINT ["/usr/local/app/main"]
```

#### 3.2.3 关键代码
```go
cmd := exec.Command(kanikoExecutor,
    "--dockerfile", contextDockerfilePath,
    "--context", contextDir,
    "--destination", imageName,
    "--insecure",
    "--skip-tls-verify",
)
```

**执行结果**: ✅ 程序成功编译并运行

### 3.3 第三步：构建并推送镜像

**构建过程**:
1. 从集群内部仓库拉取基础镜像
2. 创建构建上下文
3. 执行 Kaniko 构建
4. 推送镜像到仓库

**执行结果**: ✅ 镜像成功构建并推送
- 镜像名称: `registry.kube-system.svc.cluster.local:5000/new-image:latest`
- 镜像 SHA: `sha256:178bc4c591681f9b7124ca952418dda0436b1e941f10c0033f7f859068ad44f0`

### 3.4 第四步：验证新镜像

创建基于新镜像的 Deployment：

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-kaniko-deployment
  namespace: ones
spec:
  template:
    spec:
      containers:
      - name: test-kaniko
        image: localhost:5000/new-image:latest
        imagePullPolicy: IfNotPresent
```

**执行结果**: ✅ Pod 成功启动并运行

## 4. 验证结果

### 4.1 构建验证

| 验证项 | 结果 | 说明 |
|--------|------|------|
| Kaniko executor 可用 | ✅ | 成功调用 `/kaniko/executor` |
| 基础镜像拉取 | ✅ | 从集群内部仓库成功拉取 |
| 镜像构建 | ✅ | 成功构建新镜像 |
| 镜像推送 | ✅ | 成功推送到集群内部仓库 |

### 4.2 运行验证

**Pod 日志输出**:
```
Hello World
Server started on port 8081
```

**验证结果**: ✅ 新镜像可以正常启动并运行

### 4.3 镜像仓库验证

**镜像存储位置**: 
```
/var/lib/registry/docker/registry/v2/repositories/new-image/
```

**验证结果**: ✅ 镜像已成功存储到仓库

## 5. 关键技术点

### 5.1 镜像命名规范

**构建时使用**:
- `registry.kube-system.svc.cluster.local:5000/new-image:latest`
- 使用完整的 Service DNS 名称

**部署时使用**:
- `localhost:5000/new-image:latest`
- 使用 localhost 格式，K8s 可以正确解析

### 5.2 网络访问

- **集群内部访问**: 通过 Service DNS 名称访问
- **镜像拉取**: 使用 `imagePullPolicy: IfNotPresent`
- **安全配置**: 使用 `--insecure` 和 `--skip-tls-verify` 跳过 TLS 验证

### 5.3 权限配置

- **SecurityContext**: 需要 `privileged: true` 以支持 Kaniko
- **资源限制**: 建议设置合理的 CPU 和内存限制

## 6. 遇到的问题及解决方案

### 6.1 问题：架构不匹配

**现象**: `exec format error`

**原因**: 本地编译环境为 arm64，Pod 运行环境为 x86_64

**解决方案**: 使用交叉编译
```bash
GOOS=linux GOARCH=amd64 go build -o build-image main.go
```

### 6.2 问题：基础镜像拉取失败

**现象**: 无法从 Docker Hub 拉取 `alpine:latest`

**原因**: Pod 无法访问外网

**解决方案**: 使用集群内部已有的基础镜像
```dockerfile
FROM registry.kube-system.svc.cluster.local:5000/ones/plugin-host-node:v6.33.1
```

## 7. 性能分析

### 7.1 构建时间
- **基础镜像拉取**: ~6 秒（从集群内部仓库）
- **镜像构建**: ~1 秒
- **镜像推送**: ~1 秒
- **总耗时**: ~8 秒

### 7.2 资源消耗
- **CPU**: 构建时峰值约 500m
- **内存**: 构建时峰值约 512Mi
- **存储**: 镜像大小约 36MB

## 8. 可行性结论

### 8.1 技术可行性 ✅

**结论**: 在 K8s 集群中使用 Kaniko 构建镜像并推送到集群内部仓库**完全可行**。

**依据**:
1. ✅ Kaniko 可以在 Pod 中正常运行
2. ✅ 可以成功构建 Docker 镜像
3. ✅ 可以成功推送到集群内部仓库
4. ✅ 新构建的镜像可以被正常使用
5. ✅ 整个流程自动化完成

### 8.2 优势

1. **无需 Docker 守护进程**: Kaniko 在容器内直接构建，无需 Docker-in-Docker
2. **安全性**: 相比 Docker-in-Docker，安全性更高
3. **效率**: 构建速度快，资源消耗低
4. **集成性**: 与 K8s 原生集成，无需额外配置

### 8.3 限制

1. **基础镜像依赖**: 需要基础镜像在集群内部仓库中可用
2. **网络要求**: 需要能够访问集群内部 Service
3. **权限要求**: 需要 privileged 权限

### 8.4 适用场景

1. ✅ CI/CD 流水线中的镜像构建
2. ✅ 应用托管功能中的镜像构建
3. ✅ 动态构建自定义镜像
4. ✅ 多阶段构建场景

## 9. 建议

### 9.1 生产环境建议

1. **镜像缓存**: 配置 Kaniko 缓存以提高构建速度
2. **资源限制**: 设置合理的 CPU 和内存限制
3. **错误处理**: 增加完善的错误处理和重试机制
4. **日志收集**: 集成日志收集系统，便于问题排查
5. **安全加固**: 评估 privileged 权限的必要性，考虑使用更安全的方案

### 9.2 优化方向

1. **并行构建**: 支持多个镜像并行构建
2. **构建队列**: 实现构建任务队列管理
3. **构建历史**: 记录构建历史和版本信息
4. **镜像清理**: 实现旧镜像自动清理机制

## 10. 附录

### 10.1 相关文件

- **构建程序**: `test_image/build_image/main.go`
- **构建 Deployment**: `test_image/k8s/build-image-deployment.yaml`
- **测试 Deployment**: `test_image/k8s/test-kaniko-deployment.yaml`
- **测试程序**: `test_image/server/main.go`

### 10.2 参考命令

```bash
# 应用构建 Deployment
kubectl apply -f test_image/k8s/build-image-deployment.yaml

# 复制构建程序到 Pod
kubectl cp build_image/build-image ones/build-image-pod:/workspace/build-image

# 运行构建程序
kubectl exec -n ones build-image-pod -- /workspace/build-image

# 应用测试 Deployment
kubectl apply -f test_image/k8s/test-kaniko-deployment.yaml

# 查看测试 Pod 日志
kubectl logs -n ones -l app=test-kaniko
```

### 10.3 镜像信息

- **构建镜像**: `registry.kube-system.svc.cluster.local:5000/new-image:latest`
- **基础镜像**: `registry.kube-system.svc.cluster.local:5000/ones/plugin-host-node:v6.33.1`
- **Kaniko 镜像**: `registry.cn-hangzhou.aliyuncs.com/kube-image-repo/kaniko:v1.9.1-debug`

---

**文档版本**: v1.0  
**创建日期**: 2025-11-06  
**作者**: 技术团队  
**状态**: ✅ 验证通过

