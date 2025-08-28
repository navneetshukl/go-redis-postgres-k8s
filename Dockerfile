# Build stage
FROM golang:1.22.3 AS builder

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./ 

RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# Final stage - Alpine version (even smaller)
FROM alpine:3.19

WORKDIR /app

# Install CA certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Copy the compiled binary from the builder stage
COPY --from=builder /app/main .

# Make the binary executable ← ADD THIS CRITICAL LINE
RUN chmod +x main

# Use a non-root user for security
RUN adduser -D -g '' appuser
USER appuser

# Use absolute path for reliability ← CHANGED THIS
CMD ["/app/main"]