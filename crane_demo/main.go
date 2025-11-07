package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
)

func main() {
	// 配置参数
	baseImage := "registry.kube-system.svc.cluster.local:5000/ones/plugin-host-node:v6.33.1"
	mainFilePath := "/workspace/server/main"
	newImageName := "registry.kube-system.svc.cluster.local:5000/new-crane-image:latest"

	fmt.Println("=== 使用 Crane 在现有镜像上叠加文件 ===")

	// 构建新镜像
	if err := buildImageWithCrane(baseImage, mainFilePath, newImageName); err != nil {
		log.Fatalf("构建镜像失败: %v", err)
	}

	fmt.Printf("✓ 镜像构建并推送成功: %s\n", newImageName)
}

// 使用 Crane 在现有镜像上叠加文件
func buildImageWithCrane(baseImage, mainFilePath, newImageName string) error {
	fmt.Printf("使用基础镜像: %s\n", baseImage)

	// 检查 main 文件是否存在
	if _, err := os.Stat(mainFilePath); err != nil {
		return fmt.Errorf("main 文件不存在: %s, %w", mainFilePath, err)
	}

	// 1. 创建临时目录
	tempDir := "/tmp/crane-build"
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// 2. 创建目标目录结构（模拟镜像内的目录结构）
	targetDir := filepath.Join(tempDir, "usr", "local", "app")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("创建目标目录失败: %w", err)
	}

	// 3. 复制 main 文件到目标目录
	targetMainPath := filepath.Join(targetDir, "main")
	if err := copyFile(mainFilePath, targetMainPath); err != nil {
		return fmt.Errorf("复制 main 文件失败: %w", err)
	}
	fmt.Printf("✓ 复制文件: %s -> %s\n", mainFilePath, targetMainPath)

	// 4. 创建 tarball（包含要叠加的文件）
	tarballPath := filepath.Join(tempDir, "layer.tar")
	if err := createTarball(tempDir, tarballPath); err != nil {
		return fmt.Errorf("创建 tarball 失败: %w", err)
	}
	fmt.Println("✓ Tarball 创建成功")

	// 5. 使用 crane append 追加文件层到基础镜像
	fmt.Println("正在使用 crane append 叠加文件层...")

	// 解析镜像引用
	baseRef, err := name.ParseReference(baseImage)
	if err != nil {
		return fmt.Errorf("解析基础镜像失败: %w", err)
	}

	newRef, err := name.ParseReference(newImageName)
	if err != nil {
		return fmt.Errorf("解析新镜像名称失败: %w", err)
	}

	// 拉取基础镜像
	fmt.Printf("正在拉取基础镜像: %s\n", baseImage)
	baseImg, err := crane.Pull(baseRef.String())
	if err != nil {
		return fmt.Errorf("拉取基础镜像失败: %w", err)
	}

	// 追加文件层
	fmt.Println("正在追加文件层...")
	newImg, err := crane.Append(baseImg, tarballPath)
	if err != nil {
		return fmt.Errorf("追加文件层失败: %w", err)
	}

	// 6. 使用 crane mutate 修改镜像配置（设置入口点和工作目录）
	fmt.Println("正在修改镜像配置...")

	// 获取镜像配置
	configFile, err := newImg.ConfigFile()
	if err != nil {
		return fmt.Errorf("获取镜像配置失败: %w", err)
	}

	// 修改配置
	configFile.Config.WorkingDir = "/usr/local/app"
	configFile.Config.Entrypoint = []string{"/usr/local/app/main"}

	// 应用配置修改
	newImg, err = mutate.ConfigFile(newImg, configFile)
	if err != nil {
		return fmt.Errorf("修改镜像配置失败: %w", err)
	}

	// 7. 推送新镜像
	fmt.Printf("正在推送镜像到: %s\n", newImageName)
	if err := crane.Push(newImg, newRef.String()); err != nil {
		return fmt.Errorf("推送镜像失败: %w", err)
	}

	fmt.Printf("✓ 镜像推送成功: %s\n", newImageName)
	return nil
}

// 创建 tarball
func createTarball(sourceDir, tarballPath string) error {
	// 使用 tar 命令创建 tarball
	// 注意：需要保留目录结构，从 sourceDir 的父目录开始打包
	cmd := exec.Command("tar", "-czf", tarballPath, "-C", sourceDir, ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
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
