package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
)

func main() {
	// 配置参数（参照 build_image/main.go）
	baseImage := "registry.kube-system.svc.cluster.local:5000/ones/plugin-host-node:v6.33.1"
	mainFilePath := "/workspace/server/main"
	imageName := "registry.kube-system.svc.cluster.local:5000/new-buildah-rootless-image:latest"

	fmt.Println("=== Buildah Rootless 模式构建镜像 ===")
	fmt.Println("Rootless 模式：无需 root 权限，使用用户命名空间")

	// 检查当前用户
	currentUser, err := user.Current()
	if err != nil {
		log.Printf("警告: 无法获取当前用户信息: %v", err)
	} else {
		fmt.Printf("当前用户: %s (UID: %s, GID: %s)\n", currentUser.Username, currentUser.Uid, currentUser.Gid)
	}

	// 构建镜像
	if err := buildImageRootless(baseImage, mainFilePath, imageName); err != nil {
		log.Fatalf("构建镜像失败: %v", err)
	}

	fmt.Printf("✓ 镜像构建并推送成功: %s\n", imageName)
}

// Rootless 模式构建镜像
// 使用 buildah unshare 来创建用户命名空间，无需 root 权限
func buildImageRootless(baseImage, mainFilePath, imageName string) error {
	fmt.Printf("使用基础镜像: %s\n", baseImage)

	// 检查 main 文件是否存在
	if _, err := os.Stat(mainFilePath); err != nil {
		return fmt.Errorf("main 文件不存在: %s, %w", mainFilePath, err)
	}

	// 获取用户主目录（用于 Rootless 配置）
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		// 如果 HOME 未设置，尝试使用 /tmp 作为工作目录
		homeDir = "/tmp"
		fmt.Printf("警告: HOME 环境变量未设置，使用 /tmp 作为工作目录\n")
	}

	// Rootless 模式的配置目录
	configDir := filepath.Join(homeDir, ".config", "containers")
	storageConfPath := filepath.Join(configDir, "storage.conf")
	containersConfPath := filepath.Join(configDir, "containers.conf")

	// 确保配置目录存在
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	// 配置 Rootless 存储（使用 vfs 驱动，不需要 remount 权限）
	if err := setupRootlessStorage(storageConfPath); err != nil {
		return fmt.Errorf("配置 Rootless 存储失败: %w", err)
	}
	fmt.Println("✓ Rootless 存储配置完成")

	// 配置 Rootless 容器设置
	if err := setupRootlessContainers(containersConfPath); err != nil {
		return fmt.Errorf("配置 Rootless 容器设置失败: %w", err)
	}
	fmt.Println("✓ Rootless 容器配置完成")

	// 设置 buildah 环境变量（Rootless 模式）
	os.Setenv("CONTAINERS_STORAGE_CONF", storageConfPath)
	os.Setenv("CONTAINERS_CONF", containersConfPath)
	// Rootless 模式使用用户目录存储
	os.Setenv("XDG_RUNTIME_DIR", filepath.Join(homeDir, ".local", "share", "containers"))

	// 创建临时工作目录（在用户可写的位置）
	workDir := filepath.Join(homeDir, ".local", "buildah-work")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return fmt.Errorf("创建工作目录失败: %w", err)
	}
	defer os.RemoveAll(workDir)

	// 1. 创建临时 Dockerfile
	dockerfileContent := fmt.Sprintf(`FROM %s
WORKDIR /usr/local/app
COPY main /usr/local/app/main
ENTRYPOINT ["/usr/local/app/main"]
`, baseImage)

	dockerfilePath := filepath.Join(workDir, "Dockerfile")
	if err := os.WriteFile(dockerfilePath, []byte(dockerfileContent), 0644); err != nil {
		return fmt.Errorf("创建 Dockerfile 失败: %w", err)
	}
	fmt.Println("✓ Dockerfile 创建成功")

	// 2. 创建构建上下文目录
	contextDir := filepath.Join(workDir, "build-context")
	if err := os.MkdirAll(contextDir, 0755); err != nil {
		return fmt.Errorf("创建构建上下文目录失败: %w", err)
	}

	// 3. 复制 main 文件到构建上下文
	contextMainPath := filepath.Join(contextDir, "main")
	if err := copyFile(mainFilePath, contextMainPath); err != nil {
		return fmt.Errorf("复制 main 文件失败: %w", err)
	}
	fmt.Printf("✓ 复制文件: %s -> %s\n", mainFilePath, contextMainPath)

	// 4. 复制 Dockerfile 到构建上下文
	contextDockerfilePath := filepath.Join(contextDir, "Dockerfile")
	if err := copyFile(dockerfilePath, contextDockerfilePath); err != nil {
		return fmt.Errorf("复制 Dockerfile 失败: %w", err)
	}

	// 5. 使用 buildah 构建镜像
	// 检测当前用户：如果是 root，直接使用 buildah bud；否则使用 buildah unshare
	currentUser, err := user.Current()
	isRoot := err == nil && currentUser.Uid == "0"

	if isRoot {
		// root 用户：直接使用 buildah bud（不需要 unshare）
		// 使用 --isolation chroot 来避免需要 remount 权限
		fmt.Println("正在使用 buildah 构建镜像（root 用户模式）...")
		buildCmd := exec.Command("buildah", "bud",
			"--tls-verify=false",
			"--storage-driver", "vfs", // 使用 vfs 驱动
			"--isolation", "chroot", // 使用 chroot 隔离，避免 remount
			"-f", contextDockerfilePath,
			"-t", imageName,
			contextDir,
		)
		buildCmd.Stdout = os.Stdout
		buildCmd.Stderr = os.Stderr
		buildCmd.Env = os.Environ()

		if err := buildCmd.Run(); err != nil {
			return fmt.Errorf("构建镜像失败: %w", err)
		}
	} else {
		// 非 root 用户：使用 buildah unshare 创建用户命名空间
		fmt.Println("正在使用 Rootless 模式构建镜像...")
		fmt.Println("提示: 使用 buildah unshare 创建用户命名空间")

		buildCmd := exec.Command("buildah", "unshare", "buildah", "bud",
			"--tls-verify=false",
			"--storage-driver", "vfs", // Rootless 模式使用 vfs 驱动，不需要 remount
			"-f", contextDockerfilePath,
			"-t", imageName,
			contextDir,
		)
		buildCmd.Stdout = os.Stdout
		buildCmd.Stderr = os.Stderr
		buildCmd.Env = os.Environ()

		if err := buildCmd.Run(); err != nil {
			return fmt.Errorf("Rootless 模式构建镜像失败: %w", err)
		}
	}
	fmt.Printf("✓ 镜像构建成功: %s\n", imageName)

	// 6. 推送镜像到 registry
	if isRoot {
		fmt.Println("正在推送镜像到 registry...")
		pushCmd := exec.Command("buildah", "push",
			"--tls-verify=false",
			imageName,
			"docker://"+imageName,
		)
		pushCmd.Stdout = os.Stdout
		pushCmd.Stderr = os.Stderr
		pushCmd.Env = os.Environ()

		if err := pushCmd.Run(); err != nil {
			return fmt.Errorf("推送镜像失败: %w", err)
		}
	} else {
		fmt.Println("正在使用 Rootless 模式推送镜像到 registry...")
		pushCmd := exec.Command("buildah", "unshare", "buildah", "push",
			"--tls-verify=false",
			imageName,
			"docker://"+imageName,
		)
		pushCmd.Stdout = os.Stdout
		pushCmd.Stderr = os.Stderr
		pushCmd.Env = os.Environ()

		if err := pushCmd.Run(); err != nil {
			return fmt.Errorf("Rootless 模式推送镜像失败: %w", err)
		}
	}

	fmt.Printf("✓ 镜像推送成功: %s\n", imageName)
	return nil
}

