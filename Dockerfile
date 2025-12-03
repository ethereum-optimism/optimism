# ------------------------------------------------------------
# Build Stage
# ------------------------------------------------------------
FROM golang:1.23 AS builder

# 安装 just
RUN apt-get update && apt-get install -y curl git pkg-config build-essential
RUN curl -fsSL https://just.systems/install.sh | bash -s -- --to /usr/local/bin

# 设置工作目录为 monorepo 根目录
WORKDIR /src

# 复制整个 monorepo
COPY . /src/

# 在根目录执行 just 构建 op-node
RUN just op-node

# ------------------------------------------------------------
# Runtime Stage
# ------------------------------------------------------------
FROM alpine:3.20

RUN apk add --no-cache ca-certificates

# 拷贝构建出的二进制
COPY --from=builder /src/op-node/op-node /usr/local/bin/op-node

ENTRYPOINT ["op-node"]
