# 文档索引

本目录包含在 K8s 集群中构建镜像的相关调研和可行性研究文档。

## 📚 文档列表

### 综合对比

- **[K8s构建镜像方式对比调研.md](./K8s构建镜像方式对比调研.md)**
  - 全面对比 Kaniko、Buildah、Crane 等构建方式
  - 包含权限要求、安全性、性能等详细分析
  - 推荐阅读：了解各种构建方式的优缺点

### 可行性研究

- **[可行性研究-在K8s集群中构建和推送镜像.md](./可行性研究-使用Buildah在K8s集群中构建和推送镜像.md)**
  - 使用 Kaniko 在 K8s 集群中构建镜像的可行性研究
  - 包含技术方案、测试结果、问题分析

- **[可行性研究-使用Buildah在K8s集群中构建和推送镜像.md](./可行性研究-使用Buildah在K8s集群中构建和推送镜像.md)**
  - Buildah 特权模式在 K8s 中的可行性研究
  - 包含部署方案、测试步骤、性能分析

- **[可行性研究-使用Buildah Rootless模式在K8s集群中构建镜像.md](./可行性研究-使用Buildah Rootless模式在K8s集群中构建镜像.md)**
  - Buildah Rootless 模式在 K8s 中的可行性研究
  - 重点关注安全性和非特权模式实现

### 场景分析

- **[频繁构建场景分析.md](./频繁构建场景分析.md)**
  - 针对频繁构建场景的性能优化分析
  - Crane 在频繁构建场景下的优势分析

### 测试说明

- **[test.md](./test.md)**
  - 初始测试任务说明
  - Kaniko 构建镜像的测试步骤

## 📖 阅读建议

1. **新手入门**：先阅读 `K8s构建镜像方式对比调研.md`，了解整体情况
2. **选择方案**：根据需求查看对应的可行性研究文档
3. **性能优化**：查看 `频繁构建场景分析.md` 了解优化策略

## 🔗 相关资源

- 各 Demo 目录的 README：
  - `../kaniko_privileged_demo/README.md` - Kaniko 特权模式示例
  - `../kaniko_rootless_demo/README.md` - Kaniko 非特权模式示例
  - `../buildah_privileged_demo/README.md` - Buildah 特权模式示例
  - `../buildah_rootless_demo/README.md` - Buildah Rootless 模式示例
  - `../crane_demo/README.md` - Crane 叠加文件示例
  - `../deployments/README.md` - K8s 部署配置说明

