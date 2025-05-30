FROM gsoci.azurecr.io/giantswarm/alpine:3.22.0

 RUN apk update && apk --no-cache add ca-certificates && \
  update-ca-certificates

 ADD ./operational-load-exporter /usr/local/bin/operational-load-exporter
ENTRYPOINT ["/usr/local/bin/operational-load-exporter"]
