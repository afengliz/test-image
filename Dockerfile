FROM golang:1.20-alpine AS builder

WORKDIR /app

# 复制 go mod 文件
COPY server/go.mod server/go.sum ./
RUN go mod download

# 复制源代码
COPY server/main.go ./

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/main main.go

# 最终镜像
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/main .
EXPOSE 8081
CMD ["./main"]

