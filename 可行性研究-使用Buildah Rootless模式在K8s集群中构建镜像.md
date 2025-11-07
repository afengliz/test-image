# 在 K8s 集群中使用 Buildah Rootless 模式构建镜像的可行性研究

## 1. 研究背景

### 1.1 研究目的
验证在 Kubernetes 集群内部使用 **Buildah Rootless 模式**（非 root 权限）构建 Docker 镜像的可行性，探索在非特权容器中构建镜像的可能性，为生产环境提供更安全的镜像构建方案。

### 1.2 研究范围
- 在非特权 K8s Pod 中使用 Buildah Rootless 模式构建镜像
- 验证 `buildah unshare` 在容器环境中的可用性
- 测试不同存储驱动（vfs, overlay）在 Rootless 模式下的表现
- 对比 Rootless 模式与 Privileged 模式的差异
- 分析在 Kubernetes 环境中实现真正 Rootless 的挑战

### 1.3 技术选型
- **构建工具**: Buildah v1.41.4
- **运行模式**: Rootless（非 root 用户 + 用户命名空间）
- **基础镜像**: `localhost:5000/ones/ones/ones-toolkit:v6.37.0-ones.1`
- **目标仓库**: `registry.kube-system.svc.cluster.local:5000`
- **编程语言**: Go
- **存储驱动**: vfs（Rootless 模式推荐）
- **网络后端**: netavark / slirp4netns

## 2. 技术方案

### 2.1 Rootless 模式原理

**Rootless 模式**是指以非 root 用户身份运行容器工具，通过 Linux 用户命名空间（User Namespace）来模拟 root 权限，从而在不需要实际 root 权限的情况下执行需要特权的操作。

#### 2.1.1 核心机制

```
┌─────────────────────────────────────────────────────────┐
│  非 root 用户 (UID: 1000)                                │
│                                                           │
│  ┌────────────────────────────────────────────────────┐  │
│  │ buildah unshare                                    │  │
│  │  - 创建用户命名空间                                 │  │
│  │  - 映射 UID 1000 -> 命名空间内的 root (UID 0)      │  │
│  │  - 在命名空间内执行 buildah bud                    │  │
│  └────────────────────────────────────────────────────┘  │
│           │                                               │
│           │ 用户命名空间隔离                               │
│           ▼                                               │
│  ┌────────────────────────────────────────────────────┐  │
│  │ 用户命名空间 (User Namespace)                       │  │
│  │  - 内部 UID 0 (映射到外部 UID 1000)                │  │
│  │  - 可以执行 mount、chown 等操作                     │  │
│  │  - 不影响主机系统                                    │  │
│  └────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
```

#### 2.1.2 关键组件

1. **`buildah unshare`**
   - 创建新的用户命名空间
   - 映射用户 ID 和组 ID（通过 `/etc/subuid` 和 `/etc/subgid`）
   - 在命名空间中运行命令

2. **subuid/subgid 配置**
   ```
   /etc/subuid: 1000:100000:65536
   /etc/subgid: 1000:100000:65536
   ```
   - 定义用户命名空间的 UID/GID 映射范围
   - 格式：`用户名:起始ID:数量`

3. **存储驱动选择**
   - **vfs**: 不需要 remount 权限，适合 Rootless
   - **overlay**: 需要 fuse-overlayfs，性能更好但配置复杂

### 2.2 架构设计

```
┌─────────────────────────────────────────────────────────┐
│  K8s Cluster                                            │
│                                                          │
│  ┌──────────────────────────────────────────────────┐   │
│  │ buildah-rootless-demo-pod (非特权模式)           │   │
│  │  - 运行 Go 构建程序                              │   │
│  │  - 检测用户类型（root/非root）                   │   │
│  │  - root: 直接使用 buildah bud                    │   │
│  │  - 非root: 使用 buildah unshare                  │   │
│  │  - 构建新镜像                                     │   │
│  │  - 推送镜像到仓库                                 │   │
│  └──────────────────────────────────────────────────┘   │
│           │                                              │
│           │ 推送镜像                                     │
│           ▼                                              │
│  ┌──────────────────────────────────────────────────┐   │
│  │ registry.kube-system.svc.cluster.local:5000     │   │
│  │  - 存储构建的新镜像                               │   │
│  └──────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
```

### 2.3 核心组件

#### 2.3.1 Buildah Rootless 配置

