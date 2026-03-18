# Build binary
FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.24 AS builder
ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="-X main.Commit=$(git rev-parse HEAD)" -a -o manager cmd/main.go

# Build distroless container from binary
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /src/manager /
ENTRYPOINT ["/manager"]
