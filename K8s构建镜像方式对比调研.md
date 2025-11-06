# K8s 集群中构建镜像方式对比调研

## 概述

在 Kubernetes 集群中构建容器镜像有多种方式，每种方式都有其特点和适用场景。本文档对主流的构建方式进行了详细对比。

## 重要概念说明

### Privileged（特权模式）vs 非特权模式

在 Kubernetes 中，**Privileged（特权模式）**和**非特权模式**是容器安全性的重要概念：

#### 1. Privileged（特权模式）

**定义**: 特权模式允许容器访问主机的所有设备和内核功能，几乎拥有与主机 root 用户相同的权限。

**配置方式**:
```yaml
apiVersion: v1
kind: Pod
spec:
  containers:
  - name: example
    securityContext:
      privileged: true  # 启用特权模式
```

**特点**:
- ✅ 可以访问主机的所有设备（如 `/dev`）
- ✅ 可以修改内核参数
- ✅ 可以挂载主机文件系统
- ✅ 可以执行需要特殊权限的操作
- 🔴 **安全风险高**：容器逃逸后可能影响整个节点
- 🔴 违反最小权限原则

**使用场景**:
- Docker-in-Docker (DinD) 需要特权模式来运行 Docker 守护进程
- 某些系统级工具（如网络工具、存储工具）
- 开发/测试环境

**示例**:
```yaml
# 我们的 Kaniko 构建 Pod 配置
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - name: build-image
        securityContext:
          privileged: true  # Kaniko 默认需要特权模式
```

#### 2. 非特权模式（Non-Privileged）

**定义**: 非特权模式是容器的默认模式，容器运行在受限的环境中，只能访问被明确授予的资源。

**配置方式**:
```yaml
apiVersion: v1
kind: Pod
spec:
  containers:
  - name: example
    securityContext:
      privileged: false  # 非特权模式（默认值）
      # 或者不设置 privileged 字段
```

**特点**:
- ✅ **安全性高**：即使容器被攻破，影响范围有限
- ✅ 符合最小权限原则
- ✅ 适合生产环境
- ⚠️ 某些操作可能受限（如访问设备、修改内核参数）

**使用场景**:
- 生产环境应用
- 大多数业务容器
- 安全要求高的场景

**示例**:
```yaml
# 普通应用 Pod（非特权模式）
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - name: app
        securityContext:
          runAsNonRoot: true  # 以非 root 用户运行
          allowPrivilegeEscalation: false  # 禁止权限提升
          # privileged 默认为 false
```

#### 3. 权限对比表

| 特性 | Privileged（特权模式） | 非特权模式 |
|------|----------------------|-----------|
| **访问主机设备** | ✅ 完全访问 | ❌ 受限访问 |
| **修改内核参数** | ✅ 可以 | ❌ 不可以 |
| **挂载主机文件系统** | ✅ 可以 | ❌ 受限 |
| **运行 Docker 守护进程** | ✅ 可以 | ❌ 不可以 |
| **安全风险** | 🔴 高 | 🟢 低 |
| **适用环境** | 开发/测试 | 生产环境 |
| **容器逃逸影响** | 🔴 影响整个节点 | 🟢 影响范围有限 |

#### 4. 在构建镜像场景中的应用

**Kaniko**:
- **默认**: 需要 `privileged: true`
- **原因**: 需要访问某些系统功能来构建镜像
- **改进**: 可以通过特殊配置在非特权模式下运行（需要额外的安全配置）

**Docker-in-Docker**:
- **必须**: `privileged: true`
- **原因**: 需要运行 Docker 守护进程，必须访问主机设备

**Buildah**:
- **支持**: 可以在非特权模式下运行（rootless 模式）
- **优势**: 安全性更高

**BuildKit**:
- **支持**: 可以在非特权模式下运行
- **优势**: 适合生产环境

#### 5. 安全建议

1. **生产环境**: 优先使用非特权模式
2. **开发环境**: 可以使用特权模式，但要注意安全
3. **最小权限原则**: 只授予必要的权限
4. **定期审查**: 检查哪些 Pod 使用了特权模式，评估必要性

#### 6. 实际配置示例

**特权模式示例**（我们的 Kaniko 构建）:
```yaml
securityContext:
  privileged: true  # 需要特权模式
```

