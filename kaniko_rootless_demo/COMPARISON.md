# Kaniko Rootless vs Privileged 模式对比

本文档详细对比 Kaniko 在 Rootless（非特权）模式和 Privileged（特权）模式下的差异。

## 核心区别

| 特性 | Rootless 模式 | Privileged 模式 |
|------|--------------|----------------|
| **安全配置** | `privileged: false`（默认）<br>`allowPrivilegeEscalation: false` | `privileged: true` |
| **安全性** | 🟢 **高** - 符合最小权限原则 | 🔴 **低** - 拥有主机级权限 |
| **权限范围** | 受限，只能访问明确授予的资源 | 几乎拥有主机 root 用户的所有权限 |
| **容器逃逸风险** | 🟢 低 - 影响范围有限 | 🔴 高 - 可能影响整个节点 |
| **适用环境** | ✅ 生产环境 | ⚠️ 开发/测试环境 |
| **合规性** | ✅ 符合安全最佳实践 | ❌ 违反最小权限原则 |

## 配置对比

### Rootless 模式配置

```yaml
apiVersion: batch/v1
kind: Job
spec:
  template:
    spec:
      containers:
        - name: executor
          image: registry.cn-hangzhou.aliyuncs.com/kube-image-repo/kaniko:v1.9.1-debug
          securityContext:
            # 不设置 privileged（默认为 false）
            allowPrivilegeEscalation: false  # 禁止权限提升
            # 可选：进一步限制 capabilities
            # capabilities:
            #   drop:
            #     - ALL
          args:
            - --dockerfile=/workspace/Dockerfile
            - --context=/workspace
            - --destination=registry.kube-system.svc.cluster.local:5000/new-kaniko-image:latest
            - --skip-tls-verify
            - --skip-tls-verify-pull
```

**关键点**：
- ✅ `privileged` 字段不设置或设置为 `false`（默认值）
- ✅ `allowPrivilegeEscalation: false` - 禁止权限提升
- ✅ 可以进一步限制 capabilities（删除不需要的权限）

### Privileged 模式配置

```yaml
apiVersion: batch/v1
kind: Job
spec:
  template:
    spec:
      containers:
        - name: executor
          image: registry.cn-hangzhou.aliyuncs.com/kube-image-repo/kaniko:v1.9.1-debug
          securityContext:
            privileged: true  # 启用特权模式
            # allowPrivilegeEscalation 在 privileged 模式下通常为 true
          args:
            - --dockerfile=/workspace/Dockerfile
            - --context=/workspace
            - --destination=registry.kube-system.svc.cluster.local:5000/new-kaniko-image:latest
            - --skip-tls-verify
            - --skip-tls-verify-pull
```

**关键点**：
- ⚠️ `privileged: true` - 启用特权模式
- ⚠️ 容器拥有主机级权限
- ⚠️ 可以访问主机的所有设备和内核功能

## 功能对比

### 1. 构建能力

| 功能 | Rootless 模式 | Privileged 模式 |
|------|--------------|----------------|
| **基础镜像拉取** | ✅ 支持 | ✅ 支持 |
| **Dockerfile 解析** | ✅ 支持 | ✅ 支持 |
| **文件复制（COPY）** | ✅ 支持 | ✅ 支持 |
| **层缓存** | ✅ 支持 | ✅ 支持 |
| **镜像推送** | ✅ 支持 | ✅ 支持 |
| **多阶段构建** | ✅ 支持 | ✅ 支持 |

**结论**：两种模式在构建功能上**完全相同**，Kaniko 在非特权模式下也能完成所有构建操作。

### 2. 文件系统操作

| 操作 | Rootless 模式 | Privileged 模式 |
|------|--------------|----------------|
| **创建目录** | ✅ 支持（在容器内） | ✅ 支持（在容器内） |
| **文件读写** | ✅ 支持（在容器内） | ✅ 支持（在容器内） |
| **挂载主机文件系统** | ❌ 受限 | ✅ 可以 |
| **访问主机设备** | ❌ 受限 | ✅ 可以 |
| **修改主机内核参数** | ❌ 不可以 | ✅ 可以 |

**结论**：Rootless 模式在容器内的文件操作完全正常，但无法访问主机资源。

### 3. 网络操作

| 操作 | Rootless 模式 | Privileged 模式 |
|------|--------------|----------------|
| **拉取镜像** | ✅ 支持 | ✅ 支持 |
| **推送镜像** | ✅ 支持 | ✅ 支持 |
| **访问集群内服务** | ✅ 支持 | ✅ 支持 |
| **修改主机网络配置** | ❌ 不可以 | ✅ 可以 |

