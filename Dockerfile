FROM alpine:3.19.1

 RUN apk update && apk --no-cache add ca-certificates && \
  update-ca-certificates

 ADD ./operational-load-exporter /usr/local/bin/operational-load-exporter
ENTRYPOINT ["/usr/local/bin/operational-load-exporter"]