**非特权模式示例**（推荐的生产配置）:
```yaml
securityContext:
  privileged: false  # 非特权模式
  runAsNonRoot: true  # 以非 root 用户运行
  allowPrivilegeEscalation: false  # 禁止权限提升
  capabilities:
    drop:
    - ALL  # 删除所有 capabilities
    add:
    - NET_BIND_SERVICE  # 只添加必要的 capabilities
```

## 构建方式对比表

| 构建方式 | 是否需要 Docker 守护进程 | 权限要求 | 安全性 | 构建速度 | 资源消耗 | 易用性 | 多阶段构建 | 缓存支持 | **API 支持** | 适用场景 | 社区支持 |
|---------|----------------------|---------|--------|---------|---------|--------|-----------|---------|------------|---------|---------|
| **Kaniko** | ❌ 否 | Privileged 或非特权 | 🟢 高 | 🟡 中等 | 🟢 低 (~100-200MB) | 🟢 简单 | ✅ 支持 | ✅ 支持 | 🟡 通过 K8s API | K8s 集群内构建、CI/CD | 🟢 活跃 |
| **Docker-in-Docker (DinD)** | ✅ 是 | Privileged | 🔴 低 | 🟢 快 | 🔴 高 (~500MB+) | 🟢 简单 | ✅ 支持 | ✅ 支持 | 🟢 Docker API | 开发环境、测试 | 🟢 广泛 |
| **Buildah** | ❌ 否 | Rootless 支持 | 🟢 高 | 🟡 中等 | 🟢 低 (~50-100MB) | 🟡 中等 | ✅ 支持 | ✅ 支持 | 🟢 Go API | 安全要求高的环境 | 🟡 中等 |
| **BuildKit** | ❌ 否（独立守护进程） | 非特权 | 🟢 高 | 🟢 快 | 🟡 中等 (~200MB) | 🟡 中等 | ✅ 支持 | ✅ 高级缓存 | 🟢 gRPC API | 生产环境、大规模构建 | 🟢 活跃 |
| **Tekton** | 取决于底层工具 | 取决于底层工具 | 🟢 高 | 🟡 中等 | 🟡 中等 | 🔴 复杂 | ✅ 支持 | ✅ 支持 | 🟢 K8s API | 企业级 CI/CD | 🟢 活跃 |
| **Skaffold** | 取决于底层工具 | 取决于底层工具 | 🟢 高 | 🟢 快 | 🟡 中等 | 🟢 简单 | ✅ 支持 | ✅ 支持 | 🟡 CLI/API | 开发迭代、本地构建 | 🟢 活跃 |
| **Jib** | ❌ 否 | 无特殊要求 | 🟢 高 | 🟢 快 | 🟢 低 | 🟢 简单 | ✅ 支持 | ✅ 增量构建 | 🟢 Java API | Java 应用专用 | 🟢 活跃 |
| **img** | ❌ 否 | 非特权 | 🟢 高 | 🟡 中等 | 🟢 低 | 🟡 中等 | ✅ 支持 | ✅ 支持 | 🔴 仅 CLI | 轻量级构建 | 🟡 较少 |

## API vs 命令行方式

### 概述

构建镜像的方式可以分为两类：
1. **命令行方式**：直接执行命令（如 `docker build`、`kaniko executor`）
2. **API 方式**：通过编程接口调用（如 Docker SDK、K8s API、Buildah Go API）

### API 方式详细说明

#### 1. Docker SDK/API ⭐⭐⭐⭐⭐

**支持语言**: Go, Python, Java, Node.js 等

**Go 示例**:
```go
package main

import (
    "context"
    "github.com/docker/docker/api/types"
    "github.com/docker/docker/client"
    "github.com/docker/docker/pkg/archive"
)

func buildImage() error {
    cli, err := client.NewClientWithOpts(client.FromEnv)
    if err != nil {
        return err
    }
    defer cli.Close()

    ctx := context.Background()
    
    // 创建构建上下文
    buildContext, err := archive.TarWithOptions(".", &archive.TarOptions{})
    if err != nil {
        return err
    }

    // 构建镜像
    response, err := cli.ImageBuild(ctx, buildContext, types.ImageBuildOptions{
        Dockerfile: "Dockerfile",
        Tags:       []string{"my-image:latest"},
    })
    if err != nil {
        return err
    }
    defer response.Body.Close()

    // 读取构建输出
    // ... 处理响应
    
    return nil
}
```

