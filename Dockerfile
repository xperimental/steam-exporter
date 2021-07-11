FROM golang:1.16 AS builder

WORKDIR /build

COPY go.mod go.sum /build/
RUN go mod download
RUN go mod verify

COPY . /build/
RUN make

FROM busybox
LABEL maintainer="Robert Jacob <xperimental@solidproject.de>"
LABEL org.opencontainers.image.source="https://github.com/xperimental/steam-exporter"

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /build/steam-exporter /bin/steam-exporter

USER nobody
EXPOSE 9791

ENTRYPOINT ["/bin/steam-exporter"]
CMD ["-c", "/etc/steam-exporter.yml" ]