**存储配置** (`~/.config/containers/storage.conf`):
```ini
[storage]
driver = "vfs"
runroot = "/run/user/1000/containers/storage"
graphroot = "$HOME/.local/share/containers/storage"
```

**容器配置** (`~/.config/containers/containers.conf`):
```ini
[containers]
netns = "none"

[engine]
helper_binaries_dir = ["/usr/libexec/podman", "/usr/local/libexec/podman"]
```

#### 2.3.2 构建程序

- **文件位置**: `buildah_rootless_demo/main.go`
- **核心功能**:
  1. 自动检测当前用户类型（root 或非 root）
  2. 根据用户类型选择构建方式
  3. 自动配置 Rootless 存储和容器设置
  4. 使用 `buildah bud` 或 `buildah unshare buildah bud` 构建镜像
  5. 推送镜像到仓库

## 3. 实施步骤

### 3.1 第一步：创建构建 Pod

创建基于 Buildah 的 Deployment，**不设置 privileged**（Rootless 模式）：

```1:42:k8s/buildah-rootless-demo-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: buildah-rootless-demo-deployment
  namespace: ones
  labels:
    app: buildah-rootless-demo
spec:
  replicas: 1
  selector:
    matchLabels:
      app: buildah-rootless-demo
  template:
    metadata:
      labels:
        app: buildah-rootless-demo
    spec:
      containers:
      - name: buildah-rootless-demo
        image: localhost:5000/ones/ones/ones-toolkit:v6.37.0-ones.1
        command: ["/bin/sh"]
        args:
          - -c
          - |
            sleep 3600
        env:
        - name: REGISTRY
          value: "registry.kube-system.svc.cluster.local:5000"
        # Rootless 模式：代码会自动检测用户类型
        # 如果是 root 用户，直接使用 buildah bud（不使用 unshare）
        # 如果是非 root 用户，使用 buildah unshare
        # securityContext:
        #   runAsNonRoot: true  # 可选：以非 root 用户运行（需要配置 subuid/subgid）
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "2Gi"
            cpu: "1000m"
```

**执行命令**:
```bash
kubectl apply -f k8s/buildah-rootless-demo-deployment.yaml
```

**执行结果**: ✅ Pod 成功创建并运行

📸 **截图位置**: 执行 `kubectl get pods -n ones -l app=buildah-rootless-demo` 查看 Pod 状态

**验证输出示例**:
```
NAME                                                READY   STATUS    RESTARTS   AGE
buildah-rootless-demo-deployment-855f658df-8ftxf   1/1     Running   0          14s
```

### 3.2 第二步：安装 Buildah

在 Pod 中安装 Buildah 运行时：

```bash
kubectl exec -n ones <pod-name> -- apk add --no-cache buildah netavark
```

**执行结果**: ✅ Buildah v1.41.4 安装成功

**验证输出示例**:
```
buildah version 1.41.4 (image-spec 1.1.1, runtime-spec 1.2.1)
```

### 3.3 第三步：开发构建程序

#### 3.3.1 程序功能

1. 自动检测当前用户类型（root 或非 root）
2. 根据用户类型选择构建方式：
   - **root 用户**: 直接使用 `buildah bud`（不需要 unshare）
   - **非 root 用户**: 使用 `buildah unshare buildah bud`
3. 自动配置 Rootless 存储（vfs 驱动）
4. 自动配置容器网络设置
5. 动态生成 Dockerfile
6. 构建并推送镜像

#### 3.3.2 构建程序核心代码

完整的构建程序代码：