**优势**:
- ✅ 完全编程化，无需命令行
- ✅ 支持多种语言
- ✅ 可以实时获取构建进度
- ✅ 错误处理更灵活

**劣势**:
- ❌ 需要 Docker 守护进程
- ❌ 需要网络连接到 Docker daemon

#### 2. Kubernetes API ⭐⭐⭐⭐

**方式**: 通过 K8s API 创建 Pod/Job 来运行构建工具

**Go 示例** (使用 client-go):
```go
package main

import (
    "context"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/tools/clientcmd"
    batchv1 "k8s.io/api/batch/v1"
    corev1 "k8s.io/api/core/v1"
)

func buildImageWithKaniko() error {
    // 创建 K8s 客户端
    config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
    if err != nil {
        return err
    }
    clientset, err := kubernetes.NewForConfig(config)
    if err != nil {
        return err
    }

    // 创建 Job 来运行 Kaniko
    job := &batchv1.Job{
        ObjectMeta: metav1.ObjectMeta{
            Name: "kaniko-build",
        },
        Spec: batchv1.JobSpec{
            Template: corev1.PodTemplateSpec{
                Spec: corev1.PodSpec{
                    Containers: []corev1.Container{
                        {
                            Name:  "kaniko",
                            Image: "gcr.io/kaniko-project/executor:latest",
                            Args: []string{
                                "--dockerfile=Dockerfile",
                                "--context=.",
                                "--destination=registry.example.com/image:tag",
                            },
                        },
                    },
                    RestartPolicy: corev1.RestartPolicyNever,
                },
            },
        },
    }

    // 创建 Job
    _, err = clientset.BatchV1().Jobs("default").Create(context.TODO(), job, metav1.CreateOptions{})
    return err
}
```

**优势**:
- ✅ 完全编程化
- ✅ 可以管理构建任务的生命周期
- ✅ 支持异步构建
- ✅ 可以监控构建状态

**劣势**:
- ⚠️ 需要 K8s 集群访问权限
- ⚠️ 配置相对复杂

#### 3. Buildah Go API ⭐⭐⭐⭐

**Go 示例**:
```go
package main

import (
    "github.com/containers/buildah"
    "github.com/containers/storage/pkg/unshare"
)

func buildImageWithBuildah() error {
    // 初始化 Buildah
    store, err := buildah.GetStore(buildah.StoreOptions{})
    if err != nil {
        return err
    }
    defer store.Shutdown()

    // 创建构建选项
    options := buildah.BuilderOptions{
        FromImage: "alpine:latest",
    }

    // 创建构建器
    builder, err := buildah.NewBuilder(store, options)
    if err != nil {
        return err
    }
    defer builder.Delete()

    // 执行构建步骤
    // ... 添加文件、运行命令等

    // 提交镜像
    imageID, err := builder.Commit("my-image:latest", buildah.CommitOptions{})
    return err
}
```

**优势**:
- ✅ 完全编程化
- ✅ 无需 Docker 守护进程
- ✅ 支持 rootless 模式

**劣势**:
- ⚠️ 学习曲线较陡
- ⚠️ 文档相对较少

#### 4. BuildKit gRPC API ⭐⭐⭐⭐

**方式**: 通过 gRPC 调用 BuildKit

**Go 示例**:
```go
package main

import (
    "github.com/moby/buildkit/client"
    "github.com/moby/buildkit/session"
)

func buildImageWithBuildKit() error {
    // 连接到 BuildKit
    c, err := client.New(context.Background(), "unix:///run/buildkit/buildkitd.sock")
    if err != nil {
        return err
    }
    defer c.Close()

    // 创建构建会话
    sess, err := session.NewSession(context.Background(), "buildkit-client", "")
    if err != nil {
        return err
    }

    // 定义构建步骤
    // ... 使用 BuildKit 的 LLB (Low-Level Builder) API

    // 执行构建
    // ...
    
    return nil
}
```

**优势**:
- ✅ 高性能
- ✅ 支持并行构建
- ✅ 高级缓存机制

**劣势**:
- ⚠️ API 相对复杂
- ⚠️ 需要 BuildKit 守护进程

#### 5. Jib Java API ⭐⭐⭐⭐⭐

