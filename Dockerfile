# syntax=docker/dockerfile:1

# ---- build ----
# BUILDPLATFORM でビルドホスト側のネイティブ Go を使い、TARGET* でクロスコンパイル。
# これで arm64(RPi5) 向けを amd64 の CI 上でも高速にビルドできる。
FROM --platform=$BUILDPLATFORM golang:1.27.3 AS build

ARG TARGETOS
ARG TARGETARCH

WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /out/serve ./cmd/serve

# ---- runtime ----
# distroless static (multi-arch, nonroot)。CGO 無効の静的バイナリなのでこれで足りる。
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/serve /serve

EXPOSE 50051
ENTRYPOINT ["/serve"]
