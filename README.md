# FL Limit - 订阅流量限制代理

一个基于 Go 的反向代理服务，用于限制订阅链接的更新次数。

## 功能特性

- 基于 token 的请求限制
- 可配置的时间窗口和最大请求次数
- 反向代理到上游服务
- 健康检查端点
- Linux x86_64 支持

## GitHub Actions 自动构建和发布

本项目配置了两个GitHub Actions工作流：

### 1. 自动构建测试（每次提交）
- 文件：`.github/workflows/build.yml`
- 触发条件：推送到master/main分支或PR
- 功能：构建测试，不创建Release

### 2. 版本发布（创建tag时）
- 文件：`.github/workflows/release.yml`
- 触发条件：推送版本tag（如v1.0.0）
- 功能：构建、创建Release、上传二进制文件

### 如何触发自动发布

1. **创建并推送 tag**：

```bash
# 创建一个新的版本 tag（必须以 v 开头）
git tag v1.0.0

# 推送 tag 到 GitHub
git push origin v1.0.0
```

2. **GitHub Actions 会自动**：
   - 构建 Linux amd64 二进制文件
   - 创建压缩包
   - 生成 SHA256 校验和
   - 创建 GitHub Release
   - 上传构建产物

3. **查看发布结果**：
   - 访问项目的 Releases 页面查看发布的版本
   - 下载 Linux amd64 二进制文件

## 本地构建

如果需要在本地构建测试：

```bash
# 使用提供的构建脚本
./build.sh

# 或手动构建
GOOS=linux GOARCH=amd64 go build -o fl_limit-linux-amd64 .
```

## 配置文件

创建 `config.yaml` 文件：

```yaml
server:
  listen: ":8080"                 # 监听地址和端口

upstream:
  url: "http://your-backend.com"  # 上游服务地址

path:
  short_prefix: "/s/"             # 订阅链接前缀

limit:
  max: 10                         # 最大请求次数
  window: "24h"                   # 时间窗口
```

## 运行

### 二进制文件运行

```bash
# 下载对应架构的文件
wget https://github.com/YOUR_USERNAME/YOUR_REPO/releases/download/v1.0.0/fl_limit-linux-amd64.tar.gz

# 解压
tar -xzf fl_limit-linux-amd64.tar.gz

# 添加执行权限
chmod +x fl_limit-linux-amd64

# 运行
./fl_limit-linux-amd64 -config config.yaml
```

### Docker 运行

```bash
# 使用 GitHub Container Registry
docker run -d \
  --name fl_limit \
  -p 8080:8080 \
  -v $(pwd)/config.yaml:/app/config.yaml \
  ghcr.io/YOUR_USERNAME/YOUR_REPO:latest
```

## 健康检查

服务提供健康检查端点：

```bash
curl http://localhost:8080/healthz
```

## 开发

### 依赖

- Go 1.21+
- gopkg.in/yaml.v3

### 测试

```bash
go test ./...
```

## License

[添加你的许可证信息]