**Java 示例**:
```java
import com.google.cloud.tools.jib.api.Containerizer;
import com.google.cloud.tools.jib.api.Jib;
import com.google.cloud.tools.jib.api.RegistryImage;

public class BuildImage {
    public static void main(String[] args) throws Exception {
        RegistryImage targetImage = RegistryImage.named("registry.example.com/image:tag");
        
        Containerizer containerizer = Containerizer.to(targetImage)
            .setCredentialRetrievers(...)
            .build();

        Jib.from("openjdk:11-jre-slim")
            .addLayer(Paths.get("target/my-app.jar"), "/app")
            .setEntrypoint("java", "-jar", "/app/my-app.jar")
            .containerize(containerizer);
    }
}
```

**优势**:
- ✅ 完全编程化
- ✅ 无需 Dockerfile
- ✅ 增量构建

**劣势**:
- ❌ 仅支持 Java

### API vs 命令行对比

| 特性 | API 方式 | 命令行方式 |
|------|---------|-----------|
| **编程化** | ✅ 完全支持 | ❌ 需要执行命令 |
| **错误处理** | ✅ 结构化错误 | ⚠️ 需要解析输出 |
| **进度监控** | ✅ 实时回调 | ⚠️ 需要解析日志 |
| **集成性** | ✅ 易于集成 | ⚠️ 需要进程管理 |
| **学习成本** | 🔴 较高 | 🟢 较低 |
| **灵活性** | ✅ 高 | 🟡 中等 |

### 推荐方案

**如果需要在代码中构建镜像**:
1. **有 Docker 环境**: 使用 **Docker SDK** (Go/Python/Java)
2. **K8s 集群内**: 使用 **Kubernetes API** + Kaniko/Buildah
3. **Java 应用**: 使用 **Jib API**
4. **高性能需求**: 使用 **BuildKit gRPC API**

**如果只是简单构建**:
- 使用命令行方式更简单直接

## 详细说明

### 1. Kaniko

**描述**: Google 开源的容器镜像构建工具，在容器内无需 Docker 守护进程即可构建镜像。

**特点**:
- ✅ 无需 Docker 守护进程
- ✅ 支持多阶段构建
- ✅ 支持缓存优化
- ✅ 可在非特权容器中运行（需要特殊配置）
- ⚠️ 默认需要 privileged 权限

**使用示例**:
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: kaniko-build
spec:
  containers:
  - name: kaniko
    image: gcr.io/kaniko-project/executor:latest
    args:
    - --dockerfile=Dockerfile
    - --context=.
    - --destination=registry.example.com/image:tag
    volumeMounts:
    - name: dockerfile
      mountPath: /workspace
  volumes:
  - name: dockerfile
    configMap:
      name: dockerfile-config
```

**优势**:
- 安全性高，适合在 K8s 集群内构建
- 与 K8s 原生集成
- 支持缓存层，提升构建速度

**劣势**:
- 构建速度相对较慢
- 需要 privileged 权限（或特殊配置）
- 对复杂 Dockerfile 支持有限

**适用场景**:
- ✅ K8s 集群内构建镜像
- ✅ CI/CD 流水线
- ✅ 安全要求高的环境

---

### 2. Docker-in-Docker (DinD)

**描述**: 在容器内运行 Docker 守护进程，使用 Docker CLI 构建镜像。

**特点**:
- ✅ 使用标准 Docker 命令
- ✅ 构建速度快
- ✅ 功能完整
- ❌ 需要 privileged 权限
- ❌ 安全风险较高

**使用示例**:
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: dind-build
spec:
  containers:
  - name: docker
    image: docker:dind
    securityContext:
      privileged: true
    volumeMounts:
    - name: docker-storage
      mountPath: /var/lib/docker
  volumes:
  - name: docker-storage
    emptyDir: {}
```

**优势**:
- 使用广泛，文档丰富
- 构建速度快
- 功能完整，支持所有 Docker 特性

**劣势**:
- 安全风险高（需要 privileged）
- 资源消耗大
- 不适合生产环境

**适用场景**:
- ✅ 开发环境
- ✅ 测试环境
- ❌ 不推荐生产环境

---

### 3. Buildah

**描述**: Red Hat 开发的无守护进程容器镜像构建工具。

**特点**:
- ✅ 无需 Docker 守护进程
- ✅ 支持 rootless 模式
- ✅ 安全性高
- ✅ 资源消耗低
- ⚠️ 学习曲线较陡

**使用示例**:
```bash
buildah bud -f Dockerfile -t registry.example.com/image:tag .
buildah push registry.example.com/image:tag
```

**优势**:
- 安全性高，支持 rootless
- 资源消耗低
- 灵活性高

