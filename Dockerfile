FROM alpine:3.18.5

 RUN apk update && apk --no-cache add ca-certificates && \
  update-ca-certificates

 ADD ./operational-load-exporter /usr/local/bin/operational-load-exporter
ENTRYPOINT ["/usr/local/bin/operational-load-exporter"]
