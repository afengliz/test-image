.PHONY: build run clean

# 构建 Go 程序
build:
	@echo "构建 Go 程序..."
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o main main.go
	@echo "✓ 构建完成: main"

# 构建包含 Kaniko 的 Docker 镜像
build-image: build
	@echo "准备构建上下文..."
	@cp -f ../demo_server/main server-main 2>/dev/null || (echo "警告: ../demo_server/main 不存在，请先构建" && exit 1)
	@echo "构建 Docker 镜像..."
	@docker build -t kaniko-build:latest .
	@echo "✓ Docker 镜像构建完成: kaniko-build:latest"
	@echo "提示: 镜像已包含 server-main 文件，无需运行时挂载"

# 在本地运行（需要本地安装 Kaniko 或使用 Docker）
run-local:
	@echo "在本地运行（需要 Kaniko executor）..."
	@go run main.go

# 在 Docker 容器内运行
run-docker:
	@echo "在 Docker 容器内运行..."
	@docker run --rm \
		-v $(PWD)/../server:/workspace/server:ro \
		-v $(PWD)/main:/kaniko-build/main:ro \
		--network host \
		kaniko-build:latest

# 在 K8s 中运行（作为 Pod）
run-k8s:
	@echo "在 K8s 中运行..."
	@kubectl run kaniko-build-$$(date +%s) \
		--image=kaniko-build:latest \
		--restart=Never \
		--rm -i \
		--overrides='{"spec":{"containers":[{"name":"kaniko-build","volumeMounts":[{"name":"server","mountPath":"/workspace/server","readOnly":true}],"securityContext":{"allowPrivilegeEscalation":false}}],"volumes":[{"name":"server","hostPath":{"path":"$(PWD)/../server"}}]}}'

# 清理
clean:
	@rm -f main server-main
	@echo "✓ 清理完成"

