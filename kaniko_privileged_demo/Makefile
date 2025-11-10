.PHONY: build run clean

# 构建 Go 程序
build:
	@echo "构建 Go 程序..."
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build-image main.go
	@echo "✓ 构建完成: build-image"

# 在 Docker 容器内运行
run-docker:
	@echo "在 Docker 容器内运行（Privileged 模式）..."
	@docker run --rm \
		--privileged \
		-v $(PWD)/../demo_server:/workspace/server:ro \
		-v $(PWD)/build-image:/workspace/build-image:ro \
		registry.cn-hangzhou.aliyuncs.com/kube-image-repo/kaniko:v1.9.1-debug \
		/workspace/build-image

# 在 K8s 中运行（作为 Pod）
run-k8s:
	@echo "在 K8s 中运行（Privileged 模式）..."
	@kubectl apply -f kaniko-pod.yaml
	@echo "等待 Pod 就绪..."
	@kubectl -n imgbuild wait --for=condition=Ready pod/kaniko-privileged --timeout=60s
	@echo "✓ Pod 已就绪，请手动复制文件并运行程序"

# 清理
clean:
	@rm -f build-image
	@echo "✓ 清理完成"