**结论**：两种模式在网络操作上**完全相同**（在容器网络层面）。

## 安全性对比

### Rootless 模式安全特性

✅ **最小权限原则**
- 容器只能访问明确授予的资源
- 无法访问主机设备（如 `/dev`）
- 无法修改主机内核参数

✅ **权限隔离**
- `allowPrivilegeEscalation: false` 防止权限提升
- 可以进一步限制 capabilities

✅ **容器逃逸影响**
- 即使容器被攻破，影响范围也仅限于容器内
- 无法访问其他 Pod 或主机资源

### Privileged 模式安全风险

🔴 **主机级权限**
- 容器拥有主机 root 用户的大部分权限
- 可以访问主机的所有设备
- 可以修改主机内核参数

🔴 **容器逃逸风险**
- 如果容器被攻破，攻击者可能获得主机 root 权限
- 可能影响整个节点上的其他 Pod
- 可能访问集群中的敏感数据

🔴 **合规性问题**
- 违反最小权限原则
- 不符合大多数安全标准（如 CIS、PCI-DSS）
- 生产环境通常禁止使用

## 性能对比

| 指标 | Rootless 模式 | Privileged 模式 |
|------|--------------|----------------|
| **构建速度** | 🟢 相同 | 🟢 相同 |
| **资源占用** | 🟢 相同 | 🟢 相同 |
| **镜像大小** | 🟢 相同 | 🟢 相同 |
| **缓存效率** | 🟢 相同 | 🟢 相同 |

**结论**：两种模式在性能上**完全相同**，特权模式不会带来性能提升。

## 使用场景对比

### Rootless 模式适用场景

✅ **生产环境**
- 符合安全最佳实践
- 满足合规性要求
- 降低安全风险

✅ **CI/CD 流水线**
- 自动化构建和部署
- 多租户环境
- 需要审计和合规的场景

✅ **安全要求高的场景**
- 金融、医疗等敏感行业
- 需要通过安全审计的系统

### Privileged 模式适用场景

⚠️ **开发/测试环境**
- 快速验证功能
- 调试和排错
- 临时测试

⚠️ **特殊需求场景**
- 需要访问主机设备的场景
- 需要修改内核参数的场景
- 某些遗留系统

❌ **不推荐用于生产环境**

## 实际测试结果

### Rootless 模式测试结果

✅ **测试通过**
- 镜像构建成功
- 镜像推送成功
- 镜像验证成功（文件内容正确）

**测试日志关键信息**：
```
INFO[0006] Pushing image to registry.kube-system.svc.cluster.local:5000/new-kaniko-image:latest 
INFO[0006] Pushed registry.kube-system.svc.cluster.local:5000/new-kaniko-image@sha256:...
```

**验证结果**：
```
built-by: kaniko
message: hello from kaniko build
OK
```

### Privileged 模式测试结果

✅ **功能相同**
- 构建和推送功能与 Rootless 模式完全相同
- 但存在安全风险

## 迁移建议

### 从 Privileged 迁移到 Rootless

1. **更新安全配置**
   ```yaml
   # 移除或设置为 false
   securityContext:
     privileged: false  # 或直接删除该字段
     allowPrivilegeEscalation: false
   ```

2. **测试验证**
   - 运行相同的构建任务
   - 验证构建结果
   - 确认功能正常

3. **无需其他改动**
   - Kaniko 参数保持不变
   - Dockerfile 保持不变
   - 构建流程保持不变

## 总结

### 推荐方案

**✅ 优先使用 Rootless 模式**
- 功能完全相同
- 安全性更高
- 符合最佳实践
- 适合生产环境

**⚠️ 仅在必要时使用 Privileged 模式**
- 开发/测试环境
- 特殊需求场景
- 不推荐用于生产

### 关键结论

1. **功能相同**：Rootless 和 Privileged 模式在构建功能上完全相同
2. **安全性不同**：Rootless 模式安全性远高于 Privileged 模式
3. **性能相同**：两种模式性能完全相同
4. **推荐 Rootless**：生产环境应优先使用 Rootless 模式

## 参考

- [Kaniko 官方文档](https://github.com/GoogleContainerTools/kaniko)
- [Kubernetes Security Context](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/)
- [本项目的 Rootless 实现](./kaniko-build.md)