**劣势**:
- 学习曲线较陡
- 社区支持相对较少
- 配置相对复杂

**适用场景**:
- ✅ 安全要求高的环境
- ✅ 需要高度定制化的构建流程
- ✅ 生产环境

---

### 4. BuildKit

**描述**: Docker 的新一代构建引擎，支持并行构建和高级缓存。

**特点**:
- ✅ 构建速度快
- ✅ 支持并行构建
- ✅ 高级缓存机制
- ✅ 支持多架构构建
- ⚠️ 配置相对复杂

**使用示例**:
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: buildkit-build
spec:
  containers:
  - name: buildkitd
    image: moby/buildkit:latest
    args:
    - --addr
    - unix:///run/buildkit/buildkitd.sock
```

**优势**:
- 构建速度快
- 支持并行构建
- 高级缓存机制
- 支持多架构

**劣势**:
- 配置相对复杂
- 需要额外的守护进程
- 学习成本较高

**适用场景**:
- ✅ 生产环境
- ✅ 大规模构建
- ✅ 需要高性能的场景

---

### 5. Tekton

**描述**: Kubernetes 原生的 CI/CD 框架，支持多种构建工具。

**特点**:
- ✅ K8s 原生
- ✅ 支持多种构建工具
- ✅ 可扩展性强
- ⚠️ 配置复杂
- ⚠️ 学习曲线陡

**使用示例**:
```yaml
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: build-image
spec:
  steps:
  - name: build
    image: gcr.io/kaniko-project/executor:latest
    args:
    - --dockerfile=Dockerfile
    - --context=.
    - --destination=registry.example.com/image:tag
```

**优势**:
- K8s 原生，集成度高
- 支持复杂的 CI/CD 流程
- 可扩展性强

**劣势**:
- 配置复杂
- 学习曲线陡
- 资源消耗相对较高

**适用场景**:
- ✅ 企业级 CI/CD
- ✅ 复杂的构建流程
- ✅ 需要高度定制化的场景

---

### 6. Skaffold

**描述**: Google 开发的 Kubernetes 应用开发工具，支持多种构建方式。

**特点**:
- ✅ 开发体验友好
- ✅ 支持热重载
- ✅ 支持多种构建工具
- ⚠️ 主要用于开发环境

**使用示例**:
```yaml
apiVersion: skaffold/v2beta26
kind: Config
build:
  kaniko:
    buildContext:
      localDir: {}
    push: true
```

**优势**:
- 开发体验好
- 支持快速迭代
- 配置简单

**劣势**:
- 主要用于开发环境
- 生产环境使用较少

**适用场景**:
- ✅ 开发环境
- ✅ 快速迭代
- ✅ 本地构建

---

### 7. Jib

**描述**: Google 开发的 Java 应用容器化工具，无需 Dockerfile。

**特点**:
- ✅ 无需 Dockerfile
- ✅ 增量构建
- ✅ 构建速度快
- ❌ 仅支持 Java 应用

**使用示例**:
```xml
<plugin>
  <groupId>com.google.cloud.tools</groupId>
  <artifactId>jib-maven-plugin</artifactId>
  <version>3.3.0</version>
  <configuration>
    <to>
      <image>registry.example.com/image:tag</image>
    </to>
  </configuration>