```12:198:buildah_rootless_demo/main.go
func main() {
	// 配置参数（参照 build_image/main.go）
	baseImage := "registry.kube-system.svc.cluster.local:5000/ones/plugin-host-node:v6.33.1"
	mainFilePath := "/workspace/server/main"
	imageName := "registry.kube-system.svc.cluster.local:5000/new-buildah-rootless-image:latest"

	fmt.Println("=== Buildah Rootless 模式构建镜像 ===")
	fmt.Println("Rootless 模式：无需 root 权限，使用用户命名空间")

	// 检查当前用户
	currentUser, err := user.Current()
	if err != nil {
		log.Printf("警告: 无法获取当前用户信息: %v", err)
	} else {
		fmt.Printf("当前用户: %s (UID: %s, GID: %s)\n", currentUser.Username, currentUser.Uid, currentUser.Gid)
	}

	// 构建镜像
	if err := buildImageRootless(baseImage, mainFilePath, imageName); err != nil {
		log.Fatalf("构建镜像失败: %v", err)
	}

	fmt.Printf("✓ 镜像构建并推送成功: %s\n", imageName)
}

// Rootless 模式构建镜像
// 使用 buildah unshare 来创建用户命名空间，无需 root 权限
func buildImageRootless(baseImage, mainFilePath, imageName string) error {
	fmt.Printf("使用基础镜像: %s\n", baseImage)

	// 检查 main 文件是否存在
	if _, err := os.Stat(mainFilePath); err != nil {
		return fmt.Errorf("main 文件不存在: %s, %w", mainFilePath, err)
	}

	// 获取用户主目录（用于 Rootless 配置）
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		// 如果 HOME 未设置，尝试使用 /tmp 作为工作目录
		homeDir = "/tmp"
		fmt.Printf("警告: HOME 环境变量未设置，使用 /tmp 作为工作目录\n")
	}

	// Rootless 模式的配置目录
	configDir := filepath.Join(homeDir, ".config", "containers")
	storageConfPath := filepath.Join(configDir, "storage.conf")
	containersConfPath := filepath.Join(configDir, "containers.conf")

	// 确保配置目录存在
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	// 配置 Rootless 存储（使用 vfs 驱动，不需要 remount 权限）
	if err := setupRootlessStorage(storageConfPath); err != nil {
		return fmt.Errorf("配置 Rootless 存储失败: %w", err)
	}
	fmt.Println("✓ Rootless 存储配置完成")

	// 配置 Rootless 容器设置
	if err := setupRootlessContainers(containersConfPath); err != nil {
		return fmt.Errorf("配置 Rootless 容器设置失败: %w", err)
	}
	fmt.Println("✓ Rootless 容器配置完成")

	// 设置 buildah 环境变量（Rootless 模式）
	os.Setenv("CONTAINERS_STORAGE_CONF", storageConfPath)
	os.Setenv("CONTAINERS_CONF", containersConfPath)
	// Rootless 模式使用用户目录存储
	os.Setenv("XDG_RUNTIME_DIR", filepath.Join(homeDir, ".local", "share", "containers"))

	// 创建临时工作目录（在用户可写的位置）
	workDir := filepath.Join(homeDir, ".local", "buildah-work")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return fmt.Errorf("创建工作目录失败: %w", err)
	}
	defer os.RemoveAll(workDir)

	// 1. 创建临时 Dockerfile
	dockerfileContent := fmt.Sprintf(`FROM %s
