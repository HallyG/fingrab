FROM golang:1.24.2-bookworm@sha256:00eccd446e023d3cd9566c25a6e6a02b90db3e1e0bbe26a48fc29cd96e800901 AS builder

ENV LANG=C.UTF-8

ARG TARGETOS
ARG TARGETARCH
ARG GO_LDFLAGS="-s -w -buildid="

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY main.go  main.go
COPY cmd/fingrab cmd/fingrab
COPY internal internal

RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o /app/fingrab -trimpath -mod=readonly -ldflags="${GO_LDFLAGS}" ./

# Runtime
FROM gcr.io/distroless/base-debian12@sha256:27769871031f67460f1545a52dfacead6d18a9f197db77110cfc649ca2a91f44

ENV LANG=C.UTF-8

ARG build_date=unknown
ARG commit_hash=unknown
ARG build_version=unknown

USER nonroot:nonroot
COPY --from=builder --chown=nonroot:nonroot /app/fingrab /opt/fingrab

LABEL org.opencontainers.image.authors="HallyG" \
      org.opencontainers.image.description="A CLI for exporting financial data from various banks." \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.revision=$commit_hash \
      org.opencontainers.image.created=$build_date \
      org.opencontainers.image.version=$build_version

ENTRYPOINT ["/opt/fingrab"]