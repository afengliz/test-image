package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/containers/buildah"
	"github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/storage"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
)

func main() {
	// 配置参数（参考 crane_demo 和 kaniko_demo）
	baseImage := "registry.kube-system.svc.cluster.local:5000/ones/plugin-host-node:v6.33.1"
	mainFilePath := "/workspace/server/main"
	newImageName := "registry.kube-system.svc.cluster.local:5000/new-buildah-image:latest"

	fmt.Println("=== 使用 Buildah Go SDK 构建镜像 ===")
	fmt.Printf("基础镜像: %s\n", baseImage)
	fmt.Printf("源文件: %s\n", mainFilePath)
	fmt.Printf("目标镜像: %s\n", newImageName)

	// 构建新镜像
	if err := buildImageWithBuildah(baseImage, mainFilePath, newImageName); err != nil {
		log.Fatalf("构建镜像失败: %v", err)
	}

	fmt.Printf("✓ 镜像构建并推送成功: %s\n", newImageName)
}

// 使用 Buildah Go SDK 构建镜像（参考 crane_demo 的镜像内容）
func buildImageWithBuildah(baseImage, mainFilePath, newImageName string) error {
	// 检查 main 文件是否存在
	if _, err := os.Stat(mainFilePath); err != nil {
		return fmt.Errorf("main 文件不存在: %s, %w", mainFilePath, err)
	}

	// 1. 创建临时工作目录
	workDir := "/tmp/buildah-build"
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

	// 5. 使用 Buildah 构建镜像
	fmt.Println("正在使用 Buildah 构建镜像...")

	// 创建 Buildah 选项
	options := buildah.BuilderOptions{
		FromImage: baseImage,
	}

	// 创建构建器
	ctx := context.Background()
	storeOptions, err := buildah.GetDefaultStoreOptions()
	if err != nil {
		return fmt.Errorf("获取存储选项失败: %w", err)
	}

	// 创建存储
	store, err := storage.GetStore(storeOptions)
	if err != nil {
		return fmt.Errorf("创建存储失败: %w", err)
	}

	builder, err := buildah.NewBuilder(ctx, storeOptions, "buildah-container", options)
	if err != nil {
		return fmt.Errorf("创建构建器失败: %w", err)
	}
	defer builder.Delete()

	// 设置工作目录
	if err := builder.SetWorkDir("/usr/local/app"); err != nil {
		return fmt.Errorf("设置工作目录失败: %w", err)
	}

	// 复制文件
	destPath := "/usr/local/app/main"
	if err := builder.Add(contextMainPath, false, buildah.AddAndCopyOptions{}, destPath); err != nil {
		return fmt.Errorf("添加文件失败: %w", err)
	}
	fmt.Printf("✓ 文件已添加到镜像: %s\n", destPath)

	// 设置入口点
	if err := builder.SetCmd([]string{"/usr/local/app/main"}); err != nil {
		return fmt.Errorf("设置入口点失败: %w", err)
	}

	// 提交镜像
	fmt.Println("正在提交镜像...")
	imageID, err := builder.Commit(ctx, newImageName, buildah.CommitOptions{})
	if err != nil {
		return fmt.Errorf("提交镜像失败: %w", err)
	}
	fmt.Printf("✓ 镜像已提交: %s\n", imageID)

	// 推送到 registry（使用 containers/image 库）
	fmt.Println("正在推送到 registry...")
	systemContext := &types.SystemContext{
		// 跳过 TLS 验证（用于私有 registry）
		DockerInsecureSkipTLSVerify: types.NewOptionalBool(true),
	}

	// 创建策略上下文（允许所有镜像）
	policyContext, err := signature.NewPolicyContext(&signature.Policy{
		Default: []signature.PolicyRequirement{signature.NewPRInsecureAcceptAnything()},
	})
	if err != nil {
		return fmt.Errorf("创建策略上下文失败: %w", err)
	}
	defer policyContext.Destroy()

	// 使用 containers/image 库推送镜像
	// 注意：这里需要配置认证信息，实际使用时需要从环境变量或配置文件中读取
	destRef, err := alltransports.ParseImageName("docker://" + newImageName)
	if err != nil {
		return fmt.Errorf("解析目标镜像名称失败: %w", err)
	}

	// 获取镜像引用（使用 storage.Transport）
	srcRef, err := storage.Transport.ParseStoreReference(store, imageID)
	if err != nil {
		return fmt.Errorf("解析源镜像引用失败: %w", err)
	}

	// 推送镜像
	if err := copy.Image(ctx, policyContext, destRef, srcRef, &copy.Options{
		SourceCtx:      systemContext,
		DestinationCtx: systemContext,
	}); err != nil {
		return fmt.Errorf("推送镜像失败: %w", err)
	}

	fmt.Printf("✓ 镜像推送成功: %s\n", newImageName)
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
