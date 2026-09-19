FROM golang:1.25-bookworm AS build

WORKDIR /src

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
COPY example.config.yaml /work/example.config.yaml

WORKDIR /work
ENTRYPOINT ["vetix"]

CMD ["-h"]
