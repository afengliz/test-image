# 在 K8s 中无特权构建并推送镜像（Kaniko）

本文档说明如何在 Kubernetes 集群中，在不使用 privileged 模式、也不依赖 Docker 守护进程的前提下，基于旧镜像添加一个文件，构建并推送一个新镜像。方案采用 Kaniko 一次性 Job，可复用/参数化。

## 目标

- 基于已有镜像添加一个文件，构建并推送新镜像
- 全程不使用 privileged，不依赖 Docker 守护进程
- 结果镜像可被集群工作负载正常拉取与使用

## 环境与前提

- 已配置 `kubectl` 访问集群
- 集群内置私有镜像仓服务：`registry`（位于 `kube-system` 命名空间）
  - 推送端点：`registry.kube-system.svc.cluster.local:5000`
  - 工作负载拉取建议端点：`localhost:5000`（同一仓库内容，便于拉取）
- 基础镜像存在：`registry.kube-system.svc.cluster.local:5000/ones/plugin-host-node:v6.33.1`

> 说明：部分环境下，工作负载直接拉取 `registry.kube-system.svc.cluster.local:5000/...` 可能遇到证书/信任问题；建议在 Pod 中使用 `localhost:5000/...` 拉取，同步指向内置 registry 内容。

## 方案概述

- 构建器：Kaniko（镜像：`registry.cn-hangzhou.aliyuncs.com/kube-image-repo/kaniko:v1.9.1-debug`）
- 构建上下文通过 `ConfigMap` 提供（包含 `Dockerfile` 与要添加的文件），由 `initContainer` 拷贝至 `emptyDir:/workspace`（可写）
- Kaniko 从 `/workspace` 读取 `Dockerfile` 和上下文，构建并推送到内置 registry
- 安全：未使用 `privileged`，容器设置 `allowPrivilegeEscalation: false`，不需要 Docker 守护进程

## 建议命名空间

```bash
kubectl get ns imgbuild >/dev/null 2>&1 || kubectl create ns imgbuild
```

## 构建上下文 ConfigMap（Dockerfile + 待添加文件）

将以下内容保存为 `kaniko-context.yaml`：

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: kaniko-context
  namespace: imgbuild
  labels:
    app: kaniko-addfile
data:
  Dockerfile: |
    FROM registry.kube-system.svc.cluster.local:5000/ones/plugin-host-node:v6.33.1
    USER root
    WORKDIR /opt/app
    COPY hello.txt /opt/app/hello.txt
  hello.txt: |
    built-by: kaniko
    message: hello from kaniko build
```

应用：

```bash
kubectl apply -f kaniko-context.yaml
```

## Kaniko Job（无特权 + init 容器准备上下文）

将以下内容保存为 `kaniko-job.yaml`：

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: kaniko-addfile
  namespace: imgbuild
spec:
  backoffLimit: 0
  ttlSecondsAfterFinished: 600
  template:
    metadata:
      labels:
        app: kaniko-addfile
    spec:
      restartPolicy: Never
      initContainers:
        - name: prepare-context
          image: localhost:5000/ones/ones/ones-toolkit:v6.37.0-ones.1
          command: ["/bin/sh","-c"]
          args:
            - |
              set -eux
              cp -v /cm/Dockerfile /workspace/Dockerfile
              cp -v /cm/hello.txt /workspace/hello.txt
          volumeMounts:
            - name: context
              mountPath: /workspace
            - name: cm
              mountPath: /cm
      containers:
        - name: executor
          image: registry.cn-hangzhou.aliyuncs.com/kube-image-repo/kaniko:v1.9.1-debug
          args:
            - --dockerfile=/workspace/Dockerfile
            - --context=/workspace
            - --destination=registry.kube-system.svc.cluster.local:5000/new-kaniko-image:latest
            - --skip-tls-verify
            - --skip-tls-verify-pull
            - --verbosity=debug
          volumeMounts:
            - name: context
              mountPath: /workspace
          securityContext:
            allowPrivilegeEscalation: false
      volumes:
        - name: context
          emptyDir: {}
        - name: cm
          configMap:
            name: kaniko-context
```

应用并等待完成：

```bash
kubectl apply -f kaniko-job.yaml
kubectl -n imgbuild wait --for=condition=complete job/kaniko-addfile --timeout=5m
kubectl -n imgbuild logs job/kaniko-addfile
```

成功日志关键行（示例）：

- `Pushing image to registry.kube-system.svc.cluster.local:5000/new-kaniko-image:latest`
- `Pushed ...@sha256:<digest>`

## 验证镜像内容

使用 `localhost:5000` 拉取测试（集群内工作负载建议这样拉取）：

```bash
kubectl -n imgbuild run verify-image \
  --image=localhost:5000/new-kaniko-image:latest \
  --restart=Never --command -- sh -c 'cat /opt/app/hello.txt && echo OK'

# 查看日志
kubectl -n imgbuild logs verify-image

# 预期输出：
# built-by: kaniko
# message: hello from kaniko build
# OK

# 清理
kubectl -n imgbuild delete pod verify-image --ignore-not-found
```

## 可选：非 root 场景提示

部分场景要求 `runAsNonRoot`。Kaniko 默认写入 `/kaniko`，在非 root 场景可能遇到权限问题（例如 chown/只读）。处理方式：

- 在 `args` 增加：`--kaniko-dir=/tmp/kaniko`
- 在 Pod 级 `securityContext` 设置：`runAsNonRoot: true`、`runAsUser: 1000`
- 保持构建上下文使用 `emptyDir:/workspace`（可写），不要直接在 `ConfigMap` 上构建

示例仅需在 `containers.args` 里追加一行：

```yaml
      - --kaniko-dir=/tmp/kaniko
```

## 常见问题排查

- ImagePullBackOff（工作负载拉取失败）
  - 尝试在工作负载中将镜像地址改为 `localhost:5000/...` 拉取
  - 确认 Job 日志已成功 Push，tag 与拉取一致
- 权限/只读错误（chown/只读文件系统）
  - 通过 `initContainer + emptyDir` 提供可写上下文
  - 非 root 时使用 `--kaniko-dir=/tmp/kaniko`
- 证书/网络问题
  - 构建端已添加 `--skip-tls-verify`/`--skip-tls-verify-pull`
  - 如有正式证书，可移除上述参数并在节点/容器中配置 CA 与认证

## 清理

```bash
kubectl -n imgbuild delete job kaniko-addfile --ignore-not-found
kubectl -n imgbuild delete configmap kaniko-context --ignore-not-found
```

## 实际结果（已验证）

- 新镜像已推送：`registry.kube-system.svc.cluster.local:5000/new-kaniko-image:latest`
- 在镜像中添加文件：`/opt/app/hello.txt`
- 通过 Pod 实测读取该文件输出 OK（使用镜像 `localhost:5000/new-kaniko-image:latest` 拉取验证）