WORKDIR /usr/local/app
COPY main /usr/local/app/main
ENTRYPOINT ["/usr/local/app/main"]
`, baseImage)

	dockerfilePath := filepath.Join(workDir, "Dockerfile")
	if err := os.WriteFile(dockerfilePath, []byte(dockerfileContent), 0644); err != nil {
		return fmt.Errorf("创建 Dockerfile 失败: %w", err)
	}
	fmt.Println("✓ Dockerfile 创建成功")

	// 2. 创建构建上下文目录
	contextDir := filepath.Join(workDir, "build-context")
	if err := os.MkdirAll(contextDir, 0755); err != nil {
		return fmt.Errorf("创建构建上下文目录失败: %w", err)
	}

	// 3. 复制 main 文件到构建上下文
	contextMainPath := filepath.Join(contextDir, "main")
	if err := copyFile(mainFilePath, contextMainPath); err != nil {
		return fmt.Errorf("复制 main 文件失败: %w", err)
	}
	fmt.Printf("✓ 复制文件: %s -> %s\n", mainFilePath, contextMainPath)

	// 4. 复制 Dockerfile 到构建上下文
	contextDockerfilePath := filepath.Join(contextDir, "Dockerfile")
	if err := copyFile(dockerfilePath, contextDockerfilePath); err != nil {
		return fmt.Errorf("复制 Dockerfile 失败: %w", err)
	}

	// 5. 使用 buildah 构建镜像
	// 检测当前用户：如果是 root，直接使用 buildah bud；否则使用 buildah unshare
	currentUser, err := user.Current()
	isRoot := err == nil && currentUser.Uid == "0"
	
	if isRoot {
		// root 用户：直接使用 buildah bud（不需要 unshare）
		// 使用 --isolation chroot 来避免需要 remount 权限
		fmt.Println("正在使用 buildah 构建镜像（root 用户模式）...")
		buildCmd := exec.Command("buildah", "bud",
			"--tls-verify=false",
			"--storage-driver", "vfs", // 使用 vfs 驱动
			"--isolation", "chroot", // 使用 chroot 隔离，避免 remount
			"-f", contextDockerfilePath,
			"-t", imageName,
			contextDir,
		)
		buildCmd.Stdout = os.Stdout
		buildCmd.Stderr = os.Stderr
		buildCmd.Env = os.Environ()
		
		if err := buildCmd.Run(); err != nil {
			return fmt.Errorf("构建镜像失败: %w", err)
		}
	} else {
		// 非 root 用户：使用 buildah unshare 创建用户命名空间
		fmt.Println("正在使用 Rootless 模式构建镜像...")
		fmt.Println("提示: 使用 buildah unshare 创建用户命名空间")
		
		buildCmd := exec.Command("buildah", "unshare", "buildah", "bud",
			"--tls-verify=false",
			"--storage-driver", "vfs", // Rootless 模式使用 vfs 驱动，不需要 remount
			"-f", contextDockerfilePath,
			"-t", imageName,
			contextDir,
		)
		buildCmd.Stdout = os.Stdout
		buildCmd.Stderr = os.Stderr
		buildCmd.Env = os.Environ()
		
		if err := buildCmd.Run(); err != nil {
			return fmt.Errorf("Rootless 模式构建镜像失败: %w", err)
		}
	}
	fmt.Printf("✓ 镜像构建成功: %s\n", imageName)

	// 6. 推送镜像到 registry
	if isRoot {
		fmt.Println("正在推送镜像到 registry...")
		pushCmd := exec.Command("buildah", "push",
			"--tls-verify=false",
			imageName,
			"docker://"+imageName,
		)
		pushCmd.Stdout = os.Stdout
		pushCmd.Stderr = os.Stderr
		pushCmd.Env = os.Environ()
		
		if err := pushCmd.Run(); err != nil {
			return fmt.Errorf("推送镜像失败: %w", err)
		}
	} else {
		fmt.Println("正在使用 Rootless 模式推送镜像到 registry...")
		pushCmd := exec.Command("buildah", "unshare", "buildah", "push",
			"--tls-verify=false",
			imageName,
			"docker://"+imageName,
		)
		pushCmd.Stdout = os.Stdout
		pushCmd.Stderr = os.Stderr
		pushCmd.Env = os.Environ()
		
		if err := pushCmd.Run(); err != nil {
			return fmt.Errorf("Rootless 模式推送镜像失败: %w", err)
		}
	}

	fmt.Printf("✓ 镜像推送成功: %s\n", imageName)
	return nil
}
```

#### 3.3.3 编译程序

```bash
cd buildah_rootless_demo
GOOS=linux GOARCH=amd64 go build -o buildah-rootless-demo main.go
```

**执行结果**: ✅ 程序编译成功

### 3.4 第四步：部署并运行程序

#### 3.4.1 复制文件到 Pod

```bash
# 创建目录
kubectl exec -n ones <pod-name> -- mkdir -p /workspace/server

# 复制程序
kubectl cp buildah_rootless_demo/buildah-rootless-demo ones/<pod-name>:/workspace/buildah-rootless-demo

# 复制 server/main
kubectl cp server/main ones/<pod-name>:/workspace/server/main
```

#### 3.4.2 运行程序

```bash
kubectl exec -n ones <pod-name> -- chmod +x /workspace/buildah-rootless-demo
kubectl exec -n ones <pod-name> -- /workspace/buildah-rootless-demo
```

## 4. 测试结果

### 4.1 测试环境

- **Kubernetes 版本**: v1.22+
- **Buildah 版本**: v1.41.4
- **Pod 权限模式**: 非特权（未设置 `privileged: true`）
- **运行用户**: root（容器默认）

### 4.2 测试过程

#### 4.2.1 第一次测试：root 用户 + vfs 驱动

**测试配置**:
- 用户: root (UID: 0)
- 存储驱动: vfs
- 隔离模式: chroot
- 网络后端: netavark

**执行结果**: ❌ **失败**

**错误信息**:
```
Error: creating build container: unable to copy from source docker://registry.kube-system.svc.cluster.local:5000/ones/plugin-host-node:v6.33.1: writing blob: adding layer with blob "sha256:91f01557fe0da558070d4f24631c94e91a80877a24621b52b8b13009b62d8d96": ApplyLayer stdout:  stderr: remount /, flags: 0x44000: permission denied exit status 1
```

