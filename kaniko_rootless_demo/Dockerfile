# 用于在容器内运行 Kaniko 构建程序的 Dockerfile
FROM registry.cn-hangzhou.aliyuncs.com/kube-image-repo/kaniko:v1.9.1-debug

# 复制构建程序
COPY main /kaniko-build/main

# 复制 server/main 文件到镜像中（避免运行时挂载大文件）
# 注意：需要先复制文件到构建上下文：cp ../demo_server/main server-main
COPY server-main /workspace/server/main

# 设置工作目录
WORKDIR /workspace

# 默认命令：运行构建程序
ENTRYPOINT ["/kaniko-build/main"]

