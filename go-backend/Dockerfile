FROM golang:1.24-bookworm AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} env ${TARGETARCH:+GOARCH=${TARGETARCH}} go build -o /out/paneld ./cmd/paneld

FROM debian:bookworm-slim
WORKDIR /app

# 安装必要依赖和静态编译的 Docker CLI（仅客户端，无守护进程）
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    wget \
    && wget -q https://github.com/docker/cli/releases/download/v27.0.0/docker-27.0.0-linux-amd64.tgz \
    && tar -xzf docker-27.0.0-linux-amd64.tgz -C /usr/local/bin/ --strip-components=1 \
    && rm -rf docker-27.0.0-linux-amd64.tgz /var/lib/apt/lists/*

COPY --from=builder /out/paneld /app/paneld

ENV SERVER_ADDR=:6365
EXPOSE 6365
ENTRYPOINT ["/app/paneld"]
