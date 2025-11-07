# K8s 集群中构建镜像测试项目

本项目包含在 Kubernetes 集群中构建容器镜像的各种实现方式和测试代码。

## 📁 目录结构

```
test_image/
├── docs/                          # 文档目录
│   ├── README.md                  # 文档索引
│   ├── K8s构建镜像方式对比调研.md
│   ├── 可行性研究-*.md
│   └── 频繁构建场景分析.md
│
├── kaniko_privileged_demo/        # Kaniko 特权模式示例
│   ├── main.go
│   └── README.md
│
├── kaniko_rootless_demo/          # Kaniko 非特权模式示例
│   ├── main.go
│   └── README.md
│
├── buildah_privileged_demo/       # Buildah 特权模式示例
│   ├── main.go
│   └── README.md
│
├── buildah_rootless_demo/         # Buildah Rootless 模式示例
│   ├── main.go
│   └── README.md
│
├── crane_demo/                    # Crane 叠加文件示例
│   ├── main.go
│   ├── optimized_main.go
│   └── README.md
│
├── deployments/                   # K8s 部署配置文件
│   ├── README.md
│   └── *.yaml
│
├── demo_server/                   # 测试用的 Go 服务
│   └── main.go
│
├── image/                         # 镜像构建工具
│   └── README.md
│
└── README.md                      # 本文件
```

## 🚀 快速开始

### 1. 查看文档

```bash
# 查看文档索引
cat docs/README.md

# 查看综合对比
cat docs/K8s构建镜像方式对比调研.md
```

### 2. 运行示例

选择一个 demo 目录，查看对应的 README：

```bash
# Buildah 特权模式
cd buildah_privileged_demo
cat README.md

# Buildah Rootless 模式
cd buildah_rootless_demo
cat README.md

# Crane 叠加文件
cd crane_demo
cat README.md
```

### 3. 部署到 K8s

```bash
# 查看 K8s 部署说明
cat deployments/README.md

# 部署示例
kubectl apply -f deployments/buildah-demo-deployment.yaml
```

## 📚 构建方式对比

| 方式 | 权限要求 | 安全性 | 功能 | 适用场景 |
|------|---------|--------|------|---------|
| **Kaniko** | 特权/非特权 | 中/高 | 完整构建 | 通用构建 |
| **Buildah (Privileged)** | 特权 | 低 | 完整构建 | 开发/测试 |
| **Buildah (Rootless)** | 非特权 | 高 | 完整构建 | 生产环境 |
| **Crane** | 非特权 | 高 | 叠加文件 | 文件叠加 |

详细对比请查看：[docs/K8s构建镜像方式对比调研.md](./docs/K8s构建镜像方式对比调研.md)

## 🔧 工具说明

### kaniko_privileged_demo
使用 Kaniko 在特权模式下构建镜像，基础构建示例。

### kaniko_rootless_demo
尝试使用 Kaniko 在非特权模式下构建镜像（待验证）。

### buildah_privileged_demo
使用 Buildah 在特权模式下构建镜像，配置简单但安全性较低。

### buildah_rootless_demo
使用 Buildah 在 Rootless 模式下构建镜像，安全性高，适合生产环境。

### crane_demo
使用 Crane 在现有镜像上叠加文件，无需特权模式，轻量级。

### kaniko_rootless_demo
尝试使用 Kaniko 在非特权模式下构建镜像（待验证）。

## 📝 文档说明

所有调研和可行性研究文档都放在 `docs/` 目录下：

- **综合对比**：`K8s构建镜像方式对比调研.md`
- **可行性研究**：`可行性研究-*.md`
- **场景分析**：`频繁构建场景分析.md`

查看 [docs/README.md](./docs/README.md) 获取完整的文档索引。

## 🔗 相关链接

- [Buildah 官方文档](https://github.com/containers/buildah)
- [Kaniko 官方文档](https://github.com/GoogleContainerTools/kaniko)
- [Crane (go-containerregistry)](https://github.com/google/go-containerregistry)

## 📄 许可证

本项目为内部测试项目。

