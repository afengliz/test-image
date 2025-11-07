package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

func main() {
	// 配置参数（参照 build_image/main.go）
	baseImage := "registry.kube-system.svc.cluster.local:5000/ones/plugin-host-node:v6.33.1"
	mainFilePath := "/workspace/server/main"
	imageName := "registry.kube-system.svc.cluster.local:5000/new-buildah-image:latest"

	fmt.Println("开始构建镜像...")

	// 构建镜像
	if err := buildImage(baseImage, mainFilePath, imageName); err != nil {
		log.Fatalf("构建镜像失败: %v", err)
	}

	fmt.Printf("✓ 镜像构建并推送成功: %s\n", imageName)
}

// 脚本式构建镜像（参照 build_image/main.go 的构建逻辑，使用 buildah 命令行）
func buildImage(baseImage, mainFilePath, imageName string) error {
	fmt.Printf("使用基础镜像: %s\n", baseImage)

	// 检查 main 文件是否存在
	if _, err := os.Stat(mainFilePath); err != nil {
		return fmt.Errorf("main 文件不存在: %s, %w", mainFilePath, err)
	}

	// 设置 buildah 环境变量
	os.Setenv("CONTAINERS_STORAGE_CONF", "/root/.config/containers/storage.conf")
	os.Setenv("CONTAINERS_CONF", "/root/.config/containers/containers.conf")

	// 使用 buildah bud 从 Dockerfile 构建（更简单可靠）
	// 1. 创建临时 Dockerfile
	dockerfileContent := fmt.Sprintf(`FROM %s
WORKDIR /usr/local/app
COPY main /usr/local/app/main
ENTRYPOINT ["/usr/local/app/main"]
`, baseImage)

	dockerfilePath := "/tmp/Dockerfile"
	if err := os.WriteFile(dockerfilePath, []byte(dockerfileContent), 0644); err != nil {
		return fmt.Errorf("创建 Dockerfile 失败: %w", err)
	}
	defer os.Remove(dockerfilePath)
	fmt.Println("✓ Dockerfile 创建成功")

	// 2. 创建构建上下文目录
	contextDir := "/tmp/build-context"
	if err := os.MkdirAll(contextDir, 0755); err != nil {
		return fmt.Errorf("创建构建上下文目录失败: %w", err)
	}
	defer os.RemoveAll(contextDir)

	// 3. 复制 main 文件到构建上下文
	contextMainPath := contextDir + "/main"
	if err := copyFile(mainFilePath, contextMainPath); err != nil {
		return fmt.Errorf("复制 main 文件失败: %w", err)
	}
	fmt.Printf("✓ 复制文件: %s -> %s\n", mainFilePath, contextMainPath)

	// 4. 复制 Dockerfile 到构建上下文
	contextDockerfilePath := contextDir + "/Dockerfile"
	if err := copyFile(dockerfilePath, contextDockerfilePath); err != nil {
		return fmt.Errorf("复制 Dockerfile 失败: %w", err)
	}

	// 5. 使用 buildah bud 构建镜像（rootless 模式，使用 --isolation chroot 避免 remount）
	fmt.Println("正在构建镜像...")
	// 使用 --isolation chroot 来避免需要 remount 权限
	cmd := exec.Command("buildah", "bud", "--tls-verify=false", "--isolation", "chroot", "-f", contextDockerfilePath, "-t", imageName, contextDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("构建镜像失败: %w", err)
	}

	fmt.Printf("✓ 镜像构建成功: %s\n", imageName)

	// 6. 推送镜像到 registry（buildah bud 不会自动推送）
	fmt.Println("正在推送镜像到 registry...")
	pushCmd := exec.Command("buildah", "push", "--tls-verify=false", imageName, "docker://"+imageName)
	pushCmd.Stdout = os.Stdout
	pushCmd.Stderr = os.Stderr
	if err := pushCmd.Run(); err != nil {
		return fmt.Errorf("推送镜像失败: %w", err)
	}

	fmt.Printf("✓ 镜像推送成功: %s\n", imageName)
	return nil
}

// 复制文件
func copyFile(src, dst string) error {
	// 确保目标目录存在
	if err := os.MkdirAll(dst[:len(dst)-len(dst[strings.LastIndex(dst, "/"):])], 0755); err != nil {
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
