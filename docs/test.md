1. 构建一个 基于 registry.cn-hangzhou.aliyuncs.com/kube-image-repo/kaniko:v1.9.1-debug  镜像的deployment,然后apply 起来 ，k8s 创建一个 对应的pod（build-image-pod）
2. 开发一个go 程序,代码你帮我放在 build_image 文件夹里面，开发并编译完成之后，copy 该程序和 server/main 到 build-image-pod中，并且运行起来这个go程序
- 这个程序将调用 /kaniko/executor ，构建一个新镜像（new-image），并推送到 registry.kube-system.svc.cluster.local:5000中。
- build image 时，把 目录下的 main 文件复制到镜像的 /usr/local/app 目录下。并设置工作目录为 /usr/local/app。并且entrypoint 为 /usr/local/app/main
- build image 时，用的基础镜像是  registry.kube-system.svc.cluster.local:5000/ones/plugin-host-node:v6.33.1
3. 创建一个 基于新镜像（new-image）的deployment,然后apply 起来 ，k8s 创建一个 对应的pod（test-kaniko-pod）
- 这里的镜像名，应该用localhost:5000/ones/plugin-host-node:v6.33.1 类似这样子，不然k8s无法拿到
4. 查看其test-kaniko-pod的日志，看控制台是否打印 Hello, World!