</plugin>
```

**优势**:
- 无需 Dockerfile
- 增量构建，速度快
- 配置简单

**劣势**:
- 仅支持 Java 应用
- 功能相对有限

**适用场景**:
- ✅ Java 应用容器化
- ✅ 需要快速构建的场景

---

### 8. img

**描述**: 无守护进程的容器镜像构建工具，使用 BuildKit 后端。

**特点**:
- ✅ 无需守护进程
- ✅ 支持非特权运行
- ✅ 资源消耗低
- ⚠️ 社区支持较少

**使用示例**:
```bash
img build -t registry.example.com/image:tag .
img push registry.example.com/image:tag
```

**优势**:
- 轻量级
- 安全性高
- 资源消耗低

**劣势**:
- 社区支持较少
- 功能相对有限

**适用场景**:
- ✅ 轻量级构建需求
- ✅ 资源受限的环境

---

## 综合对比总结

### 按场景选择

| 场景 | 推荐方案 | 理由 |
|------|---------|------|
| **K8s 集群内构建** | Kaniko | 安全性高，与 K8s 集成好 |
| **开发环境** | Docker-in-Docker 或 Skaffold | 使用简单，构建快速 |
| **生产环境** | Kaniko 或 BuildKit | 安全性高，性能好 |
| **Java 应用** | Jib | 专用工具，构建快速 |
| **企业级 CI/CD** | Tekton | 功能完整，可扩展性强 |
| **安全要求高** | Buildah 或 Kaniko | 支持 rootless，安全性高 |
| **快速迭代** | Skaffold | 开发体验好，支持热重载 |

### 性能对比

| 构建方式 | 构建速度 | 资源消耗 | 缓存效率 |
|---------|---------|---------|---------|
| **Kaniko** | 🟡 中等 | 🟢 低 | 🟢 高 |
| **Docker-in-Docker** | 🟢 快 | 🔴 高 | 🟢 高 |
| **Buildah** | 🟡 中等 | 🟢 低 | 🟡 中等 |
| **BuildKit** | 🟢 快 | 🟡 中等 | 🟢 很高 |
| **Tekton** | 🟡 中等 | 🟡 中等 | 🟢 高 |
| **Skaffold** | 🟢 快 | 🟡 中等 | 🟢 高 |
| **Jib** | 🟢 快 | 🟢 低 | 🟢 很高（增量） |
| **img** | 🟡 中等 | 🟢 低 | 🟡 中等 |

### 安全性对比

| 构建方式 | 权限要求 | 安全风险 | 推荐度 |
|---------|---------|---------|--------|
| **Kaniko** | Privileged（可配置非特权） | 🟢 低 | ⭐⭐⭐⭐⭐ |
| **Docker-in-Docker** | Privileged | 🔴 高 | ⭐⭐ |
| **Buildah** | Rootless 支持 | 🟢 低 | ⭐⭐⭐⭐⭐ |
| **BuildKit** | 非特权 | 🟢 低 | ⭐⭐⭐⭐ |
| **Tekton** | 取决于底层工具 | 🟢 低 | ⭐⭐⭐⭐ |
| **Skaffold** | 取决于底层工具 | 🟢 低 | ⭐⭐⭐⭐ |
| **Jib** | 无特殊要求 | 🟢 低 | ⭐⭐⭐⭐⭐ |
| **img** | 非特权 | 🟢 低 | ⭐⭐⭐⭐ |

## 推荐方案

### 1. 生产环境推荐：Kaniko ⭐⭐⭐⭐⭐

**理由**:
- ✅ 安全性高，适合 K8s 集群内构建
- ✅ 无需 Docker 守护进程
- ✅ 支持缓存优化
- ✅ 社区活跃，文档完善

**适用场景**:
- K8s 集群内构建镜像
- CI/CD 流水线
- 安全要求高的环境

### 2. 开发环境推荐：Docker-in-Docker 或 Skaffold ⭐⭐⭐⭐

**理由**:
- ✅ 使用简单，学习成本低
- ✅ 构建速度快
- ✅ 功能完整

**适用场景**:
- 开发环境
- 测试环境
- 快速迭代

### 3. Java 应用推荐：Jib ⭐⭐⭐⭐⭐

**理由**:
- ✅ 无需 Dockerfile
- ✅ 增量构建，速度快
- ✅ 配置简单

**适用场景**:
- Java 应用容器化
- 需要快速构建的场景

### 4. 企业级 CI/CD 推荐：Tekton ⭐⭐⭐⭐

**理由**:
- ✅ K8s 原生，集成度高
- ✅ 支持复杂的构建流程
- ✅ 可扩展性强

**适用场景**:
- 企业级 CI/CD
- 复杂的构建流程
- 需要高度定制化的场景

## 结论

在 K8s 集群中构建镜像，**Kaniko** 是最推荐的方案，因为：

1. ✅ **安全性高**：无需 Docker 守护进程，支持非特权运行
2. ✅ **K8s 原生**：与 K8s 集成好，易于使用
3. ✅ **性能良好**：支持缓存优化，构建速度可接受
4. ✅ **社区活跃**：文档完善，问题解决及时

对于不同的场景，可以根据具体需求选择合适的工具：
- **生产环境**：Kaniko 或 BuildKit
- **开发环境**：Docker-in-Docker 或 Skaffold
- **Java 应用**：Jib
- **企业级 CI/CD**：Tekton

---

**文档版本**: v1.0  
**创建日期**: 2025-11-06  
**作者**: 技术团队

