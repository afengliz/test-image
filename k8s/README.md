# Kubernetes 构建镜像部署说明

## 已完成的工作

### 1. ✅ 找到镜像仓库地址
- **集群内部地址**：`registry.kube-system.svc.cluster.local:5000`
- **Service 信息**：
  - Namespace: `kube-system`
  - Service Name: `registry`
  - Port: `5000`
  - ClusterIP: `10.43.185.166`

### 2. ✅ 创建了 Deployment 配置
文件位置：`ones-platform-api/test_image/k8s/build-image-deployment.yaml`

**功能**：
- 使用 `registry.cn-hangzhou.aliyuncs.com/kube-image-repo/kaniko:v1.9.1-debug` 作为基础镜像
- 构建包含 server 文件夹的 Docker 镜像
- 推送镜像到集群内部仓库 `registry.kube-system.svc.cluster.local:5000`
- 设置工作目录为 `/usr/local/app`

### 3. ✅ 构建流程
1. 拉取基础镜像 `localhost:5000/ones/plugin-host-node:v6.33.1`
2. 创建 Dockerfile（动态生成）
3. 创建 server 文件夹和示例 Go 代码
4. 使用 buildah 构建镜像
5. 推送镜像到仓库
6. 验证镜像



## 部署命令

```bash
# 部署
kubectl apply -f ones-platform-api/test_image/k8s/build-image-deployment.yaml

# 查看状态
kubectl get pods -n ones -l app=build-image

# 查看日志
kubectl logs -n ones -l app=build-image -f

# 删除部署
kubectl delete deployment build-image-deployment -n ones
```

## 查看构建结果

构建成功后，镜像会推送到：
```
registry.kube-system.svc.cluster.local:5000/test-app:<timestamp>
```
`

