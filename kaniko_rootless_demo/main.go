package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	// 配置参数（参考 crane_demo）
	baseImage := "registry.kube-system.svc.cluster.local:5000/ones/plugin-host-node:v6.33.1"
	mainFilePath := "/workspace/server/main"
	newImageName := "registry.kube-system.svc.cluster.local:5000/new-kaniko-image:latest"

	// Kaniko executor 路径（如果在容器内运行，使用 /kaniko/executor）
	// 如果在本地运行且已安装 Kaniko，可以使用系统路径
	kanikoExecutor := getKanikoExecutor()

	fmt.Println("=== 使用 Kaniko 在程序内构建镜像 ===")
	fmt.Printf("基础镜像: %s\n", baseImage)
	fmt.Printf("源文件: %s\n", mainFilePath)
	fmt.Printf("目标镜像: %s\n", newImageName)

	// 构建新镜像
	if err := buildImageWithKaniko(baseImage, mainFilePath, newImageName, kanikoExecutor); err != nil {
		log.Fatalf("构建镜像失败: %v", err)
	}

	fmt.Printf("✓ 镜像构建并推送成功: %s\n", newImageName)
}

// 获取 Kaniko executor 路径
func getKanikoExecutor() string {
	// 优先使用环境变量
	if executor := os.Getenv("KANIKO_EXECUTOR"); executor != "" {
		return executor
	}
	// 默认路径（在 Kaniko 容器内）
	return "/kaniko/executor"
}

// 使用 Kaniko 构建镜像（参考 crane_demo 的镜像内容）
func buildImageWithKaniko(baseImage, mainFilePath, newImageName, kanikoExecutor string) error {
	// 检查 main 文件是否存在
	if _, err := os.Stat(mainFilePath); err != nil {
		return fmt.Errorf("main 文件不存在: %s, %w", mainFilePath, err)
	}

	// 检查 Kaniko executor 是否存在
	if _, err := os.Stat(kanikoExecutor); err != nil {
		return fmt.Errorf("Kaniko executor 不存在: %s, %w\n提示: 如果在本地运行，请安装 Kaniko 或使用 Kaniko 容器", kanikoExecutor, err)
	}

	// 1. 创建临时工作目录
	workDir := "/tmp/kaniko-build"
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer os.RemoveAll(workDir)

	// 2. 创建构建上下文目录
	contextDir := filepath.Join(workDir, "build-context")
	if err := os.MkdirAll(contextDir, 0755); err != nil {
		return fmt.Errorf("创建构建上下文目录失败: %w", err)
	}

	// 3. 创建 Dockerfile（参考 crane_demo：将 main 复制到 /usr/local/app/main，设置工作目录和入口点）
	dockerfileContent := fmt.Sprintf(`FROM %s
WORKDIR /usr/local/app
COPY main /usr/local/app/main
ENTRYPOINT ["/usr/local/app/main"]
`, baseImage)

	dockerfilePath := filepath.Join(contextDir, "Dockerfile")
	if err := os.WriteFile(dockerfilePath, []byte(dockerfileContent), 0644); err != nil {
		return fmt.Errorf("创建 Dockerfile 失败: %w", err)
	}
	fmt.Println("✓ Dockerfile 创建成功")

	// 4. 复制 main 文件到构建上下文
	contextMainPath := filepath.Join(contextDir, "main")
	if err := copyFile(mainFilePath, contextMainPath); err != nil {
		return fmt.Errorf("复制 main 文件失败: %w", err)
	}
	fmt.Printf("✓ 复制文件: %s -> %s\n", mainFilePath, contextMainPath)

	// 5. 调用 Kaniko executor 构建镜像
	fmt.Println("正在使用 Kaniko 构建镜像...")
	fmt.Printf("执行命令: %s --dockerfile %s --context %s --destination %s\n",
		kanikoExecutor, dockerfilePath, contextDir, newImageName)

	cmd := exec.Command(kanikoExecutor,
		"--dockerfile", dockerfilePath,
		"--context", contextDir,
		"--destination", newImageName,
		"--skip-tls-verify",      // 跳过 TLS 验证（用于私有 registry）
		"--skip-tls-verify-pull", // 拉取时跳过 TLS 验证
		"--insecure",             // 允许不安全的 registry
		"--verbosity=info",       // 日志级别
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Kaniko 构建失败: %w", err)
	}

	fmt.Println("✓ 镜像构建并推送成功")
	return nil
}

// 复制文件
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
