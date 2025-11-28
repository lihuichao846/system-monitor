# ===== 构建阶段 =====
FROM golang:1.23.5-alpine AS builder

WORKDIR /app

# 复制 go.mod/go.sum 并预下载依赖
COPY go.mod go.sum ./
RUN go mod download

# 复制剩余后端代码
COPY . .

# 构建可执行文件
RUN go build -o system-monitor-backend ./main.go

# ===== 运行阶段 =====
FROM alpine:3.19

WORKDIR /app

# 安装基础 CA 证书（如果未来会访问 https 外部服务）
RUN apk add --no-cache ca-certificates

# 从构建阶段复制二进制
COPY --from=builder /app/system-monitor-backend /app/system-monitor-backend

# 后端监听 8080
EXPOSE 8080

CMD ["/app/system-monitor-backend"]