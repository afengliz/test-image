package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	// 配置参数（参照 build_image/main.go）
	kanikoExecutor := "/kaniko/executor"
	dockerfilePath := "/workspace/Dockerfile"
	contextDir := "/workspace/build-context"
	mainFilePath := "/workspace/server/main"
	imageName := "registry.kube-system.svc.cluster.local:5000/new-kaniko-rootless-image:latest"

	fmt.Println("=== Kaniko 非特权模式构建镜像 ===")
	fmt.Println("非特权模式：无需 privileged: true，提高安全性")

	// 1. 创建构建上下文目录
	if err := os.MkdirAll(contextDir, 0755); err != nil {
		fmt.Printf("创建构建上下文目录失败: %v\n", err)
		os.Exit(1)
	}

	// 2. 创建 Dockerfile
	dockerfileContent := `FROM registry.kube-system.svc.cluster.local:5000/ones/plugin-host-node:v6.33.1
WORKDIR /usr/local/app
COPY main /usr/local/app/main
ENTRYPOINT ["/usr/local/app/main"]
`
	if err := os.WriteFile(dockerfilePath, []byte(dockerfileContent), 0644); err != nil {
		fmt.Printf("创建 Dockerfile 失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Dockerfile 创建成功")

	// 3. 复制 main 文件到构建上下文
	contextMainPath := filepath.Join(contextDir, "main")
	if err := copyFile(mainFilePath, contextMainPath); err != nil {
		fmt.Printf("复制 main 文件失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ main 文件复制成功")

	// 4. 复制 Dockerfile 到构建上下文
	contextDockerfilePath := filepath.Join(contextDir, "Dockerfile")
	if err := copyFile(dockerfilePath, contextDockerfilePath); err != nil {
		fmt.Printf("复制 Dockerfile 到构建上下文失败: %v\n", err)
		os.Exit(1)
	}

	// 5. 调用 kaniko executor 构建镜像（非特权模式）
	fmt.Printf("调用 kaniko executor 构建镜像（非特权模式）: %s\n", imageName)
	cmd := exec.Command(kanikoExecutor,
		"--dockerfile", contextDockerfilePath,
		"--context", "dir://"+contextDir, // 使用 dir:// 前缀
		"--destination", imageName,
		"--insecure",
		"--skip-tls-verify",
		// 非特权模式可能需要这些参数
		"--single-snapshot", // 使用单层快照，减少权限需求
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("构建镜像失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ 镜像构建并推送成功: %s\n", imageName)
}

func copyFile(src, dst string) error {
	// 确保目标目录存在
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	// 读取源文件
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	// 写入目标文件
	return os.WriteFile(dst, data, 0755)
}