// 配置 Rootless 存储（使用 vfs 驱动）
func setupRootlessStorage(storageConfPath string) error {
	// 如果配置文件已存在，不覆盖（可能用户已经配置过）
	if _, err := os.Stat(storageConfPath); err == nil {
		fmt.Printf("存储配置文件已存在: %s，跳过创建\n", storageConfPath)
		return nil
	}

	// Rootless 模式使用 vfs 存储驱动
	// vfs 驱动不需要 remount 权限，适合 Rootless 模式
	storageConf := `[storage]
driver = "vfs"
runroot = "/run/user/1000/containers/storage"
graphroot = "$HOME/.local/share/containers/storage"

[storage.options]
mount_program = ""
mountopt = ""
`
	return os.WriteFile(storageConfPath, []byte(storageConf), 0644)
}

// 配置 Rootless 容器设置
func setupRootlessContainers(containersConfPath string) error {
	// 如果配置文件已存在，不覆盖
	if _, err := os.Stat(containersConfPath); err == nil {
		fmt.Printf("容器配置文件已存在: %s，跳过创建\n", containersConfPath)
		return nil
	}

	// Rootless 模式配置
	containersConf := `[containers]
netns = "none"
default_ulimits = []

[engine]
helper_binaries_dir = ["/usr/libexec/podman", "/usr/local/libexec/podman", "/usr/lib/podman", "/usr/local/lib/podman"]
`
	return os.WriteFile(containersConfPath, []byte(containersConf), 0644)
}

// 复制文件
func copyFile(src, dst string) error {
	// 确保目标目录存在
	dir := filepath.Dir(dst)
	if err := os.MkdirAll(dir, 0755); err != nil {
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
