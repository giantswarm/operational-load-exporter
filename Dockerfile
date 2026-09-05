FROM gsoci.azurecr.io/giantswarm/alpine:3.24.1

RUN apk update && apk --no-cache add ca-certificates && \
  update-ca-certificates

# architect/go-build emits one static binary per target platform
# (operational-load-exporter-linux-amd64, -linux-arm64) plus an unsuffixed copy
# of the linux/amd64 build. Copy the one matching buildx's TARGETARCH so the
# arm64 image gets an arm64 binary. For a local build, produce it first:
#   CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o operational-load-exporter-linux-amd64 .
ARG TARGETARCH
COPY ./operational-load-exporter-linux-${TARGETARCH} /usr/local/bin/operational-load-exporter
ENTRYPOINT ["/usr/local/bin/operational-load-exporter"]
