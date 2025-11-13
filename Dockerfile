# 多阶段构建 - 第一阶段：构建
FROM golang:1.21-alpine AS builder

# 安装必要的构建工具
RUN apk add --no-cache git

# 设置工作目录
WORKDIR /build

# 复制 go.mod 和 go.sum（如果存在）
COPY go.mod go.sum* ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建二进制文件
# -ldflags="-s -w" 用于减小二进制文件大小
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o fl_limit .

# 多阶段构建 - 第二阶段：运行
FROM alpine:latest

# 安装必要的运行时依赖
RUN apk add --no-cache ca-certificates tzdata && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone

# 创建非 root 用户
RUN addgroup -g 1000 -S app && \
    adduser -u 1000 -S app -G app

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /build/fl_limit /app/fl_limit

# 复制示例配置文件
COPY config.yaml /app/config.yaml.example

# 修改文件权限
RUN chmod +x /app/fl_limit && \
    chown -R app:app /app

# 切换到非 root 用户
USER app

# 暴露端口（根据配置文件调整）
EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/healthz || exit 1

# 运行程序
ENTRYPOINT ["/app/fl_limit"]
CMD ["-config", "/app/config.yaml"]