**问题分析**:
- Buildah 在应用镜像层时需要执行 `remount` 操作
- 即使使用 `vfs` 驱动和 `--isolation chroot`，仍然需要 remount 权限
- 非特权容器无法执行 remount 操作

#### 4.2.2 第二次测试：非 root 用户 + buildah unshare

**测试配置**:
- 用户: 非 root (UID: 1000)
- 使用: `buildah unshare`
- 存储驱动: vfs

**执行结果**: ❌ **失败**

**错误信息**:
```
Error: writing "0 0 1\n1 100000 65536\n" to /proc/359/gid_map: write /proc/359/gid_map: operation not permitted
```

**问题分析**:
- `buildah unshare` 需要创建用户命名空间
- 创建用户命名空间需要写入 `/proc/*/gid_map`
- 这需要 `CAP_SYS_ADMIN` 权限或 `privileged: true`

#### 4.2.3 第三次测试：配置 subuid/subgid

**测试配置**:
- 配置 `/etc/subuid` 和 `/etc/subgid`
- 用户: root（尝试配置 subuid/subgid）

**执行结果**: ❌ **失败**

**问题分析**:
- root 用户通常不需要 subuid/subgid 配置
- 即使配置了，仍然遇到 remount 权限问题

### 4.3 测试结论

❌ **在非特权 Kubernetes Pod 中，Buildah Rootless 模式无法正常工作**

## 5. 问题分析

### 5.1 核心问题

#### 5.1.1 remount 权限问题

**问题描述**:
即使使用 `vfs` 存储驱动和 `--isolation chroot`，Buildah 在应用镜像层时仍然需要执行 `remount` 操作。

**技术原因**:
- Buildah 在应用镜像层时需要修改文件系统的挂载选项
- `remount` 操作需要 `CAP_SYS_ADMIN` 权限
- 非特权容器默认没有 `CAP_SYS_ADMIN` 权限

**错误示例**:
```
remount /, flags: 0x44000: permission denied
```

#### 5.1.2 用户命名空间问题

**问题描述**:
`buildah unshare` 需要创建用户命名空间，这需要写入 `/proc/*/gid_map`。

**技术原因**:
- 创建用户命名空间需要 `CAP_SYS_ADMIN` 权限
- 或者需要 `privileged: true`
- 非特权容器无法创建用户命名空间

**错误示例**:
```
write /proc/359/gid_map: operation not permitted
```

#### 5.1.3 网络配置问题

**问题描述**:
Buildah 需要网络后端（netavark 或 slirp4netns）来管理容器网络。

**解决方案**:
- 安装 `netavark` 或 `slirp4netns`
- 配置 `containers.conf` 中的网络后端
- 设置 `helper_binaries_dir`

### 5.2 权限要求对比

