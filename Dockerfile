FROM docker.io/library/golang:1.26 as builder

# Set up the build environment
WORKDIR /workdir
ADD config ./config/
ADD go.mod go.sum metrics.go main.go ./

# Build for Linux by default
ARG GOOS=linux
RUN go mod download
RUN CGO_ENABLED=0 GOOS=${GOOS} go build

# Package into a scratch image to minimise image size
FROM scratch

WORKDIR /app
COPY --from=builder /workdir/dns_exporter /app/dns_exporter
COPY examples/config.yml /app

EXPOSE 9117

ENV USER_ID=1001
USER ${USER_ID}

CMD ["/app/dns_exporter", "-config", "/app/config.yml"]
