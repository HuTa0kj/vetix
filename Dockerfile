# Vetix — one-shot SKILL security scanner.
#
# 单条静态二进制：提示词与 helper skill 通过 //go:embed 编进镜像，运行时只需要
# config.yaml（挂载进来）和待扫描目录。
FROM golang:1.25-bookworm AS build

WORKDIR /src

# 默认走 goproxy.cn 加速依赖拉取；其他网络环境用 --build-arg GOPROXY=... 覆盖。
ARG GOPROXY=https://goproxy.cn,direct
ARG GOSUMDB=off
ENV GOPROXY=${GOPROXY} GOSUMDB=${GOSUMDB} CGO_ENABLED=0

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /out/vetix ./cmd/vetix

FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY --from=build /out/vetix /usr/local/bin/vetix
# 只带模板，绝不把凭据打进镜像；忘记挂载 config.yaml 时会直接报错退出。
COPY example.config.yaml /work/example.config.yaml

WORKDIR /work
ENTRYPOINT ["vetix"]
# 安全的默认行为：打印 usage 并以 0 退出。
CMD ["-h"]