| 操作 | 所需权限 | 非特权容器 | Privileged 容器 |
|------|---------|-----------|----------------|
| **remount** | `CAP_SYS_ADMIN` | ❌ 不支持 | ✅ 支持 |
| **创建用户命名空间** | `CAP_SYS_ADMIN` | ❌ 不支持 | ✅ 支持 |
| **写入 /proc/*/gid_map** | `CAP_SYS_ADMIN` | ❌ 不支持 | ✅ 支持 |
| **mount 操作** | `CAP_SYS_ADMIN` | ❌ 不支持 | ✅ 支持 |

## 6. 可行性结论

### 6.1 结论

❌ **在非特权 Kubernetes Pod 中，Buildah Rootless 模式目前不可行**

### 6.2 原因分析

1. **remount 权限限制**
   - Buildah 在应用镜像层时需要 remount 权限
   - 非特权容器无法获得 remount 权限
   - 即使使用 `vfs` 驱动和 `chroot` 隔离，问题依然存在

2. **用户命名空间限制**
   - `buildah unshare` 需要创建用户命名空间
   - 创建用户命名空间需要 `CAP_SYS_ADMIN` 权限
   - 非特权容器无法创建用户命名空间

3. **Kubernetes 安全模型**
   - Kubernetes 默认限制容器的权限
   - 非特权容器无法执行需要特权的操作
   - 真正的 Rootless 需要主机级别的配置支持

### 6.3 替代方案

#### 方案 1：使用 Privileged 模式（推荐用于开发/测试）

**配置**:
```yaml
securityContext:
  privileged: true
```

**优势**:
- ✅ 可以正常工作
- ✅ 配置简单
- ✅ 性能好

**劣势**:
- ❌ 安全性低
- ❌ 失去了 Rootless 的意义

#### 方案 2：使用 Kaniko（推荐用于生产）

**优势**:
- ✅ 相对成熟
- ✅ 支持非特权模式（需要特殊配置）
- ✅ 安全性较高

**劣势**:
- ⚠️ 默认也需要 privileged
- ⚠️ 功能相对简单

#### 方案 3：主机级别配置（真正的 Rootless）

**要求**:
- 在 Kubernetes 节点上配置用户命名空间支持
- 配置 `/etc/subuid` 和 `/etc/subgid`
- 可能需要修改 Kubernetes 配置

**优势**:
- ✅ 真正的 Rootless
- ✅ 安全性高

**劣势**:
- ❌ 配置复杂
- ❌ 需要节点级别权限
- ❌ 可能影响其他容器

## 7. 建议

### 7.1 开发/测试环境

**推荐方案**: 使用 `privileged: true` 模式

```yaml
securityContext:
  privileged: true
```

**理由**:
- 配置简单
- 可以正常工作
- 开发/测试环境对安全性要求相对较低

### 7.2 生产环境

**推荐方案**: 使用 Kaniko 或 Buildah + Privileged

**理由**:
- Kaniko 相对成熟，有更多生产环境使用案例
- 如果必须使用 Buildah，使用 Privileged 模式
- 真正的 Rootless 在 Kubernetes 中实现复杂，需要权衡成本和收益

### 7.3 未来改进方向

1. **等待 Buildah 改进**
   - 支持真正的无 remount 模式
   - 改进存储驱动，避免需要 remount

2. **Kubernetes 支持**
   - 更好的用户命名空间支持
   - 更细粒度的权限控制

3. **替代工具**
   - 探索其他支持真正 Rootless 的构建工具
   - 考虑使用 BuildKit（支持 rootless）

## 8. 附录

### 8.1 相关文件

- **构建程序**: `buildah_rootless_demo/main.go`
- **Deployment 配置**: `k8s/buildah-rootless-demo-deployment.yaml`
- **文档**: `buildah_rootless_demo/README.md`

### 8.2 参考链接

- [Buildah Rootless 文档](https://github.com/containers/buildah/blob/main/docs/tutorials/01-intro.md#rootless-mode)
- [Podman/Buildah Rootless 指南](https://github.com/containers/podman/blob/main/docs/tutorials/rootless_tutorial.md)
- [用户命名空间文档](https://man7.org/linux/man-pages/man7/user_namespaces.7.html)
- [Kubernetes Security Context](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/)

### 8.3 错误日志示例

#### remount 权限错误
```
Error: creating build container: unable to copy from source docker://registry.kube-system.svc.cluster.local:5000/ones/plugin-host-node:v6.33.1: writing blob: adding layer with blob "sha256:91f01557fe0da558070d4f24631c94e91a80877a24621b52b8b13009b62d8d96": ApplyLayer stdout:  stderr: remount /, flags: 0x44000: permission denied exit status 1
```

#### 用户命名空间错误
```
Error: writing "0 0 1\n1 100000 65536\n" to /proc/359/gid_map: write /proc/359/gid_map: operation not permitted
```

#### 网络配置错误
```
Error: creating build container: could not find "netavark" in one of [/usr/libexec/podman /usr/local/libexec/podman /usr/lib/podman /usr/local/lib/podman]
```

### 8.4 测试命令记录

```bash
# 1. 创建 Deployment
kubectl apply -f k8s/buildah-rootless-demo-deployment.yaml

# 2. 安装 Buildah
kubectl exec -n ones <pod-name> -- apk add --no-cache buildah netavark

# 3. 复制程序
kubectl cp buildah_rootless_demo/buildah-rootless-demo ones/<pod-name>:/workspace/buildah-rootless-demo
kubectl cp server/main ones/<pod-name>:/workspace/server/main

# 4. 运行程序
kubectl exec -n ones <pod-name> -- /workspace/buildah-rootless-demo
```

---

**文档版本**: v1.0  
**最后更新**: 2025-11-07  
**作者**: AI Assistant  
**状态**: ❌ 不可行（在非特权 Pod 中）

