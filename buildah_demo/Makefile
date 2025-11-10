.PHONY: build build-image run-docker run-k8s clean

# 构建 Go 程序
build:
	@echo "构建 Go 程序..."
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o main main.go
	@echo "✓ 构建完成: main"

# 构建包含 Buildah 的 Docker 镜像
build-image: build
	@echo "构建 Docker 镜像..."
	@docker build -t buildah-demo:latest .
	@echo "✓ Docker 镜像构建完成: buildah-demo:latest"

# 在 Docker 容器内运行
run-docker:
	@echo "在 Docker 容器内运行..."
	@docker run --rm \
		--device /dev/fuse \
		--security-opt seccomp=unconfined \
		--security-opt apparmor=unconfined \
		-v $(PWD)/../demo_server:/workspace/server:ro \
		-v $(PWD)/main:/workspace/main:ro \
		buildah-demo:latest \
		/workspace/main

# 在 K8s 中运行（作为 Pod）
run-k8s:
	@echo "在 K8s 中运行..."
	@kubectl apply -f buildah-pod.yaml
	@echo "等待 Pod 就绪..."
	@kubectl -n imgbuild wait --for=condition=Ready pod/buildah-demo --timeout=60s
	@echo "✓ Pod 已就绪，请手动复制文件并运行程序"

# 清理
clean:
	@rm -f main
	@echo "✓ 清理完成"

