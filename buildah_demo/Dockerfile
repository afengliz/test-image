# 用于在容器内运行 Buildah 构建程序的 Dockerfile
FROM quay.io/buildah/stable:latest

# 安装必要的工具
RUN microdnf install -y \
    fuse-overlayfs \
    && microdnf clean all

# 复制构建程序
COPY main /workspace/main

# 设置工作目录
WORKDIR /workspace

# 配置 rootless 模式
RUN mkdir -p /root/.config/containers && \
    echo '[storage]' > /root/.config/containers/storage.conf && \
    echo 'driver = "overlay"' >> /root/.config/containers/storage.conf && \
    echo '[storage.options]' >> /root/.config/containers/storage.conf && \
    echo 'mount_program = "/usr/bin/fuse-overlayfs"' >> /root/.config/containers/storage.conf

# 默认命令：保持容器运行
ENTRYPOINT ["/bin/sh", "-c", "tail -f /dev/null"]

