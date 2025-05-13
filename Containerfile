# Build stage
FROM golang:1.20-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata build-base

# Copy go.mod and go.sum files first to leverage Docker caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the application with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-w -s" -o /go/bin/tough-streets ./cmd/server

# Runtime stage
FROM alpine:3.18

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata libcap && \
    # Add capability to bind to privileged ports as non-root
    setcap 'cap_net_bind_service=+ep' /app/tough-streets

# Import timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy the binary from the builder stage
COPY --from=builder /go/bin/tough-streets /app/tough-streets

# Copy configuration files
COPY config/ /app/config/

# Create a non-root user to run the application
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

# Expose necessary ports
EXPOSE 9090/tcp
EXPOSE 53/tcp
EXPOSE 53/udp

# Start the application with a production configuration
ENTRYPOINT ["/app/tough-streets"]
CMD ["--config", "/app/config/production.yaml"]

# Include labels for OpenShift compatibility
LABEL io.k8s.description="Network discovery and DHCP/DNS service" \
      io.k8s.display-name="Tough Streets" \
      io.openshift.tags="networking,dhcp,dns" \
      io.openshift.non-scalable="true" \
      io.openshift.min-memory="512Mi" \
      io.openshift.min-cpu="100m"
