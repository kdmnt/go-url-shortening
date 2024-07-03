# Stage 1: Build the Go app
FROM golang:1.22-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./

RUN go mod download

COPY . .

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Stage 2: Setup the runtime environment for the main application
FROM alpine:latest AS app

# Create a non-root user and group
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder --chown=appuser:appgroup /build/main /app/main

# Switch to the non-root user
USER appuser

EXPOSE 3000

CMD ["./main"]

# Stage 3: Setup the environment for k6 tests
FROM grafana/k6:0.52.0 AS k6

# Copy k6 test scripts
COPY tests/performance /scripts

# Change ownership of the scripts directory to the k6 user
USER root
RUN apk add --no-cache bash && \
    chown -R k6:k6 /scripts

# Make all run_test*.sh files executable
RUN chmod +x /scripts/run_test*.sh

# Switch back to the k6 user
USER k6

# Set the entrypoint to bash
ENTRYPOINT ["/bin/bash"]

# Default command (can be overridden)
CMD ["/scripts/run_tests.sh"]

