package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
)

// 基础镜像缓存（优化频繁构建）
var (
	baseImageCache = make(map[string]v1.Image)
	cacheMutex     sync.RWMutex
)

func main() {
	// 配置参数
	baseImage := "registry.kube-system.svc.cluster.local:5000/ones/plugin-host-node:v6.33.1"
	mainFilePath := "/workspace/server/main"
	newImageName := "registry.kube-system.svc.cluster.local:5000/new-crane-image:latest"

	fmt.Println("=== 使用 Crane 在现有镜像上叠加文件（优化版）===")

	// 构建新镜像（使用缓存优化）
	if err := buildImageWithCraneOptimized(baseImage, mainFilePath, newImageName); err != nil {
		log.Fatalf("构建镜像失败: %v", err)
	}

	fmt.Printf("✓ 镜像构建并推送成功: %s\n", newImageName)
}

// 优化版本：使用基础镜像缓存
func buildImageWithCraneOptimized(baseImage, mainFilePath, newImageName string) error {
	fmt.Printf("使用基础镜像: %s\n", baseImage)

	// 检查 main 文件是否存在
	if _, err := os.Stat(mainFilePath); err != nil {
		return fmt.Errorf("main 文件不存在: %s, %w", mainFilePath, err)
	}

	// 1. 获取或拉取基础镜像（使用缓存）
	baseImg, err := getOrPullBaseImage(baseImage)
	if err != nil {
		return fmt.Errorf("获取基础镜像失败: %w", err)
	}

	// 2. 创建临时目录
	tempDir := "/tmp/crane-build"
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// 3. 创建目标目录结构
	targetDir := filepath.Join(tempDir, "usr", "local", "app")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("创建目标目录失败: %w", err)
	}

	// 4. 复制 main 文件到目标目录
	targetMainPath := filepath.Join(targetDir, "main")
	if err := copyFile(mainFilePath, targetMainPath); err != nil {
		return fmt.Errorf("复制 main 文件失败: %w", err)
	}
	fmt.Printf("✓ 复制文件: %s -> %s\n", mainFilePath, targetMainPath)

	// 5. 创建 tarball
	tarballPath := filepath.Join(tempDir, "layer.tar")
	if err := createTarball(tempDir, tarballPath); err != nil {
		return fmt.Errorf("创建 tarball 失败: %w", err)
	}
	fmt.Println("✓ Tarball 创建成功")

	// 6. 追加文件层
	fmt.Println("正在追加文件层...")
	newImg, err := crane.Append(baseImg, tarballPath)
	if err != nil {
		return fmt.Errorf("追加文件层失败: %w", err)
	}

	// 7. 修改镜像配置
	fmt.Println("正在修改镜像配置...")
	configFile, err := newImg.ConfigFile()
	if err != nil {
		return fmt.Errorf("获取镜像配置失败: %w", err)
	}

	configFile.Config.WorkingDir = "/usr/local/app"
	configFile.Config.Entrypoint = []string{"/usr/local/app/main"}

	newImg, err = mutate.ConfigFile(newImg, configFile)
	if err != nil {
		return fmt.Errorf("修改镜像配置失败: %w", err)
	}

	// 8. 推送新镜像
	fmt.Printf("正在推送镜像到: %s\n", newImageName)
	newRef, err := name.ParseReference(newImageName)
	if err != nil {
		return fmt.Errorf("解析新镜像名称失败: %w", err)
	}

	if err := crane.Push(newImg, newRef.String()); err != nil {
		return fmt.Errorf("推送镜像失败: %w", err)
	}

	fmt.Printf("✓ 镜像推送成功: %s\n", newImageName)
	return nil
}

// 获取或拉取基础镜像（带缓存）
func getOrPullBaseImage(baseImage string) (v1.Image, error) {
	// 先检查缓存
	cacheMutex.RLock()
	if cachedImg, ok := baseImageCache[baseImage]; ok {
		cacheMutex.RUnlock()
		fmt.Println("✓ 使用缓存的基础镜像")
		return cachedImg, nil
	}
	cacheMutex.RUnlock()

	// 缓存未命中，拉取镜像
	fmt.Printf("正在拉取基础镜像: %s\n", baseImage)
	baseRef, err := name.ParseReference(baseImage)
	if err != nil {
		return nil, fmt.Errorf("解析基础镜像失败: %w", err)
	}

	baseImg, err := crane.Pull(baseRef.String())
	if err != nil {
		return nil, fmt.Errorf("拉取基础镜像失败: %w", err)
	}

	// 存入缓存
	cacheMutex.Lock()
	baseImageCache[baseImage] = baseImg
	cacheMutex.Unlock()

	fmt.Println("✓ 基础镜像已缓存")
	return baseImg, nil
}

// 创建 tarball
func createTarball(sourceDir, tarballPath string) error {
	// 使用 tar 命令创建 tarball
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
